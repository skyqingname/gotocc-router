//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const advisorTestUsage = `{"input_tokens":1000,"output_tokens":100,"iterations":[
 {"type":"message","input_tokens":400,"output_tokens":20},
 {"type":"advisor_message","model":"advisor-priced","input_tokens":500,"output_tokens":50,"cache_read_input_tokens":100,"cache_creation_input_tokens":50,"cache_creation":{"ephemeral_5m_input_tokens":40,"ephemeral_1h_input_tokens":10}},
 {"type":"message","input_tokens":600,"output_tokens":80},
 {"type":"advisor_message","model":"advisor-priced","input_tokens":10,"output_tokens":1}]}`

const advisorTestContent = `[{"type":"server_tool_use","id":"srvtoolu_cc_ses_advisor","name":"advisor","input":{}},{"type":"advisor_tool_result","tool_use_id":"srvtoolu_cc_ses_advisor","content":{"type":"advisor_redacted_result","encrypted_content":"cc_ses_opaque"}},{"type":"text","text":"cc_ses_regular"}]`

func advisorTestResponse() string {
	return `{"id":"msg_advisor","type":"message","role":"assistant","model":"advisor-priced","content":` + advisorTestContent + `,"usage":` + advisorTestUsage + `}`
}

func TestClaudeAdvisorUsageParsing(t *testing.T) {
	expected := []ClaudeAdvisorUsage{
		{Index: 1, Model: "advisor-priced", Usage: ClaudeUsage{InputTokens: 500, OutputTokens: 50, CacheReadInputTokens: 100, CacheCreationInputTokens: 50, CacheCreation5mTokens: 40, CacheCreation1hTokens: 10}},
		{Index: 3, Model: "advisor-priced", Usage: ClaudeUsage{InputTokens: 10, OutputTokens: 1}},
	}
	for _, passthrough := range []bool{false, true} {
		usage := &ClaudeUsage{}
		parse := func(data string) {
			if passthrough {
				parseSSEUsagePassthrough(data, usage)
			} else {
				(&GatewayService{}).parseSSEUsage(data, usage)
			}
		}
		parse(`{"type":"message_start","message":{"usage":{"input_tokens":400}}}`)
		data := `{"type":"message_delta","usage":` + advisorTestUsage + `}`
		parse(data)
		parse(data) // The cumulative final snapshot must replace, not append.
		require.Equal(t, 1000, usage.InputTokens)
		require.Equal(t, 100, usage.OutputTokens)
		require.Equal(t, expected, usage.AdvisorIterations)
		parse(`{"type":"message_delta","usage":{"output_tokens":100}}`)
		require.Equal(t, expected, usage.AdvisorIterations)
	}
	usage := parseClaudeUsageFromResponseBody([]byte(advisorTestResponse()))
	require.Equal(t, expected, usage.AdvisorIterations)
	require.Equal(t, 1000, usage.InputTokens)
	require.Equal(t, 100, usage.OutputTokens)
	require.Nil(t, parseClaudeUsageFromResponseBody([]byte(`{"usage":{"input_tokens":1}}`)).AdvisorIterations)
}

func TestClaudeAdvisorNativeResponsePassthrough(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, passthrough := range []bool{false, true} {
			name := "json"
			if stream {
				name = "sse"
			}
			if passthrough {
				name += "_passthrough"
			}
			t.Run(name, func(t *testing.T) {
				gin.SetMode(gin.TestMode)
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
				svc := &GatewayService{cfg: &config.Config{}, rateLimitService: &RateLimitService{}}
				account := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}
				body := advisorTestResponse()
				if stream {
					body = "event: message_start\ndata: " + `{"type":"message_start","message":{"id":"msg_advisor","type":"message","role":"assistant","model":"advisor-priced","content":[],"usage":{"input_tokens":400}}}` + "\n\n"
					for index, block := range gjson.Parse(advisorTestContent).Array() {
						event, err := json.Marshal(map[string]any{"type": "content_block_start", "index": index, "content_block": json.RawMessage(block.Raw)})
						require.NoError(t, err)
						body += "event: content_block_start\ndata: " + string(event) + "\n\n"
					}
					var compactUsage bytes.Buffer
					require.NoError(t, json.Compact(&compactUsage, []byte(advisorTestUsage)))
					body += "event: message_delta\ndata: " + `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":` + compactUsage.String() + "}\n\n"
					body += "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
				}
				resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
				var usage *ClaudeUsage
				var err error
				if stream {
					var result *streamingResult
					if passthrough {
						result, err = svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, account, time.Now(), "advisor-priced")
					} else {
						result, err = svc.handleStreamingResponse(context.Background(), resp, c, account, time.Now(), "client-executor", "advisor-priced", false)
					}
					require.NoError(t, err)
					usage = result.usage
				} else if passthrough {
					usage, err = svc.handleNonStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, account)
				} else {
					usage, err = svc.handleNonStreamingResponse(context.Background(), resp, c, account, "client-executor", "advisor-priced")
				}
				require.NoError(t, err)
				require.Len(t, usage.AdvisorIterations, 2)
				require.Equal(t, "advisor-priced", usage.AdvisorIterations[0].Model)
				require.Contains(t, recorder.Body.String(), `"type":"server_tool_use"`)
				require.Contains(t, recorder.Body.String(), `"type":"advisor_tool_result"`)
				require.Contains(t, recorder.Body.String(), `"encrypted_content":"cc_ses_opaque"`)
				require.Contains(t, recorder.Body.String(), `"srvtoolu_cc_ses_advisor"`)
				if !passthrough {
					require.Contains(t, recorder.Body.String(), `"model":"client-executor"`)
				}
				if stream {
					for _, line := range strings.Split(recorder.Body.String(), "\n") {
						data, ok := extractAnthropicSSEDataLine(line)
						if ok && gjson.Get(data, "type").String() == "message_delta" {
							require.JSONEq(t, gjson.Get(advisorTestUsage, "iterations").Raw, gjson.Get(data, "usage.iterations").Raw)
						}
					}
				} else {
					require.JSONEq(t, gjson.Get(advisorTestUsage, "iterations").Raw, gjson.Get(recorder.Body.String(), "usage.iterations").Raw)
				}
			})
		}
	}
}

func TestClaudeAdvisorRequestPreservation(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-5","max_tokens":256,"tools":[{"type":"advisor_20260301","name":"advisor","model":"claude-opus-5-5","max_uses":1},{"name":"sessions_read","input_schema":{"type":"object"}}],"messages":[{"role":"user","content":"hello"}]}`)
	header := http.Header{"Anthropic-Beta": []string{claudeAdvisorBeta}}
	svc := &GatewayService{cfg: &config.Config{}}
	for _, mode := range []struct {
		kind  string
		mimic bool
	}{{"apikey", false}, {"oauth", false}, {"oauth", true}} {
		beta, set := svc.computeFinalAnthropicBeta(mode.kind, mode.mimic, "claude-sonnet-5", header, body, nil)
		require.True(t, set)
		require.True(t, containsBetaToken(beta, claudeAdvisorBeta))
	}
	rewritten := applyToolNameRewriteToBody(body, buildToolNameRewriteFromBody(body))
	require.Equal(t, "advisor", gjson.GetBytes(rewritten, "tools.0.name").String())
	require.Equal(t, "advisor_20260301", gjson.GetBytes(rewritten, "tools.0.type").String())
	require.Equal(t, "claude-opus-5-5", gjson.GetBytes(rewritten, "tools.0.model").String())
	require.EqualValues(t, 1, gjson.GetBytes(rewritten, "tools.0.max_uses").Int())
	require.Equal(t, "cc_sess_read", gjson.GetBytes(rewritten, "tools.1.name").String())
	vertex := filterVertexBetaTokens(claudeAdvisorBeta, nil, hasClaudeAdvisorTool(body))
	require.Equal(t, claudeAdvisorBeta, vertex)
	bedrock, err := PrepareBedrockRequestBody(body, "claude-sonnet-5", claudeAdvisorBeta)
	require.NoError(t, err)
	require.Equal(t, claudeAdvisorBeta, gjson.GetBytes(bedrock, "anthropic_beta.0").String())
	require.Equal(t, "advisor", gjson.GetBytes(bedrock, "tools.0.name").String())
	// Incidental beta headers without the tool retain cloud-provider behavior.
	require.Empty(t, filterVertexBetaTokens(claudeAdvisorBeta, nil))
	require.Empty(t, filterBedrockBetaTokens([]string{claudeAdvisorBeta}))
}

func TestClaudeAdvisorFullForwardPreservesWireDefinition(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		gin.SetMode(gin.TestMode)
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		body := []byte(`{"model":"claude-sonnet-5","max_tokens":512,"stream":false,"tools":[{"type":"advisor_20260301","name":"advisor","model":"claude-opus-5-5","max_uses":1}],"tool_choice":{"type":"tool","name":"advisor"},"messages":[{"role":"user","content":"synthetic advisor verification"}]}`)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
		c.Request.Header.Set("Anthropic-Beta", claudeAdvisorBeta)
		upstreamBody := strings.Replace(advisorTestResponse(), `"model":"advisor-priced"`, `"model":"claude-sonnet-5"`, 1)
		upstreamBody = strings.ReplaceAll(upstreamBody, `"model":"advisor-priced"`, `"model":"claude-opus-5-5"`)
		upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(upstreamBody))}}
		svc := &GatewayService{cfg: &config.Config{}, httpUpstream: upstream, rateLimitService: &RateLimitService{}, deferredService: &DeferredService{}}
		account := newAnthropicAPIKeyAccountForTest()
		account.Extra["anthropic_passthrough"] = passthrough
		parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-5"}
		result, err := svc.Forward(context.Background(), c, account, parsed)
		require.NoError(t, err)
		require.NotNil(t, upstream.lastReq)
		require.True(t, containsBetaToken(getHeaderRaw(upstream.lastReq.Header, "anthropic-beta"), claudeAdvisorBeta))
		for _, field := range []string{"type", "name", "model", "max_uses"} {
			require.Equal(t, gjson.GetBytes(body, "tools.0."+field).Raw, gjson.GetBytes(upstream.lastBody, "tools.0."+field).Raw)
		}
		require.Len(t, result.Usage.AdvisorIterations, 2)
		require.Equal(t, "claude-opus-5-5", result.Usage.AdvisorIterations[0].Model)
	}
}

func TestClaudeAdvisorMentionWithoutToolKeepsLegacyRestoration(t *testing.T) {
	body := []byte(`{"content":[{"type":"text","text":"advisor cc_ses_demo"}],"usage":{"iterations":[{"type":"message","model":"cc_ses_model"}]}}`)
	require.Equal(t, restoreToolNamesInBytes(body, nil), restoreToolNamesPreservingAdvisor(body, nil))
}

type advisorBillingLedger struct {
	UsageBillingRepository
	commands   map[string]*UsageBillingCommand
	balance    float64
	afterFirst func()
}

func (r *advisorBillingLedger) Apply(ctx context.Context, command *UsageBillingCommand) (*UsageBillingApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if previous := r.commands[command.RequestID]; previous != nil {
		if previous.RequestFingerprint != command.RequestFingerprint {
			return nil, ErrUsageBillingRequestConflict
		}
		return &UsageBillingApplyResult{Applied: false}, nil
	}
	r.commands[command.RequestID] = command
	r.balance -= command.BalanceCost
	if len(r.commands) == 1 && r.afterFirst != nil {
		r.afterFirst()
	}
	return &UsageBillingApplyResult{Applied: true, UsageLogPersisted: true}, nil
}

func advisorBillingFixture() (*GatewayService, *RecordUsageInput, *advisorBillingLedger) {
	ledger := &advisorBillingLedger{commands: map[string]*UsageBillingCommand{}, balance: 10}
	svc := newGatewayRecordUsageServiceWithBillingRepoForTest(&openAIRecordUsageLogRepoStub{}, ledger, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{})
	catalog := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"executor-test":  {InputCostPerToken: 2e-6, OutputCostPerToken: 3e-6},
		"advisor-priced": {InputCostPerToken: 4e-6, OutputCostPerToken: 6e-6, CacheReadInputTokenCost: 0.4e-6, CacheCreationInputTokenCost: 5e-6, CacheCreationInputTokenCostAbove1hr: 8e-6},
	}}
	svc.billingService = NewBillingService(svc.cfg, catalog)
	svc.resolver = NewModelPricingResolver(nil, svc.billingService)
	groupID := int64(7)
	user := &User{ID: 22, Balance: 10}
	key := &APIKey{ID: 11, UserID: user.ID, User: user, GroupID: &groupID, Group: &Group{ID: groupID, Platform: PlatformAnthropic, RateMultiplier: 2}}
	usage := parseClaudeUsageFromResponseBody([]byte(advisorTestResponse()))
	input := &RecordUsageInput{Result: &ForwardResult{RequestID: "upstream-parent", Usage: *usage, Model: "executor-test"}, APIKey: key, User: user, Account: &Account{ID: 33, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}, PricingAt: time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)}
	return svc, input, ledger
}

func TestClaudeAdvisorBillingSeparatesIterationsAndRetry(t *testing.T) {
	svc, input, ledger := advisorBillingFixture()
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "parent-turn")
	require.NoError(t, svc.RecordUsage(ctx, input))
	require.NoError(t, svc.RecordUsage(ctx, input))
	require.Len(t, ledger.commands, 3)
	main := ledger.commands["client:parent-turn"]
	first := ledger.commands["client:parent-turn:advisor:1"]
	second := ledger.commands["client:parent-turn:advisor:3"]
	require.NotNil(t, main)
	require.NotNil(t, first)
	require.NotNil(t, second)
	require.Equal(t, 1000, main.InputTokens)
	require.Equal(t, 100, main.OutputTokens)
	require.Equal(t, "advisor-priced", first.Model)
	require.Equal(t, "advisor-priced", first.UsageLog.RequestedModel)
	require.InDelta(t, 0.0046, main.BalanceCost, 1e-10)
	require.InDelta(t, 0.00524, first.BalanceCost, 1e-10)
	require.InDelta(t, 0.000092, second.BalanceCost, 1e-10)
	require.InDelta(t, 10-0.0046-0.00524-0.000092, ledger.balance, 1e-10)
	require.Equal(t, 2.0, first.UsageLog.RateMultiplier)
	require.Equal(t, main.APIKeyID, first.APIKeyID)
	require.Equal(t, main.UserID, first.UserID)
	require.Equal(t, main.AccountID, first.AccountID)
}

func TestClaudeAdvisorBillingMissingPriceUsesMainAndWarns(t *testing.T) {
	svc, input, ledger := advisorBillingFixture()
	input.Result.Usage.AdvisorIterations = []ClaudeAdvisorUsage{{Index: 1, Model: "unpriced-advisor", Usage: ClaudeUsage{InputTokens: 500, OutputTokens: 50}}}
	var warnings bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&warnings, nil)))
	defer slog.SetDefault(old)
	require.NoError(t, svc.RecordUsage(context.Background(), input))
	child := ledger.commands["upstream-parent:advisor:1"]
	require.NotNil(t, child)
	require.Equal(t, "unpriced-advisor", child.Model)
	require.InDelta(t, 0.0023, child.BalanceCost, 1e-10)
	require.Contains(t, warnings.String(), "claude_advisor_pricing_fallback")
	require.Contains(t, warnings.String(), "executor-test")
}

func TestClaudeAdvisorFallbackUsesMainSelectedResponsePrice(t *testing.T) {
	svc, input, ledger := advisorBillingFixture()
	svc.billingService.pricingService.pricingData["executor-cheap"] = &LiteLLMModelPricing{InputCostPerToken: 1e-6, OutputCostPerToken: 1e-6}
	input.BillingModelSource = "response_model"
	input.Result.UpstreamResponseModel = "executor-cheap"
	input.Result.Usage.AdvisorIterations = []ClaudeAdvisorUsage{{Index: 1, Model: "unpriced-advisor", Usage: ClaudeUsage{InputTokens: 500, OutputTokens: 50}}}
	require.NoError(t, svc.RecordUsage(context.Background(), input))
	require.InDelta(t, 0.0022, ledger.commands["upstream-parent"].BalanceCost, 1e-10)
	require.InDelta(t, 0.0011, ledger.commands["upstream-parent:advisor:1"].BalanceCost, 1e-10)
}

func TestClaudeAdvisorBillingKeepsFrozenPrices(t *testing.T) {
	svc, input, ledger := advisorBillingFixture()
	ctx := svc.WithBillingPricingSnapshot(context.Background(), input.APIKey, input.PricingAt, "executor-test")
	input.PricingSnapshot = BillingPricingSnapshotFromContext(ctx)
	ledger.afterFirst = func() {
		input.APIKey.Group.RateMultiplier = 9
		// Production publishes a new immutable catalog generation on reload.
		catalog := svc.billingService.pricingService
		catalog.mu.Lock()
		catalog.pricingData = map[string]*LiteLLMModelPricing{
			"executor-test":  {InputCostPerToken: 2e-6, OutputCostPerToken: 3e-6},
			"advisor-priced": {InputCostPerToken: 1, OutputCostPerToken: 1},
		}
		catalog.mu.Unlock()
	}
	require.NoError(t, svc.RecordUsage(ctx, input))
	require.InDelta(t, 0.00524, ledger.commands["upstream-parent:advisor:1"].BalanceCost, 1e-10)
	require.NoError(t, svc.RecordUsage(ctx, input))
	require.Len(t, ledger.commands, 3)
}

func TestClaudeAdvisorBillingRetainsTeamPayer(t *testing.T) {
	svc, input, ledger := advisorBillingFixture()
	teamID := int64(77)
	input.APIKey.TeamID = &teamID
	input.APIKey.TeamMembership = &TeamMembership{ID: 78}
	input.APIKey.ActorUser = &User{ID: 23}
	input.APIKey.UserID = 23
	require.NoError(t, svc.RecordUsage(context.Background(), input))
	for _, command := range ledger.commands {
		require.Equal(t, int64(22), command.UserID)
		require.Equal(t, int64(23), command.ActorUserID)
		require.Equal(t, &teamID, command.TeamID)
		require.Equal(t, int64(78), command.TeamMembershipID)
		require.True(t, command.TeamWallet)
	}
}

func TestClaudeAdvisorBillingMainMappingDoesNotPriceAdvisor(t *testing.T) {
	svc, input, ledger := advisorBillingFixture()
	input.Result.Model = "client-executor-alias"
	input.Result.UpstreamModel = "executor-test"
	input.ChannelUsageFields = ChannelUsageFields{OriginalModel: "client-executor-alias", ChannelMappedModel: "executor-test", BillingModelSource: BillingModelSourceChannelMapped}
	require.NoError(t, svc.RecordUsage(context.Background(), input))
	require.InDelta(t, 0.0046, ledger.commands["upstream-parent"].BalanceCost, 1e-10)
	require.InDelta(t, 0.00524, ledger.commands["upstream-parent:advisor:1"].BalanceCost, 1e-10)
	require.Equal(t, "advisor-priced", ledger.commands["upstream-parent:advisor:1"].UsageLog.Model)
}

func TestClaudeAdvisorBillingLongParentIDRemainsLinkedAndIdempotent(t *testing.T) {
	svc, input, ledger := advisorBillingFixture()
	parent := strings.Repeat("p", 64)
	input.Result.RequestID = parent
	require.NoError(t, svc.RecordUsage(context.Background(), input))
	require.NoError(t, svc.RecordUsage(context.Background(), input))
	require.Len(t, ledger.commands, 3)
	for id, command := range ledger.commands {
		require.LessOrEqual(t, len(id), 64)
		if id == parent {
			continue
		}
		metadata, ok := command.UsageLog.BillingRateSnapshot["advisor"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, parent, metadata["parent_request_id"])
		require.Equal(t, "advisor-priced", metadata["pricing_model"])
	}
}

func TestClaudeAdvisorAbsentAndDisconnectDoNotChangeMainBilling(t *testing.T) {
	for _, disconnected := range []bool{false, true} {
		svc, input, ledger := advisorBillingFixture()
		input.Result.Usage.AdvisorIterations = nil
		input.Result.ClientDisconnect = disconnected
		input.Result.Stream = disconnected
		ctx, cancel := context.WithCancel(context.Background())
		if disconnected {
			cancel()
		}
		defer cancel()
		require.NoError(t, svc.RecordUsage(ctx, input))
		require.Len(t, ledger.commands, 1)
		require.InDelta(t, 0.0046, ledger.commands["upstream-parent"].BalanceCost, 1e-10)
	}
}

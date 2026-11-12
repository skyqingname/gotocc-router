package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

const claudeAdvisorBeta = "advisor-tool-2026-03-01"

func hasClaudeAdvisorTool(body []byte) bool {
	for _, tool := range gjson.GetBytes(body, "tools").Array() {
		if tool.Get("type").String() == "advisor_20260301" && tool.Get("name").String() == "advisor" {
			return true
		}
	}
	return false
}

// Advisor result payloads and iteration models are provider-owned. Keep their
// raw bytes out of the legacy whole-response tool-name restoration.
func restoreToolNamesPreservingAdvisor(chunk []byte, rewrite *ToolNameRewrite) []byte {
	if !bytes.Contains(chunk, []byte("advisor")) {
		return restoreToolNamesInBytes(chunk, rewrite)
	}
	if gjson.ValidBytes(chunk) {
		return restoreNonAdvisorJSON(chunk, rewrite)
	}
	var output bytes.Buffer
	for _, line := range bytes.SplitAfter(chunk, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("data:")) {
			payload := bytes.TrimSpace(trimmed[len("data:"):])
			if gjson.ValidBytes(payload) {
				start := bytes.Index(line, payload)
				output.Write(line[:start])
				output.Write(restoreNonAdvisorJSON(payload, rewrite))
				output.Write(line[start+len(payload):])
				continue
			}
		}
		output.Write(restoreToolNamesInBytes(line, rewrite))
	}
	return output.Bytes()
}

func restoreNonAdvisorJSON(body []byte, rewrite *ToolNameRewrite) []byte {
	type span struct{ start, end int }
	var protected []span
	var visit func(gjson.Result)
	visit = func(value gjson.Result) {
		kind := value.Get("type").String()
		if kind == "advisor_tool_result" || kind == "advisor_message" ||
			(kind == "server_tool_use" && value.Get("name").String() == "advisor") {
			protected = append(protected, span{value.Index, value.Index + len(value.Raw)})
			return
		}
		if value.IsObject() || value.IsArray() {
			value.ForEach(func(key, child gjson.Result) bool {
				advisorIterations := false
				if key.String() == "iterations" && child.IsArray() {
					for _, iteration := range child.Array() {
						advisorIterations = advisorIterations || iteration.Get("type").String() == "advisor_message"
					}
				}
				if advisorIterations {
					protected = append(protected, span{child.Index, child.Index + len(child.Raw)})
				} else {
					visit(child)
				}
				return true
			})
		}
	}
	visit(gjson.ParseBytes(body))
	if len(protected) == 0 {
		return restoreToolNamesInBytes(body, rewrite)
	}
	sort.Slice(protected, func(i, j int) bool { return protected[i].start < protected[j].start })
	var output bytes.Buffer
	start := 0
	for _, item := range protected {
		output.Write(restoreToolNamesInBytes(body[start:item.start], rewrite))
		output.Write(body[item.start:item.end])
		start = item.end
	}
	output.Write(restoreToolNamesInBytes(body[start:], rewrite))
	return output.Bytes()
}

// ClaudeAdvisorUsage is separate from the executor's top-level usage totals.
// Index is the original iterations[] position, stable across repeated snapshots.
type ClaudeAdvisorUsage struct {
	Index int
	Model string
	Usage ClaudeUsage
}

func parseClaudeAdvisorUsage(value any) []ClaudeAdvisorUsage {
	items, _ := value.([]any)
	var advisors []ClaudeAdvisorUsage
	for index, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok || item["type"] != "advisor_message" {
			continue
		}
		model, _ := item["model"].(string)
		usage := ClaudeUsage{}
		usage.InputTokens, _ = parseSSEUsageInt(item["input_tokens"])
		usage.OutputTokens, _ = parseSSEUsageInt(item["output_tokens"])
		usage.CacheCreationInputTokens, _ = parseSSEUsageInt(item["cache_creation_input_tokens"])
		usage.CacheReadInputTokens, _ = parseSSEUsageInt(item["cache_read_input_tokens"])
		if cache, ok := item["cache_creation"].(map[string]any); ok {
			usage.CacheCreation5mTokens, _ = parseSSEUsageInt(cache["ephemeral_5m_input_tokens"])
			usage.CacheCreation1hTokens, _ = parseSSEUsageInt(cache["ephemeral_1h_input_tokens"])
		}
		advisors = append(advisors, ClaudeAdvisorUsage{Index: index, Model: model, Usage: usage})
	}
	return advisors
}

func claudeAdvisorRequestID(parent string, index int) string {
	requestID := fmt.Sprintf("%s:advisor:%d", parent, index)
	// usage_logs.request_id is VARCHAR(64). Keep a deterministic compact ID
	// when the suffix would overflow; the parent is also stored in the row.
	if len(requestID) > 64 {
		return "advisor:" + uuid.NewSHA1(uuid.NameSpaceOID, []byte(requestID)).String()
	}
	return requestID
}

func (s *GatewayService) recordUsageWithAdvisor(ctx context.Context, input *recordUsageCoreInput) error {
	// Pin the same catalog, group, payer and pricing instant for every charge.
	ctx = s.WithBillingPricingSnapshot(ctx, billingKeyWithUser(input.APIKey, input.User), input.PricingAt)
	input.requestID = resolveUsageBillingRequestID(ctx, input.Result.RequestID)
	if err := s.recordUsageCore(ctx, input); err != nil {
		return err
	}
	mainPricingModel := input.settledPricingModel
	for _, advisor := range input.Result.Usage.AdvisorIterations {
		child := *input
		child.requestID = claudeAdvisorRequestID(input.requestID, advisor.Index)
		child.advisorParentRequestID = input.requestID
		child.advisorIteration = advisor.Index
		child.ForceCacheBilling = false
		child.ChannelUsageFields = ChannelUsageFields{ChannelID: input.ChannelID, OriginalModel: advisor.Model}
		child.Result = &ForwardResult{
			RequestID: child.requestID, UpstreamHeaders: input.Result.UpstreamHeaders,
			Usage: advisor.Usage, Model: advisor.Model,
			UpstreamResponseModel: advisor.Model,
			Stream:                input.Result.Stream, Duration: input.Result.Duration,
			ClientDisconnect:            input.Result.ClientDisconnect,
			ClientDisconnectUsageSource: input.Result.ClientDisconnectUsageSource,
		}
		childCtx := claudeAdvisorPricingContext(ctx, input.Account.Platform, advisor.Model)
		child.advisorPricingModel = advisor.Model
		if identified, _ := s.hasIdentifiedResponseModelPricing(childCtx, advisor.Model, input.APIKey); !identified {
			child.advisorPricingModel = mainPricingModel
			slog.WarnContext(childCtx, "claude_advisor_pricing_fallback",
				"request_id", input.requestID, "advisor_request_id", child.requestID,
				"advisor_model", advisor.Model, "pricing_model", mainPricingModel,
				"account_id", input.Account.ID)
		}
		if err := s.recordUsageCore(childCtx, &child); err != nil {
			return fmt.Errorf("record advisor iteration %d: %w", advisor.Index, err)
		}
	}
	return nil
}

func claudeAdvisorPricingContext(ctx context.Context, platform, model string) context.Context {
	ctx = BindBillingPricingModel(ctx, platform, model)
	if snapshot := BillingPricingSnapshotFromContext(ctx); snapshot != nil {
		copy := *snapshot
		copy.routingModel = model
		ctx = pricingSnapshotContext(ctx, &copy)
	}
	return WithCompositeRouteDecision(ctx, CompositeRouteDecision{
		Matched: true, TargetPlatform: platform, PublicModel: model, UpstreamModel: model,
	})
}

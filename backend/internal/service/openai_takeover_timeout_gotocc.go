package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/accountbalance"
	"github.com/tidwall/gjson"
)

// A migration is accepted only by the existing protocol parser. These clocks
// measure transport progress while that acceptance is pending.
const OpenAITakeoverTimeoutCode = "session_takeover_timeout"

type takeoverProgressKey struct{}
type takeoverTimeout string

func (t takeoverTimeout) Error() string { return "session takeover timed out: " + string(t) }

type takeoverProgress struct {
	mu         sync.Mutex
	phase      string
	last       time.Time
	absolute   time.Time
	initial    time.Duration
	headerWait time.Duration
	changed    chan struct{}
	accepted   <-chan struct{}
}

func (p *takeoverProgress) advance(phase string, startOnly bool) {
	select {
	case <-p.accepted:
		return
	default:
	}
	p.mu.Lock()
	changed := p.phase != phase
	if changed || !startOnly {
		p.phase, p.last = phase, time.Now()
	}
	p.mu.Unlock()
	if changed {
		select {
		case p.changed <- struct{}{}:
		default:
		}
	}
}

func (p *takeoverProgress) next() (time.Duration, takeoverTimeout) {
	p.mu.Lock()
	defer p.mu.Unlock()
	window := p.initial
	switch p.phase {
	case "headers":
		window = p.headerWait
	case "body":
		window = accountbalance.TakeoverUpstreamIdleSeconds * time.Second
	}
	deadline := p.last.Add(window)
	if !p.absolute.After(deadline) {
		return time.Until(p.absolute), takeoverTimeout("total")
	}
	return time.Until(deadline), takeoverTimeout(p.phase)
}

// Track only a generation request carrying a pending takeover, never the
// downstream heartbeat. A response header moves the timer to the body stage;
// actual reads (including SSE comments) refresh the idle window.
func ObserveOpenAITakeoverHTTP(req *http.Request) func(*http.Response) {
	if req.Method != http.MethodPost {
		return nil
	}
	p, _ := req.Context().Value(takeoverProgressKey{}).(*takeoverProgress)
	if p == nil {
		return nil
	}
	path := strings.TrimRight(req.URL.Path, "/")
	if !strings.HasSuffix(path, "/responses") && !strings.HasSuffix(path, "/messages") && !strings.HasSuffix(path, "/chat/completions") {
		return nil
	}
	p.advance("headers", false)
	return func(resp *http.Response) {
		if resp == nil || resp.StatusCode < 200 || resp.StatusCode >= 300 || resp.Body == nil {
			return
		}
		p.advance("body", false)
		resp.Body = &takeoverProgressBody{ReadCloser: resp.Body, p: p}
	}
}

type takeoverProgressBody struct {
	io.ReadCloser
	p *takeoverProgress
}

func (b *takeoverProgressBody) Read(buf []byte) (int, error) {
	n, err := b.ReadCloser.Read(buf)
	if n > 0 {
		b.p.advance("body", false)
	}
	return n, err
}

// A timeout gives no evidence about the execution of provider-side tools or
// background jobs. Only a self-contained foreground request whose tools run
// on the client can be replayed automatically.
func takeoverRequestReplayable(body []byte) bool {
	req := openAIRequestPayloadView(body)
	if !req.IsObject() || req.Get("background").Bool() || req.Get("conversation").Exists() {
		return false
	}
	if req.Get("web_search_options").Exists() || gjsonArrayHasElements(req.Get("mcp_servers")) {
		return false
	}
	tools := req.Get("tools")
	if !takeoverClientToolsOnly(tools) {
		return false
	}
	hasTools := gjsonArrayHasElements(tools)
	hasSearchHistory := false
	for _, item := range req.Get("input").Array() {
		switch strings.TrimSpace(item.Get("type").String()) {
		case openAIAdditionalToolsItemType:
			additional := item.Get("tools")
			if !takeoverClientToolsOnly(additional) {
				return false
			}
			hasTools = hasTools || gjsonArrayHasElements(additional)
		case openAIWebSearchCallItemType:
			hasSearchHistory = true
		}
	}
	// OAuth adds a hosted search tool for this history. The no-tools
	// compaction case pins tool_choice to none during normalization.
	choice := req.Get("tool_choice")
	return !hasSearchHistory || choice.String() == "none" || (!hasTools && shouldPinOpenAIWebSearchHistoryToolChoice(choice.Value()))
}

func takeoverClientToolsOnly(tools gjson.Result) bool {
	if tools.Exists() && tools.Type != gjson.Null && !tools.IsArray() {
		return false
	}
	for _, tool := range tools.Array() {
		switch tool.Get("type").String() {
		case "function", "custom":
		case "":
			// An Anthropic client tool must declare its input schema.
			if !tool.Get("input_schema").IsObject() || !tool.Get("name").Exists() {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func takeoverTimedOutError(ctx context.Context, cause takeoverTimeout, replay bool) error {
	action := NextAccountStop
	if replay && (cause == "headers" || cause == "body") && ctx != nil && ctx.Err() == nil {
		action = NextAccountRetry
	}
	return &UpstreamFailoverError{
		StatusCode: http.StatusServiceUnavailable, ClientStatusCode: http.StatusServiceUnavailable,
		Scope: GatewayFailureScopeRequest, NextAccountAction: action, Reason: OpenAITakeoverTimeoutCode,
		ClientMessage: "Session takeover timed out before the upstream account accepted the request.",
	}
}

// A generation write starts the WS idle clock once. Only upstream reads
// refresh it after that. Pool prewarm and client control frames are excluded.
func noteOpenAITakeoverSocketActivity(ctx context.Context, requestSent bool) {
	if ctx == nil {
		return
	}
	if p, _ := ctx.Value(takeoverProgressKey{}).(*takeoverProgress); p != nil {
		p.advance("body", requestSent)
	}
}

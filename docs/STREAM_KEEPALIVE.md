# Streaming keepalive

A client streaming request can be silent for a long time before the first
model byte: while it waits for a concurrency slot, while the selected upstream
has not returned response headers, and between upstream events. Clients and
intermediate proxies (Cloudflare cuts an idle HTTP response after about
100–125 seconds) drop such connections. The gateway therefore writes
non-semantic SSE heartbeats in each phase.

| Phase | Parameter | Heartbeat |
| --- | --- | --- |
| Account/user-message queue wait | `concurrency.ping_interval` | Entry's queue ping (`data: {"type": "ping"}` on the Anthropic gateway, `:` comment on the OpenAI gateway) |
| Upstream header wait | `gateway.stream_keepalive_interval` | Inbound protocol heartbeat, see below |
| After upstream headers | `gateway.stream_keepalive_interval` | Existing per-protocol idle keepalive |

`gateway.stream_keepalive_interval` defaults to 10 seconds; `0` disables both
the header-wait and the post-header keepalive. Valid non-zero values are 5–30.

## Upstream header wait

Every upstream HTTP attempt of a client streaming request opens a wait when it
is sent and closes it when its response headers arrive or the attempt fails.
When a wait reaches the interval, the gateway commits HTTP 200 with SSE
headers and writes the first heartbeat, then one per interval until upstream
headers arrive or the request ends. A request that finishes or fails before
the first heartbeat keeps its original status code and body.

| Inbound endpoint | Handler | Heartbeat | Failure after a heartbeat |
| --- | --- | --- | --- |
| Anthropic `/v1/messages` | Anthropic gateway | `data: {"type": "ping"}` | `event: error` with `data: {"type":"error",...}` |
| Anthropic `/v1/messages` | OpenAI gateway | `:` comment | `event: error` with an Anthropic error object |
| OpenAI `/v1/responses` | Anthropic or OpenAI gateway | `:` comment | `response.failed` |
| OpenAI `/v1/chat/completions` | OpenAI gateway | `:` comment | `event: error` with `{"error": ...}` |

The Anthropic gateway also serves Gemini and Antigravity accounts through the
same `/v1/messages` and `/v1/responses` handlers. Root aliases and
`/backend-api/codex/responses` share the Responses handler.

Heartbeats are counted with the queue heartbeats as non-semantic output:

- A response that contains only heartbeats is still "not started". An upstream
  failover error switches accounts as before, and the next attempt's wait
  continues the heartbeats.
- When failover is exhausted after a heartbeat, the failure uses the stream
  error event the entry already sends after a queue heartbeat (table above).
  On the Anthropic gateway a stream that carried only heartbeats gets the
  `event: error` line: official Anthropic SDKs dispatch on the event name and
  drop a data-only error frame, leaving an empty stream without an error.
  Errors after model output keep the existing data-only frame.
  Operations error logs keep the real error status through the existing stream
  error marking.
- Upstream errors that a forwarding service writes itself (for example a
  non-failover HTTP 400) behave as they do after a queue heartbeat.
- Upstream response headers such as `x-request-id` that arrive after the
  commit cannot be sent.
- Time to first token, first-byte time and usage come from upstream output and
  ignore heartbeats. A client disconnect during the wait follows the
  downstream-disconnect attribution: the committed 200 is kept, usage is
  settled, and no upstream or account-health failure is recorded.

Not covered, unchanged:

- Gemini native `/v1beta/models/*:streamGenerateContent`: its queue sends no
  heartbeat, go-genai and python-genai reject SSE comment lines, and its error
  responses have no in-stream form, so a committed 200 could not report failure.
- `/v1/chat/completions` on Anthropic-gateway groups: after a queue heartbeat its
  exhausted failover writes no stream error event.
- WebSocket, non-streaming requests, images, videos, and remote compact, which
  keeps its own keepalive.

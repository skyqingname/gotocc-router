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
While a wait is open and nothing has been written to the client for one
interval, the gateway commits HTTP 200 with SSE headers and writes the first
heartbeat, then one per interval. The interval counts from the last byte
written to the client across retries and account switches, so a series of short
attempts cannot leave the client silent for much longer than one interval. A
request that finishes or fails before the first heartbeat keeps its original
status code and body.

| Inbound endpoint | Handler | Heartbeat | Failure after a heartbeat |
| --- | --- | --- | --- |
| Anthropic `/v1/messages` | Anthropic gateway | `data: {"type": "ping"}` | `event: error` with an Anthropic error object |
| Anthropic `/v1/messages` | OpenAI gateway | `:` comment | `event: error` with an Anthropic error object |
| OpenAI `/v1/responses` | Anthropic or OpenAI gateway | `:` comment | `response.failed` |
| OpenAI `/v1/chat/completions` | OpenAI gateway | `:` comment | `event: error` with `{"error": ...}` |

The Anthropic gateway also serves Gemini and Antigravity accounts through the
same `/v1/messages` and `/v1/responses` handlers. Root aliases and
`/backend-api/codex/responses` share the Responses handler.

Heartbeats are invisible to the rest of the request:

- While only heartbeats have been written, the response counts as not started:
  the writer reports nothing written and excludes heartbeat bytes from its size.
  Failover, same-account retry, fallbacks and every other "nothing written yet"
  decision in the handlers and forwarding services behave as they do without the
  keepalive, and the next attempt's wait continues the heartbeats. Non-semantic
  bytes the request wrote itself (queue heartbeats, or the post-header idle
  keepalive of an attempt that then failed over safely) do not stop it either.
- An error response the request writes after the commit, meaning a status of 400
  or higher with a non-SSE body or with no body, becomes the entry's terminal
  stream event from the table above, carrying the type, code and message of the
  error body. Operations error logs record the intended status. Anything the
  request writes after that event is dropped.
- After the commit, response headers set by the request no longer reach the
  client: the status stays 200, so rate-limit and overload failures arrive as
  stream events instead of HTTP status codes and `Retry-After` is not sent, and
  upstream headers such as `x-request-id` or Codex `x-codex-turn-state` are lost
  for that response (Codex loses its turn routing hint; the gateway records the
  turn-state provenance only for headers the client actually received).
- On the Anthropic gateway, a stream that carried only queue heartbeats sends
  its error with the `event: error` line: official Anthropic SDKs dispatch on
  the event name and drop a data-only error frame.
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

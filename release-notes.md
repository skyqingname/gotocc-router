Sub2API Plus v0.1.183+custom.006

## Highlights

- Added a durable OpenAI-compatible video task state machine backed by
  PostgreSQL leases.
- Video channel pricing now uses the accepted request `seconds` and resolution
  instead of HTTP request duration or a single request unit.
- Balance billing reserves before upstream create, captures only on provider
  success, and releases on create failure, provider failure, cancellation, or
  local expiry.
- Status, content, and background polling stay bound to the original account
  and mapped model.
- Preserved the complete deployed `0.1.183+custom.005` GotoCC contract set.

## Changed

- Added explicit `video_task` polling, lease, timeout, response limit, batch,
  and terminal-status configuration.
- Successful video usage records now carry `billing_mode=video`,
  `video_count=1`, requested duration seconds, resolution, immutable routing,
  and pricing snapshots.
- Subscription video usage is deferred until successful terminal state;
  failed tasks consume neither subscription nor successful usage.

## Fixed

- Fixed video create requests priced at one configured unit even when the
  channel is configured per generated second.
- Fixed failed asynchronous video tasks retaining charges because no durable
  poller or refund path existed.
- Fixed video status/content reads being rescheduled to a default model or a
  different account after create.

## Compatibility and migration

- Existing migrations 1-237 remain byte-identical.
- Migration 238 creates an empty `openai_video_tasks` table and indexes. It
  does not inspect, charge, refund, or backfill historical video requests.
- Deployment must add explicit `video_task` values to `config.yaml`; no secret,
  `.env`, Compose, systemd, PostgreSQL/Redis package, DMIT, network, account,
  channel, model, or price-row change is included.
- A binary-only rollback is safe only after all new video tasks are terminal
  and no balance hold remains; migration 238 is left forward-compatible.

## Known issues

- Provider-specific terminal status spellings must be included in the explicit
  production `video_task` terminal sets before enabling the worker.

## Upstream baseline

Deployed GotoCC baseline: 0.1.183+custom.005
GotoCC commit: 298e1584c8916897794a5e6311c580ff56f0ed7d
Plus release: v0.1.183+custom.004
Plus commit: 6c1e6d69398398022a832f869cdb70e69ba47c4d
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564

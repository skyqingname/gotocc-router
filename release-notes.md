Sub2API Plus v0.1.183+custom.008

## Highlights

- Video pricing now keeps the configured unit explicit: `per_request` charges
  once per generated video, while `video` charges once per requested second.
- Asynchronous video tasks freeze that resolved billing mode together with the
  existing price and routing snapshot, then reuse it for terminal usage logs.
- Usage filters, available-channel pricing, and Model Plaza now distinguish
  per-request video prices from per-second video prices.
- Preserved the complete deployed `0.1.183+custom.007` GotoCC video task,
  billing hold/capture/refund, and original-account polling contracts.

## Changed

- Added `billing_mode` to durable OpenAI-compatible video tasks and restricted
  stored values to `per_request` or `video`.
- Model Plaza displays `/ request` and `/ second` row units, uses matching
  badges, and shows a neutral billing-unit header for mixed-mode groups.
- Channel pricing details expose `video` prices as per-second values instead of
  omitting the price or presenting it as a per-request value.
- Chinese and English usage labels identify `video` as per-second video
  billing without changing the persisted API enum.

## Fixed

- Fixed per-request video quotes being rewritten to `billing_mode=video` even
  though their cost used one unit per generated video.
- Fixed terminal asynchronous usage records always being labeled as
  per-second, regardless of the frozen quote.
- Fixed Model Plaza presenting all non-image video models as per-request and
  labeling a mixed table with a token unit.

## Compatibility and migration

- Existing migrations 1-238 remain byte-identical.
- Migration 239 adds `billing_mode` to `openai_video_tasks`, backfills any
  existing task as `video`, removes the temporary default, and adds a check
  constraint for `per_request` and `video`.
- The migration does not change usage logs, balances, channel prices, group
  prices, model membership, account routing, configuration, or Redis data.
- Because the new non-defaulted column is required by task inserts, rollback
  is not binary-only. Stop video writers and restore the matched PostgreSQL,
  Redis, configuration, frontend, and binary set, or apply a reviewed forward
  repair.

## Known issues

- Provider-specific terminal status spellings must remain in the explicit
  production `video_task` terminal sets before enabling the worker.

## Upstream baseline

Deployed GotoCC baseline: 0.1.183+custom.007
GotoCC commit: 81486f4b459ad631f0bddb736f19684cc8e1e0e5
Plus release: v0.1.183+custom.004
Plus commit: 6c1e6d69398398022a832f869cdb70e69ba47c4d
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564

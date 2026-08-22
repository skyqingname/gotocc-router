Sub2API Plus v0.1.178+custom.002

## Highlights

- Import official v0.1.178 channel-monitor quota modes, Chinese-provider
  balance/quota support, and time-based channel pricing.
- Preserve the latest Plus Codex mode-only behavior, including the explicit
  `device` default, prompt-cache, compaction, and WebSocket relay safeguards,
  while adopting the upstream protocol fixes.
- Add the upstream account, scheduler, billing, and platform UI updates with
  synchronized English and Chinese locales.
- Preserve every active GotoCC production contract (LC-001 through LC-004 and
  LC-006 through LC-011) while keeping retired LC-005 absent.

## Changed

- Add forward-only migrations for platform quotas, the Plus Codex mode backfill,
  channel time pricing, and monitor quota configuration.
- Keep the custom Go module identity and Plus outbound identity precedence
  while importing the official v0.1.178 baseline.
- Compose asynchronous-image failed-task deletion and private exact ZIP keys
  with durable PostgreSQL ownership, same-user URL renewal, and streaming
  uploads; storage keys are not exposed in public JSON.
- Preserve atomic registration-time consumption for both one-time and reusable
  invitation codes.
- Make delayed release finalization resumable after a newer version has already
  been prepared, while keeping rollback documentation synchronized.

## Fixed

- Converge the selected Codex fingerprint identity across HTTP, WebSocket,
  compaction, and account-test transports without changing source precedence.
- Localize the dashboard prompt-cache hit-rate label consistently in English
  and Chinese.

## Compatibility and migration

- Existing data remains compatible. Startup applies migrations 224, 225, 226,
  and 228 in lexical order; no manual migration command is required. Migration
  224 normalizes the Codex fingerprint mode for top-level OpenAI OAuth
  accounts (defaulting missing or invalid values to `device`) and removes that
  field from non-applicable accounts. Migrations 225, 226, and 228 add channel
  time pricing, monitor quota modes, and the expanded platform-quota constraint;
  migration 227 is intentionally unused.
- Migrations are forward-only. Rolling back the application does not undo the
  migration or its database trigger; back up PostgreSQL before upgrading. A
  database rollback requires restoring a backup or applying an audited
  compensating SQL migration.
- Existing GotoCC migrations `224_add_image_objects.sql` and
  `225_restore_openai_video_prices.sql` remain immutable. Their full filenames
  differ from the new Plus `224` and `225` files, so both lineages remain
  independently tracked.

## Known issues

- None known.

## Upstream baseline

Official release: v0.1.178
Official commit: e0c48a19ed794a565e3858662520afe0a1f9f0ba

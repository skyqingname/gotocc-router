Sub2API Plus v0.2.0+custom.002

## Highlights

- Imported official Sub2API v0.2.0 while preserving the complete GotoCC
  production contract set, including teams, permanent image objects, unified
  OpenAI-compatible video routing, and terminal-state video billing.
- Added per-group OpenAI reasoning-effort over-limit handling, Force Fast /
  Free Fast controls, one-hour cache-write pricing, native compaction usage,
  requested reasoning-effort visibility, and per-user public-group controls.
- Added a configurable Prompt Audit defense template, delivered as an isolated
  system message while audited content remains JSON-encoded untrusted data.

## Changed

- Added Codex automation and delegation bootstrap normalization after the
  immutable ingress security-audit gate.
- Added Kimi-native Responses routing, fallback sanitization, upstream session
  isolation, and the v0.2.0 model and reasoning policy updates.
- Administrators can edit or restore the Prompt Audit template; synchronous
  blocking, asynchronous workers, and endpoint probes share the active value.
- API-key authentication snapshot v23 combines team attribution and billing
  state with the v0.2.0 Free Fast group field.

## Fixed

- Preserved requested and effective reasoning effort, native compaction, team
  actor/billing-owner attribution, and video billing units in the same usage
  record and query contract.
- Preserved image input pricing and per-second video pricing alongside the new
  one-hour cache-write price in channel and Model Plaza fallbacks.
- Kept audited content unable to override the configured Guard system prompt;
  configuration summaries record only the prompt hash.

## Compatibility and migration

- Existing production migrations through 239 remain byte-identical, including
  the durable video task table and its frozen billing mode.
- The v0.2.0 schema additions use migration identities 240 through 246 for
  native compaction, requested reasoning effort, public-group restrictions,
  one-hour cache-write pricing, Force Fast, reasoning over-limit behavior, and
  Free Fast. This avoids reusing production migration identities 238 and 239.
- Existing Prompt Audit configurations without `audit_prompt` load the built-in
  template. No new environment variable, Redis format, Compose, or systemd
  change is required.

## Known issues

- The published upstream `v0.2.0+custom.001` tag still records its own mapping
  as `planned`; this release pins the published tag and peeled commit.

## Upstream baseline

Plus release: v0.2.0+custom.001
Plus commit: 2b921d7bf09c0484678862b854b52a4a0fb08dda
Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918

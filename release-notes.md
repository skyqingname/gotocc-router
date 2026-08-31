Sub2API Plus v0.1.183+custom.005

## Highlights

- Rebased the complete GotoCC production contract set onto the immutable
  Sub2API Plus `v0.1.183+custom.004` release.
- Added configurable OpenAI-compatible Content Moderation endpoint pools with priority failover, cooldown, manual pause, and health visibility.
- Improved Prompt Audit and Content Moderation observability and protocol coverage.
- Added Moderation platform attribution and status semantics to audit records.
- Improved asynchronous image task durability, requested/actual image observability, and PostgreSQL-backed ZIP download recovery.
- Preserved reusable invitations, OpenAI and Grok video routes, CC Switch,
  direct Images model forwarding, the GotoCC homepage, teams, model plaza,
  durable async-image objects, ranking privacy, and JSON video billing.

## Changed

- Prompt Guard audits user input only.
- Added text Moderation API strategy guidance and clarified its interaction with global moderation modes.
- Removed the retired upstream billing probe while retaining GotoCC team and
  usage ownership semantics.
- Retained the complete `v0.1.183+custom.003` PR #62 ancestry, including
  migration 234 and its prompt-audit billing fixes, before applying the
  `v0.1.183+custom.004` changes.
- Standardized deterministic candidate validation on Docker for macOS and
  Linux while keeping the production output CGO-disabled for `linux/amd64`.

## Fixed

- Fixed manual Moderation endpoint tests being overwritten by persisted pool configuration.
- Fixed multi-image async edit submission and durable task metadata.
- Fixed completed async image ZIP downloads after Redis task expiry.
- Preserved Gemini image-response normalization together with the target
  Codex local-group quota headers.

## Compatibility and migration

- Existing migration filenames and bytes through the deployed GotoCC/Plus 233
  lineage are unchanged.
- Migration 234 removes retired upstream-billing-probe account keys and its
  global settings row, and emits account-change outbox entries. Restoring that
  retired configuration requires the deployment backup, not a binary-only
  rollback.
- Migrations 235-237 add prompt-audit observability, a concurrent client-IP
  index, Moderation endpoint attribution, and asynchronous-image storage/count
  metadata. Existing completed image tasks can recover actual image counts
  from stored result data, but tasks completed before storage keys were
  persisted cannot recover ZIP downloads after Redis expiry.
- No `.env`, secret, Compose, systemd, PostgreSQL/Redis package, DMIT, network,
  API-key group, account allowlist, channel, model, or price data change is
  included in the local candidate.

## Known issues

- HTTP image object URLs on a different origin remain blocked by the default
  CSP. Use an HTTPS image storage or CDN URL for browser previews.
- The immutable `v0.1.183+custom.004` release exists, but its own `UPSTREAM.md`
  mapping row still says `planned`; this candidate pins the peeled commit and
  records the metadata discrepancy.

## Upstream baseline

Plus release: v0.1.183+custom.004
Plus commit: 6c1e6d69398398022a832f869cdb70e69ba47c4d
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564

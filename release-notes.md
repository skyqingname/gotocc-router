Sub2API Plus v0.1.177+custom.002

## Highlights

- Rebase all active GotoCC production contracts onto Sub2API Plus
  v0.1.177+custom.001.
- Preserve permanent invitations, video routing, team billing, Model Plaza,
  asynchronous Images objects, and user ranking privacy.
- Include upstream server-timezone group-usage rollups, Codex remote
  compaction v2, and turn-state provenance protection.

## Changed

- Retain the production model-membership source and pricing fallback rules.
- Keep user `/monitor?tab=users` routing role-aware without issuing ranking
  requests for ordinary users.
- Preserve the credential-owner Codex outbound identity precedence.

## Fixed

- Synchronize the usage-log `last_token_ms` field with GotoCC team attribution
  across every write and read path.
- Keep the migration release policy fail-closed for the historical GotoCC and
  v0.1.177 numeric-prefix collisions.
- Generate team invitation, reissue, and ownership-transfer email links from
  the configured `api_base_url` origin when `frontend_url` is unset, then from
  a trusted same-origin or explicit CORS origin. Untrusted Origins and
  wildcard CORS are not accepted as email destinations.

## Compatibility and migration

- Existing GotoCC migrations remain immutable and are identified by complete
  filenames plus SHA-256, including the historical 221/222/223 prefixes.
- New upstream migrations add `last_token_ms`, daily group rollups, and the
  rollup timezone state. They require a production-consistent migration and
  rollback rehearsal before deployment.
- No production `.env`, Compose, systemd, Redis persistence contract, DMIT,
  or network-boundary change is included.

## Known issues

- Daily rollups are derived data; the first startup after migration or a
  timezone change may perform additional synchronization work.
- Turn-state provenance falls back to process-local best effort when the
  shared GatewayCache is unavailable.
- A real Infinite Canvas session and paid image/video generation remain
  separate production acceptance checks.

## Upstream baseline

Official release: v0.1.177
Official commit: 073e92d17178a1ccdb0a27017f572f10c9c7ab62

## Why

GotoCC production runs `0.1.176+custom.003` with active local contracts that
are absent from the immutable Plus `v0.1.177+custom.001` release. Deploying
the upstream Plus artifact directly would remove supported product behavior.

## What Changes

- Start from the immutable Plus tag `v0.1.177+custom.001` and create the local
  candidate `0.1.177+custom.002`.
- Reapply every active local contract and keep LC-005 retired.
- Preserve the production filenames and checksums for the existing GotoCC
  migrations while adopting upstream `221_add_usage_log_last_token_ms.sql`,
  `222_group_usage_daily_rollups.sql`, and
  `223_group_usage_rollup_timezone.sql`.
- Repair team invitation email links when an explicit `frontend_url` is absent
  by using the configured `api_base_url` origin, then a trusted request Origin,
  without allowing a caller-controlled Origin to become an email destination.
- Regenerate Ent, Wire, and embedded frontend assets from the combined source.

## Impact

- The upgrade adds additive database structures and usage-log triggers. Local
  packaging uses the existing customization hook; production migration still
  needs an explicit deploy confirmation and a documented rollback path.
- No `.env`, Compose, systemd, Redis persistence contract, DMIT, or network
  boundary change is included.
- Candidate preparation does not publish a tag, image, Release, or deployment.

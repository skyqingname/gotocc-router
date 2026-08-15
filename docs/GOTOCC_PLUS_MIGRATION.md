# GotoCC Plus Migration

This document defines the production cutover boundary for the migration from
the current GotoCC Sub2API deployment to the pinned Sub2API Plus baseline.

## Persistent data ownership

- PostgreSQL is the business source of truth. Run the migration-lineage
  preflight before Atlas alignment or any new migration.
- Redis is retained across the cutover. Never use `FLUSHDB` or replace its
  volume as part of this migration.
- Keep `JWT_SECRET` unchanged. Existing access tokens are HMAC compatible and
  old tokens without `sid` or `bnd` remain readable.
- Preserve `refresh_token:*`, `user_refresh_tokens:*`, and `token_family:*`.
  These keys carry login continuity and cannot be reconstructed from
  PostgreSQL.
- Preserve `image_task:*` until its normal TTL expires so accepted async image
  requests remain pollable. PostgreSQL `async_image_tasks` is history, not a
  replacement for in-flight Redis state.

## Migration lineage registry

The fail-closed preflight contains fourteen exact rules: nine map divergent
official production/Plus filenames with byte-identical SQL, and five map
GotoCC-owned data into the Plus migration sequence:

| Production filename | Plus filename | Data owner |
| --- | --- | --- |
| `154_reusable_invitation_codes.sql` | `220_reusable_invitation_codes.sql` | reusable invitations |
| `191_add_teams.sql` | `221_add_teams.sql` | teams and attribution |
| `192_harden_team_lifecycle.sql` | `222_harden_team_lifecycle.sql` | team lifecycle |
| `193_add_team_attribution_indexes_notx.sql` | `223_add_team_attribution_indexes_notx.sql` | team indexes |
| `196_add_image_objects.sql` | `224_add_image_objects.sql` | durable image objects |

Every rule binds both filenames, both SHA-256 values, and a schema assertion.
The preflight checks every applicable rule before opening the registration
transaction. It then writes the equivalent Plus rows to `schema_migrations`
atomically. This write means even a clone rehearsal needs explicit database
authorization; never use production startup as a migration probe.

## Prefix-specific cache handling

| Prefix | Ownership | Cutover rule |
| --- | --- | --- |
| `refresh_token:*`, `user_refresh_tokens:*`, `token_family:*` | Session continuity | Preserve; never bulk-delete |
| `image_task:*` | In-flight async image state | Preserve; drain submissions and wait for processing tasks |
| `sticky_session:*` | Derived routing affinity | Preserve when possible; safe to expire naturally after drain |
| `sched:*` | Derived scheduler snapshots and leases | Stop old writers, drain, then let the candidate rebuild; do not delete while either version writes |
| `billing:*`, `apikey:rate:*` | Derived billing/quota cache | PostgreSQL is authoritative; invalidate only reviewed prefixes after drain when schema/version changed |
| `concurrency:*`, `wait:*` | Short-lived request leases | Drain active requests and wait for TTL; never delete under live traffic |
| `apikey:auth:*` | Versioned derived authorization snapshot | Candidate rejects stale versions and rehydrates from PostgreSQL |
| `batch_image:*` | Queue/worker coordination | Stop producers, wait for active workers, preserve PostgreSQL job state, then restart one candidate worker set |

## Schema migration risk gates

- `221_add_teams.sql` is the highest-risk migration. Measure table-lock wait,
  foreign-key validation scans, execution time, WAL growth, and disk peak on a
  production-consistent PostgreSQL copy.
- `223_add_team_attribution_indexes_notx.sql` uses concurrent indexes. A failed
  attempt can leave `indisvalid=false` or `indisready=false`; inspect all five
  target indexes before retrying or approving cutover.
- Record row counts and null distributions for `api_keys`, `usage_logs`, and
  `batch_image_jobs` before and after migration. The migration must not perform
  an unconditional historical attribution backfill.
- Verify existing reusable invitations, teams, memberships, and image objects
  through their Plus API paths. Table presence alone does not prove behavioral
  compatibility.
- Plus `218_clear_non_grok_video_generation_config.sql` stays immutable and
  still clears non-Grok, non-composite video prices. GotoCC Jimeng billing on
  OpenAI-platform groups is restored immediately afterwards by
  `225_restore_openai_video_prices.sql` from `groups_video_price_backup_218`.
  Confirm OpenAI-platform video prices after both files, not after 218 alone.
- Remaining Plus-only migrations after 186, except the OpenAI restore above,
  follow Plus. Keep Plus factory defaults; do not enable IP access control or
  channel monitor v2 as part of cutover.
- The built-in homepage is the current GotoCC production page, not the Plus
  default or compact home. Clear production `home_content` at cutover so that
  page is visible; the admin HTML/URL override remains available afterwards.

## Required drain sequence

1. Disable new async image and batch-image submissions at the edge or
   application gate while keeping read endpoints available.
2. Stop old background producers and wait for active gateway requests, batch
   workers, async image tasks, and usage-write queues to reach zero or the
   documented timeout.
3. Stop every old application writer. Do not overlap old and candidate
   scheduler, billing, or queue writers.
4. Snapshot PostgreSQL, Redis, deployment configuration, and the current
   binary/image; verify restoration in isolation.
5. Start one candidate against isolated production-consistent copies and run
   migration preflight. Only a separately authorized rehearsal may execute the
   new schema migrations.
6. For production, run the same preflight, migrate once, start one candidate,
   verify sessions and critical reads, then scale out.

## User impact and rollback

With the same `JWT_SECRET` and retained Redis volume, normal browser sessions
and refresh tokens do not require a forced login. A cold deletion of session
prefixes would log users out and is prohibited.

Requests accepted before the drain must either finish on the old version or
remain pollable on the candidate. Stopping workers without a drain can strand
async work until timeout. Rebuilding scheduler and billing caches can add
temporary database load and first-request latency, so scale out only after the
single-instance verification is clean.

Database rollback is not a binary-only rollback after forward migrations. The
old application must be tested against the post-migration additive schema, or
production must be restored as a matched PostgreSQL, Redis, configuration, and
binary/image set.

## Local candidate evidence

The migrated semantic source is recorded by commit
`f60ee2fa98961423ab3b29c3b5621bc0c530cba7` (`feat: migrate GotoCC
production baseline to Plus`). The final release manifest, rather than this
document, is authoritative for the later documentation-only closure commit.

The 2026-08-14 local candidate passed the repository checks and the complete
GotoCC verification sequence: upgrade-tool tests `125/125`, markers `160/160`,
targeted `176/176`, full `180/180`, and release `182/182`. Regenerating Ent,
Wire, and the embedded frontend produced no drift. The migration OpenSpec
change also passed strict validation. The before/after evidence fingerprints
were:

- Ent: `0f12bab86603fbb018b4339d4f65fabb5e2d2c87671c39af84685f273171eb0b`
- Wire: `43a93d89d5dfa3392467bac7db657a2504f4d3b2381cc1ef477a57ec1cf5eb26`
- embedded dist: `b65eb8ea27ec8a075a2789a44b2e933d6418b0d31b238eb68c7a286b6d7e669b`

The first release rebuilt from that source commit recorded the same full
commit in its manifest. Its independently recalculated Linux/amd64 binary
SHA-256 was
`ec584a9ba0ef62e246e6ad7e7ce0c845987725dcad950f68bf9005d29ef48ec9`.
It is a reproducibility checkpoint; the final closure release must be rebuilt
after the documentation-only commit and is the only candidate package eligible
for a separately authorized rehearsal.

Real-browser checks covered `/home`, `/team`, `/keys?scope=team`,
`/admin/reusable-invitation-codes`, `/async-image`, `/model-plaza`, and
`/models` at desktop and 390 x 844 viewports. Page-level horizontal overflow
was zero, `/models` redirected to `/model-plaza`, and the team onboarding flow
completed all nine steps. PostgreSQL/Redis clone rehearsal and rollback-asset
restoration remain separately authorized and have not been performed.

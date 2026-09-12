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

## Unified NewAPI video group

GotoCC keeps all video models exposed by the same NewAPI upstream in one
OpenAI-compatible API key group. Model names such as `grok-imagine-video`,
`video-ds-*`, Seedance, Kling, and MiniMax do not select a local Sub2API platform;
the group platform remains OpenAI and the upstream performs provider routing.

The canonical upstream surface is `POST /v1/videos`, followed by
`GET /v1/videos/:task_id` and `GET /v1/videos/:task_id/content`. GotoCC also
accepts the legacy `/videos/generations` aliases used by Infinite Canvas and
normalizes them to the canonical NewAPI paths before forwarding. Grok-native
edits and extensions remain restricted to a real Grok group.

Channel create/update validation accepts both video pricing contracts without
introducing another persisted enum: `per_request` means one unit per generated
video, while `video` means one unit per requested video second. The admin UI
labels the latter as per-second video billing so operators do not confuse the
two units.

With `video_task.enabled`, create validates the numeric JSON `seconds` field
and resolves `size` to the configured video resolution tier before forwarding.
Balance-mode requests reserve the exact seconds-based quote first. From owned
0.2.1+custom.004, provider completion must also pass an authenticated GET of
`/v1/videos/{task_id}/content` on the original account before capture. A 200/206
response must identify video/binary media and yield at least one byte; the worker
closes the response without downloading the entire video. This verifies initial
content availability, not full-file integrity or permanent future availability.
HTTP 4xx except 408/409/425/429 and non-media 200/206 responses fail the local
task and release the quote, retaining the original provider status. Transport
errors, readiness responses, empty bodies, 408/409/425/429 and 5xx keep the quote
held and retry until the existing task deadline. Create failure, provider failure,
cancellation, or local expiry also releases the quote. Subscription usage is applied only
after success. PostgreSQL stores the task owner and original account, and both
the worker and client status/content reads use that account instead of running
the scheduler again. `NOT_START`, `IN_PROGRESS`, and other unknown states stay
non-terminal; only values in the explicit success/failure/cancelled sets can
settle or release a task.

Status and content verification share the configured worker request timeout.
Recovered completed-but-uncaptured rows go through status/content verification
again; a transient failure moves them back to processing with the next polling
time. Already captured tasks are not automatically refunded or rechecked by this
change. No schema/configuration migration is added.

The resolved billing mode is part of the immutable asynchronous-task quote.
Task settlement and terminal usage recording read that frozen value rather
than re-resolving mutable channel pricing. Migration 239 adds the required
`billing_mode` column, backfills pre-existing task rows as `video`, removes the
temporary default, and restricts future rows to `per_request` or `video`. It
does not rewrite existing usage logs, balances, price rows, routes, or Redis.

Migration 238 only creates the empty `openai_video_tasks` table and indexes. It
does not scan or reinterpret historical usage. Before rollback, all new tasks
must be terminal and no row may retain a balance hold; an older binary cannot
poll or settle the new task records.

Before production, review the OpenAI account Base URL, credential type, model
allowlist, channel mapping, and per-model billing mode. Changing group accounts,
models, or pricing is a separately authorized data/configuration mutation; the
code compatibility layer alone does not authorize it.

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

## Upgrade from 0.1.178+custom.006 to 0.1.183+custom.003

The 0.1.183 candidate keeps every migration file already present in the
deployed GotoCC/Plus lineage byte-equivalent and adds five Plus migrations:

- 229 creates effective requested/upstream model indexes concurrently;
- 230 widens Composite route targets to Kimi, Zhipu, and DeepSeek;
- 231 adds nullable fast, flex, and interval pricing multipliers;
- 232 creates disabled-by-default plugin installation and binding tables;
- 233 adds nullable plugin artifact storage.

The first health gate may restore the frozen 0.1.178 binary without reversing
these additive or constraint-widening migrations, provided no operator has used
the new multiplier or plugin surfaces. Once 0.1.183-only data is written, stop
writers before rollback and preserve the matched PostgreSQL, Redis,
configuration, binary, and plugin data-directory state. Never drop the new
tables, columns, or indexes as an incidental rollback step.

Migration 229 can leave an invalid concurrent index after interruption. Before
retrying, inspect both target indexes in `pg_index` and verify `indisvalid` and
`indisready`; do not assume `IF NOT EXISTS` repairs an invalid index.

## Upgrade from 0.1.183+custom.003 to 0.1.183+custom.005

This candidate pins the immutable Plus `v0.1.183+custom.004` tag at
`6c1e6d69398398022a832f869cdb70e69ba47c4d`. That commit contains
`v0.1.183+custom.003` commit
`e94f300b586d8ceb91ba526b13313407b99ffbff` (PR #62) as an ancestor, so the
`.003` release was not skipped. Its retired upstream-billing-probe behavior,
prompt-audit billing fixes, and migration 234 remain in the candidate.

Migrations 235-237 then add prompt-audit observability, a concurrent client-IP
index, Moderation endpoint attribution, and asynchronous-image storage/count
metadata. All active GotoCC LC-001 through LC-012 contracts are adapted on top;
LC-005 remains retired and guarded against regression.

Local generation and validation run in a deterministic Docker image on the
host, while the production artifact is built CGO-disabled for `linux/amd64`.
This upgrade does not authorize or include changes to `.env`, secrets,
Compose, systemd, PostgreSQL/Redis packages, DMIT, network boundaries, account
allowlists, groups, channels, models, or price data.

Migration 234 removes retired probe configuration from existing rows. After it
has executed, a binary-only rollback cannot restore those values; production
rollback must use the matched deployment backup or an audited forward fix.

## Upgrade from 0.1.183+custom.007 to 0.1.183+custom.008

This release extends LC-012 and LC-013 without adding a new protocol or billing
enum. It preserves `.007` durable polling and terminal hold/capture/refund,
fixes the unit identity carried from quote to usage, and makes every pricing
surface distinguish per-request from per-second video models.

Migration 239 is an additive task-table migration but its new column has no
steady-state default. An older binary cannot safely resume video-task writes
against the migrated schema. Rollback therefore requires stopping video
writers and restoring the matched PostgreSQL, Redis, configuration, frontend,
and binary set, or applying a reviewed forward repair.

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

## Upgrade from GotoCC 0.2.0+custom.002 to 0.2.0+custom.003

The candidate imports Plus v0.2.0+custom.002 at
`cd1d8438cbe19358936605af7e6b20954283bf15`, including PR #70.
All GotoCC production SQL through 246 remains unchanged. Newly imported SQL
uses the following filename mapping; file contents are retained from the
upstream release.

| Upstream filename | GotoCC filename |
| --- | --- |
| `245_client_disconnect_risk.sql` | `247_client_disconnect_risk.sql` |
| `246_client_disconnect_lifecycle_observability.sql` | `248_client_disconnect_lifecycle_observability.sql` |
| `247_usage_log_completion_metadata.sql` | `249_usage_log_completion_metadata.sql` |
| `248_content_moderation_session_blocks.sql` | `250_content_moderation_session_blocks.sql` |
| `249_content_moderation_session_blocks_unique.sql` | `251_content_moderation_session_blocks_unique.sql` |
| `250_content_moderation_input_content.sql` | `252_content_moderation_input_content.sql` |

The six migrations create disconnect state/event tables and session-block
tables, add usage completion/source and moderation-input columns, backfill
existing usage completion metadata, and create the session-block identity
constraint. PostgreSQL remains authoritative; no Redis format migration is
required. Existing users, credentials, balances, team ownership and frozen
video billing units are preserved.

Take a matched database/runtime backup before production startup and stop the
old writer before starting the candidate. Backfill and index creation require
a measured local rehearsal and a production size/lock review. After forward
migration or new writes, a binary-only downgrade is not a data rollback.


## GoToCC 0.2.1 candidate lineage

Plus input: `v0.2.1+custom.001`, commit `39f6e2908975636956c184bbc084e90c8b392f74`.
The 303 previously owned SQL files remain byte-for-byte unchanged. PR #5 adds
`253_api_key_smart_routing.sql` and `254_batch_image_routing_group.sql` unchanged.

| Plus filename | Owned filename |
| --- | --- |
| `251_channel_monitor_gpt6_astra.sql` | `255_channel_monitor_gpt6_astra.sql` |
| `252_add_usage_log_upstream_request_id.sql` | `256_add_usage_log_upstream_request_id.sql` |
| `253_add_usage_log_upstream_request_id_index_notx.sql` | `257_add_usage_log_upstream_request_id_index_notx.sql` |
| `254_channel_max_reasoning_effort_multiplier.sql` | `258_channel_max_reasoning_effort_multiplier.sql` |
| `255_group_codex_models_manifest_config.sql` | `259_group_codex_models_manifest_config.sql` |

Only imported filenames change, preserving SQL contents and checksum identity.
Column additions and constraints require table locks; constraint validation scans
API keys, and the usage request-ID index performs a concurrent scan with extra
index and WAL disk usage. Legacy request IDs and batch group IDs stay NULL;
no business-data backfill is attempted. The Astra monitor update applies only
to factory settings. Existing credentials and balances are not changed.

The auth snapshot version includes routing mode, team attribution and Codex
manifest fields; existing session/JWT data and unrelated Redis keys remain.
Smart response affinity uses the existing response TTL in shared Redis.
Downgrades must first disable smart keys or restore explicit fixed groups;
retain additive schema and historical ownership. Never overlap writer versions
or restore an older dump over new transactions.

## GoToCC 0.2.1+custom.003 lineage

Plus input: `v0.2.1+custom.002`, commit `1b95c72f186275f582714bed38a1a4af122429f1`.
All owned SQL through 259 remains unchanged. Upstream
`256_client_disconnect_session_scope.sql` is imported as
`260_client_disconnect_session_scope.sql` with identical SQL contents.

Migration 260 adds session identity and user/key display snapshots to disconnect
events, backfills existing user emails and key names, and assigns old events to
`legacy`. It replaces the event primary key, drops and recreates the derived
risk-state table with a per-session key, and builds four ordinary indexes.
ALTER TABLE, primary-key replacement and non-concurrent indexes take table
locks; event backfills and indexes require additional heap/index/WAL space.
Runtime depends on these new columns. Stop the old core before startup and
keep database and matching runtime backups; old/new writers must not overlap.

The migration turns consecutive-disconnect banning off and advances its
generation. Administrators must review session scope before re-enabling it.
It preserves event rows, balances, usage, credentials and task ownership, but
reinitializes derived streak counters. No Redis flush or new config variables
are required. The old application is not compatible with the new risk-state
primary key/session columns: binary-only rollback is unsupported. After new
business writes, preserve current data and use a forward fix or separately
planned schema adaptation; never overwrite it with an old dump.


## GoToCC 0.2.1+custom.006 lineage

Plus input `v0.2.1+custom.003`, commit `94beb01630fa163bc7c112202eafd9b8946ca725`.
Owned SQL through 260 is byte-for-byte preserved. Upstream
`257_usage_timing_version.sql` becomes `261_usage_timing_version.sql`;
`258_openai_group_quota_follow_reset.sql` becomes `262_openai_group_quota_follow_reset.sql`.
Both imports retain their exact upstream SQL contents. No checksum exceptions are added.

261 adds the timing version and compaction output constraint, clears derived old
TTFT aggregates/histograms and removes openai_ttft_mode. Raw usage and money
remain intact; historical strict token timing cannot be backfilled. ALTER TABLE
requires table locks and aggregate rewrites consume WAL/space.
262 adds group source configuration, subscription event markers, observation and
reset-event tables and ordinary indexes. Index building takes locks and disk;
initial source observation establishes a baseline without resetting usage.
Monthly resets require explicit opt-in. No new Redis or external config is needed.

The subscription charge transaction applies pending reset events before charging;
team/member/key attribution and video terminal settlement remain in the same
owned transaction flow. Stop old writers before migration. Preserve database
and matched runtime resources before updating. Older binaries cannot maintain
follow-reset event/charge ordering after activation, and a binary rollback does
not restore cleared derived metrics. After migration or new writes, preserve
current data and forward-fix; do not restore an old dump over live data.


## Owned 0.2.4+custom.002: Plus v0.2.4+custom.001

Source: `92e12acd4b39f030b56e635bcc02e239e14843e9`; previous owned tree:
`6446a85256e206996b9cc0694856fab2a171f053`. The target Plus tree is the base.
All historical owned SQL through 262 remains byte-for-byte unchanged.

| Plus file | Owned file |
| --- | --- |
| `259_group_model_allowlist.sql` | `263_group_model_allowlist.sql` |
| `260_group_model_allowlist_repair.sql` | `264_group_model_allowlist_repair.sql` |
| `261_add_minimax_platform.sql` | `265_add_minimax_platform.sql` |
| `262_normalize_legacy_model_allowlist.sql` | `266_normalize_legacy_model_allowlist.sql` |
| `263_preserve_existing_access_log_persistence.sql` | `267_preserve_existing_access_log_persistence.sql` |

The five incoming files retain upstream content, including original commentary.
Their new filenames follow the owned lineage. 263/264 rename and repair the
groups model policy; 265 adds MiniMax to existing platform constraints; 266
normalizes legacy allowlists and disables enabled empty lists; 267 preserves
explicit runtime access-log persistence for existing installations.

DDL takes table locks; constraint validation scans the four existing platform
columns. Group backfills/normalization take row locks and generate WAL. Reserve
space for a database backup and WAL; this change creates no business task table
and does not backfill billing/usage. Invalid model policies or runtime-log JSON
stop migration with an identifier-only diagnostic. Management clients must move
from `models_list_config` to `model_allowlist`; enabled policies now govern
inference as well as discovery. API-key cache version 26 invalidates older
snapshots through the existing cache mechanism; Redis is not flushed.

The previous binary expects the old group column. After forward migration or
new writes, keep current data and repair forward; do not restore an old dump
over current data or describe an old-binary replacement as rollback. Stop the
old core before starting the candidate; scheduler/billing/video workers must
not overlap. Local backup and actual runtime evidence belong in the workspace
history record. Publication and production update remain separate user steps.

## Owned affiliate generation change

Owned custom.003 adds migration 268 for reusable-code ownership, actual invitation-code snapshots and three-generation ledger snapshots/deduplication. It preserves historical money and only backfills unbound referral relationships when an administrator assigns a code owner. See [AFFILIATE.md](AFFILIATE.md) for table locks, index space and old-binary semantic incompatibility.

## Owned v0.2.4 custom.004

The full Plus input is `v0.2.4+custom.002` at `fdb9c6de8a959056d6678778979b60c0b0bf20e6`. Existing owned SQL through 268 remains byte-for-byte unchanged. Upstream `264_clarify_openai_quota_reset_baseline.sql` is imported verbatim as `269_clarify_openai_quota_reset_baseline.sql`; its sole COMMENT statement documents the accepted weekly reset baseline without changing schema or data. The upstream release prose says no new migration, but the source tree does contain this metadata-only SQL file.

The release combines [three-generation affiliate changes](AFFILIATE.md) and PR #6 routing-priority visibility with upstream timing, post-audit WebSocket leases and atomic quota-source/group-copy behavior. No new Redis persistence layout or runtime file is introduced. The first accepted weekly observation remains a non-resetting baseline; later eligible windows and live source membership control reset writes. Keep one writer, preserve current data after writes, and use forward correction for recovery.


## 0.2.4+custom.005 邀请列表修复

本版继承 `.004` 的上游与全部既有迁移，只修正管理员邀请记录查询的 GROUP BY 字段。相对 `.004` 无新 migration、数据回填、配置或运行资源变化，不改变返佣计算、余额和邀请关系。无需恢复历史备份。退回 `.004` 会重新出现列表查询错误；此前迁移和资金写入的回滚限制仍然适用。


## 0.2.4+custom.006 邀请返利展示

`GET /api/v1/user/aff` 新增 `show_rebate_details` 布尔字段，以 `reusable_invitation_codes.owner_user_id` 是否存在当前用户为准；停用、到期和用尽不清除归属。仅顶部统计、三代分佣说明和最近三代名单读取此字段，分享及转余额仍可使用。这是界面展示条件，不改变 API 原有数据字段或返佣获得资格。无新增SQL迁移、配置和数据回填；回到 `.005` 会重新显示这三部分，既有迁移及资金写入仍遵循原回滚限制。

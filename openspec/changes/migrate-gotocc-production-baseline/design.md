## Context

The old production branch and Plus share an official Sub2API history, but they
made independent persistent-data and product changes after migration 186. The
migration runner identifies a migration by its complete filename and SHA256,
not by numeric prefix or resulting schema. Nine production migrations are
byte-identical to Plus migrations under different filenames. Other production
tables, including teams, reusable invitation codes, and durable image objects,
have no Plus repository implementation yet.

The new baseline must preserve Plus ownership. The old repository is a source
of behavioral contracts, reviewed data lineage, tests, and rollback evidence;
it is not a source-tree merge parent.

## Baseline and ownership

The candidate starts directly from annotated tag `v0.1.173+custom.004`, peeled
commit `5439ce72f21dbdd35b5e1466b0ad6b58b4e9541a`. Plus owns all existing files,
core behavior, migrations, dependencies, repository rules, and release
metadata. GotoCC additions use the current Plus module path and architecture.

When both codebases implement the same concern differently, the Plus design is
retained and the GotoCC requirement is integrated through the narrowest
current extension point. Generated files are recreated only after semantic
source changes are complete.

Local customization identifiers are immutable audit identities. LC-005 remains
a retired tombstone. LC-006 through LC-010 keep their existing meanings.

## Migration lineage preflight

The runner owns a closed legacy-equivalence registry. Each rule contains:

- the legacy production filename;
- its exact released SHA256;
- the Plus target filename and exact current SHA256; and
- a schema verifier for the tables, columns, constraints, indexes, functions,
  or triggers created by that migration.

After acquiring the existing PostgreSQL advisory lock and before Atlas
alignment or any migration execution, the runner reads the full migration
state and evaluates all relevant rules. A rule may record the target filename
only when the legacy row exists with the expected checksum, the target file is
still the expected immutable content, the target row is absent, and the schema
verifier succeeds. Recording all applicable target filenames happens in one
transaction after every rule passes.

If both filenames already exist, both checksums must match. If neither exists,
normal Plus migration execution owns the change. If only the target exists,
normal checksum validation owns it. A mismatched checksum or schema fails
before any new Plus migration executes. The mechanism is not a generic
checksum allow-list and never edits a historical migration file or database
checksum.

The registry contains fourteen reviewed pairs. Nine map the divergent official
production/Plus lineage:

| Legacy production filename | Plus filename |
| --- | --- |
| `187_add_usage_log_session_id.sql` | `189_add_usage_log_session_id.sql` |
| `188_allow_live_usage_request_type.sql` | `190_allow_live_usage_request_type.sql` |
| `189_add_group_allow_live.sql` | `191_add_group_allow_live.sql` |
| `190_add_users_email_alias_dedup_index_notx.sql` | `192_add_users_email_alias_dedup_index_notx.sql` |
| `194_passkey_credentials.sql` | `196_passkey_credentials.sql` |
| `192_group_profit_control.sql` | `198_group_profit_control.sql` |
| `193_group_profit_control_auth_cache_invalidation.sql` | `199_group_profit_control_auth_cache_invalidation.sql` |
| `194_add_usage_log_upstream_response_model.sql` | `200_add_usage_log_upstream_response_model.sql` |
| `195_add_usage_log_upstream_model_mismatch_index_notx.sql` | `201_add_usage_log_upstream_model_mismatch_index_notx.sql` |

Plus migration `218_clear_non_grok_video_generation_config.sql` remains
immutable. It still snapshots and clears video prices on non-Grok,
non-composite groups. GotoCC Jimeng video billing uses OpenAI-platform groups
(LC-002) and must not follow that cleanup. The candidate therefore adds
forward-only `225_restore_openai_video_prices.sql`, which restores only rows
whose backup and current platform are both `openai`. Other non-Grok leftovers
stay cleared.

Five map migrations whose data is owned by active GotoCC contracts:

| Legacy production filename | Plus filename | Contract |
| --- | --- | --- |
| `154_reusable_invitation_codes.sql` | `220_reusable_invitation_codes.sql` | reusable invitations |
| `191_add_teams.sql` | `221_add_teams.sql` | team storage and attribution |
| `192_harden_team_lifecycle.sql` | `222_harden_team_lifecycle.sql` | team ownership lifecycle |
| `193_add_team_attribution_indexes_notx.sql` | `223_add_team_attribution_indexes_notx.sql` | team query indexes |
| `196_add_image_objects.sql` | `224_add_image_objects.sql` | durable image ownership |

The preflight must also reject a production-only migration whose schema is
unknown to the candidate, unless that migration belongs to an explicitly
implemented active GotoCC contract. This prevents an apparently successful
startup that silently ignores data owned by missing application code.

## Redis and session continuity

PostgreSQL remains the business source of truth. Redis remains the execution
and session store. The candidate keeps the existing Redis database and the
`refresh_token:*`, `user_refresh_tokens:*`, `sticky_session:*`, `sched:*`,
`billing:*`, `concurrency:*`, and `image_task:*` contracts where they overlap.

Refresh-token records and the persisted JWT secret are continuity data and are
not flushed. Scheduler, authorization, billing, and concurrency entries are
derived or leased state; the deployment procedure drains requests and uses
bounded, prefix-specific invalidation or normal expiry rather than `FLUSHDB`.
Compatibility tests serialize old values with the old implementation and read
them with the candidate.

## Local contract integration

Implementation proceeds in dependency order:

1. reusable invitation repository, service, handlers, OAuth consumption, Wire,
   administration UI, and registration double-path tests;
2. team schema/repository/service/authorization/billing attribution and UI;
3. durable image objects integrated with Plus `async_image_tasks`, while Redis
   remains the short-lived execution state;
4. Jingmeng create/query/content routes integrated beside Plus Grok routes;
5. Images model identity passthrough and Gemini inline-data normalization;
6. one-click key import on the Plus key-management UI;
7. GotoCC homepage and lightweight public statistics on Plus frontend and
   dashboard abstractions; and
8. `/models` compatibility routing to the official Plus Model Plaza.

Every public read/write path must have authorization, ownership, billing, and
failure-behavior tests. Existing data is never considered preserved merely
because its tables remain present.

## Homepage adaptation

The Plus frontend structure is authoritative. The homepage may be refactored
to use current Plus components, router, state, locale, and API clients. It must
retain GotoCC brand identity, business copy, public aggregate statistics,
provider/model summary, primary actions, responsive behavior, and local
`logo.png`. It must not retain Plus default branding or restore TokenFlux.

Visual acceptance covers a desktop viewport and a 390px mobile viewport in a
real browser. Source markers alone are insufficient.

## Verification and release boundary

The candidate must pass strict OpenSpec validation, Plus unit/integration and
frontend checks, the complete GotoCC marker/targeted/full/release sequence,
migration-equivalence tests, old-to-new Redis fixtures, generated-code checks,
and embedded-asset tests.

A representative PostgreSQL and Redis clone rehearsal is a separate authorized
operation. Production deployment requires independently verified rollback
artifacts and separate authorization.

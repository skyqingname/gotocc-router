## Baselines

| Role | Identity |
| --- | --- |
| Production runtime/source | `0.1.177+custom.002` / `e3145f0a8642d54957417f122b17decdbeef132d` |
| Plus input tag | `v0.1.178+custom.001` / tag `fdcdb8370d7c7988597dc530f56924001162807f` / commit `47619b50a039b03a1255f2a1e3d1ea719f365e4e` |
| Official input | `v0.1.178` / `e0c48a19ed794a565e3858662520afe0a1f9f0ba` |
| Local candidate | `0.1.178+custom.002` |

## Integration

The candidate starts directly from the verified Plus tag. GotoCC behavior is
reapplied as semantic source changes rather than by merging its Git ancestry or
copying generated output. Ent, Wire, and embedded frontend output are generated
from the combined tree.

All active LC contracts receive focused verification. LC-005 remains a retired
marker and its TokenFlux marketplace behavior remains absent.

## Migration lineage

Migration identity is the full filename plus trimmed SHA-256. The deployed
GotoCC files `224_add_image_objects.sql` and
`225_restore_openai_video_prices.sql` remain byte-equivalent and immutable.
The new Plus files have distinct full names despite reusing numeric prefixes:

1. `224_backfill_codex_fingerprint_mode.sql`
2. `225_channel_model_time_pricing.sql`
3. `226_channel_monitor_quota_mode.sql`
4. `228_user_platform_quotas_add_cn_providers.sql`

The migration runner keys records by full filename, so an upgraded production
database skips the recorded GotoCC files and applies each new Plus file once.
Migration 224 updates existing account data and creates a trigger; the other
three change schema and constraints. Binary-only rollback does not undo them.

## Asynchronous image composition

One upload pass creates durable PostgreSQL ownership records and records exact
private keys in Redis for bounded ZIP assembly. A single server-time timestamp
drives optional date paths. Public task results and URL-renewal responses expose
object IDs and URLs but never storage keys. Same-user, different-key renewal is
allowed; task polling, deletion, and ZIP downloads remain scoped to the
submitting API key.

## Verification

Run `markers → targeted → full → release`. The release gate builds a local
Linux/amd64 embedded package with a `NOT DEPLOYED` manifest. Production remains
unchanged until a separate deployment authorization covers the migrations and
rollback limits.

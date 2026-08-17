## Baselines

| Role | Identity |
| --- | --- |
| Production runtime | `0.1.176+custom.003` / `95b4848d34ad352f9a4c6a88861a29753160d09f` |
| Local production source | `3bedaa685a37bb93889d808fa82c157814ff582d` |
| Plus input tag | `v0.1.177+custom.001` / `2f2dbcb444bb2329ef463ab30900ca050be3f0bc` |
| Official input | `v0.1.177` / `073e92d17178a1ccdb0a27017f572f10c9c7ab62` |
| Local candidate | `0.1.177+custom.002` |

## Integration

The candidate is created directly from the Plus tag. GotoCC behavior is
reapplied as semantic source changes, never by replacing the target source
tree or copying old generated output. Ent, Wire, and embedded frontend output
are generated from the resulting source tree.

All active LC contracts must be tested individually. LC-005 remains a retired
marker: the TokenFlux marketplace page and local curated model-list API remain
absent.

## Migration Lineage

Migration identity is the full filename and SHA-256. The already deployed
GotoCC migrations `221_add_teams.sql`, `222_harden_team_lifecycle.sql`, and
`223_add_team_attribution_indexes_notx.sql` remain immutable even though
v0.1.177 adds files with the same prefixes. The release gate allows only the
four reviewed GotoCC paths with their exact hashes; ordinary reused prefixes
remain blocked.

On an upgraded production copy, the runner skips recorded GotoCC files and
applies exactly these new upstream files in lexical order:

1. `221_add_usage_log_last_token_ms.sql`
2. `222_group_usage_daily_rollups.sql`
3. `223_group_usage_rollup_timezone.sql`

The changes are additive, but the daily-rollup triggers operate on
`usage_logs`. The rehearsal therefore measures startup time, lock behavior,
database size, trigger effects, rollup synchronization, and binary rollback.
Existing service timezone settings are observed but not changed.

## Verification

Run `markers -> targeted -> full -> release`, then build a Linux/amd64 package
with a `NOT DEPLOYED` manifest. The isolated rehearsal restores a
production-consistent PostgreSQL/Redis copy, runs the candidate migrations,
checks active LC behavior and the new rollup state, verifies the old binary
against the migrated copy, and restores the complete pre-migration set.

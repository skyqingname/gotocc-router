## Baselines

| Role | Identity |
| --- | --- |
| Production runtime/source | `0.1.178+custom.006` / `e4b5e63b6753ef083bdb3f55109fdfcc9a272137` |
| Production public snapshot | `057e423f5324a3085c5ef6e890bb848495d6a1d9` / tree `085f1e31808f82f5fcd25b885be156c1207c84cf` |
| Plus input tag | `v0.1.183+custom.002` / `2b5bd31478415617831d49eea9988be90111d3b7` |
| Official input | `v0.1.183` / `e8cb019fabf8b55199436229044cbf9aa7a82564` |
| Local candidate | `0.1.183+custom.003`; exact commit recorded by the release manifest |

The Plus tag exists as an immutable release, although its checked-in
`UPSTREAM.md` row still says `planned`. The candidate pins the tag and peeled
commit and reports the inconsistent status metadata rather than inferring a
different source identity.

## Integration

The candidate starts directly from the verified Plus tag. GotoCC behavior is
reapplied as semantic source changes, without merging the production snapshot's
Git ancestry. Generated Ent, Wire, and embedded frontend files are rebuilt from
the combined schemas, providers, and Vue source.

All active LC contracts receive focused verification. LC-005 remains a retired
marker and the removed TokenFlux marketplace remains absent. Model-plaza
membership uses the gateway's schedulable account inventory, while the v0.1.183
pricing resolver remains the enrichment source. OpenAI Images keeps requested
model identity, and OpenAI Videos adopts the new account-aware scheduling
result interface.

## Migration lineage and rollback

All migration files present in production remain byte-equivalent. Five new
full filenames are introduced:

1. `229_add_usage_log_effective_model_indexes_notx.sql` creates two indexes
   concurrently and may leave an invalid index after interruption; deployment
   verification must inspect `indisvalid` and `indisready` before retrying.
2. `230_composite_routes_add_cn_providers.sql` widens the existing target
   platform constraint to Kimi, Zhipu, and DeepSeek.
3. `231_channel_pricing_multipliers.sql` adds nullable multiplier columns and
   positive-value constraints.
4. `232_plugins.sql` creates plugin installation/binding tables with all
   installations disabled by default.
5. `233_plugin_artifacts.sql` adds nullable artifact storage for multi-instance
   restoration.

The deployed 0.1.178 binary ignores these additive tables, columns, and indexes,
and accepts rows covered by the widened constraint. Therefore an automatic
binary rollback is valid during the immediate health gate, before any new
pricing multiplier or plugin operation. A later rollback must freeze writers,
retain PostgreSQL/Redis/configuration and the plugin data directory, and review
any data created through 0.1.183-only features. Migrations are not reversed or
dropped during normal rollback.

## Verification

Run `markers -> targeted -> full -> release`. The release gate builds a local
Linux/amd64 embedded package and records `NOT DEPLOYED`, source identities,
generated fingerprints, runtime-resource checksums, and LC results. Production
remains unchanged until a separate deployment authorization covers backup,
migrations, service replacement, browser checks, and rollback limits.

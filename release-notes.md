Sub2API Plus v0.1.183+custom.003

## Highlights

- Rebase the complete GotoCC production contract set onto the immutable
  Sub2API Plus `v0.1.183+custom.002` release and official Sub2API `v0.1.183`.
- Preserve reusable invitations, OpenAI and Grok video routes, CC Switch,
  direct Images model forwarding, the GotoCC homepage, teams, model plaza,
  durable async-image objects, ranking privacy, and JSON video billing.
- Retain the v0.1.183 Codex session affinity, sticky spillover, OAuth 429
  scheduling, Responses tool-call identity, account recovery, monitoring,
  pricing, and plugin changes.

## Fixed

- Model-plaza membership now follows the same schedulable account inventory as
  the live gateway; channel pricing enriches visible models but cannot invent
  models or leak inactive groups.
- Model-plaza links and homepage data requests now honor both the feature flag
  and authentication requirement.
- OpenAI video scheduling reports the selected account through the v0.1.183
  health-observation interface and records the effective scheduled model.
- OpenAI Images continues forwarding the requested model directly without
  applying account or channel model mappings.

## Compatibility and migration

- Existing migration filenames and bytes through the deployed GotoCC/Plus 228
  lineage are unchanged.
- Upgrades from the deployed `0.1.178+custom.006` line apply five forward-only
  Plus migrations: 229 creates concurrent effective-model indexes, 230 widens
  Composite routes to the CN providers, 231 adds optional pricing multipliers,
  and 232-233 add disabled-by-default plugin metadata and artifact storage.
- No `.env`, secret, Compose, systemd, PostgreSQL/Redis package, DMIT, network,
  API-key group, account allowlist, channel, model, or price data change is
  included in the local candidate.
- Immediate health-gate rollback can use the previous binary because the new
  schema is additive or constraint-widening. After operators configure new
  multipliers or install/enable plugins, rollback requires a data-aware review
  and preservation of the plugin artifact directory.

## Known issues

- The immutable `v0.1.183+custom.002` release exists, but its own `UPSTREAM.md`
  mapping row still says `planned`; the candidate records this metadata
  discrepancy and pins the peeled commit instead of trusting the status word.

## Upstream baseline

Plus release: v0.1.183+custom.002
Plus commit: 2b5bd31478415617831d49eea9988be90111d3b7
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564

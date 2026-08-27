## Why

GotoCC production runs `0.1.178+custom.006` with twelve active local contracts.
The immutable Plus `v0.1.183+custom.002` release contains the newer official
runtime and Plus feature line but does not contain the complete GotoCC product
surface. Deploying it directly would regress production behavior.

## What Changes

- Start from immutable Plus tag `v0.1.183+custom.002` at peeled commit
  `2b5bd31478415617831d49eea9988be90111d3b7` and prepare local candidate
  `0.1.183+custom.003`.
- Reapply LC-001 through LC-004 and LC-006 through LC-012 while keeping LC-005
  retired.
- Compose the v0.1.183 scheduler, pricing, plugin, monitoring, and identity
  behavior with GotoCC video, Images, model-plaza, team, and durable-object
  contracts.
- Regenerate Ent, Wire, and the embedded frontend from the combined semantic
  source.

## Impact

- Production startup would apply Plus migrations 229-233. They add concurrent
  indexes, widen one Composite constraint, add nullable pricing multiplier
  columns, and add disabled-by-default plugin tables/artifact storage.
- No existing migration is edited. No `.env`, secret, Compose, systemd,
  PostgreSQL/Redis package, DMIT, or network change is included.
- Candidate preparation produces only a local `NOT DEPLOYED` artifact. Upload,
  public snapshot push, migration, and production replacement require one
  later explicit deployment authorization.

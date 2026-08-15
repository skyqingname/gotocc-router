## Why

GotoCC production currently runs an official Sub2API v0.1.172-derived branch
with local invitation, video, image, homepage, team, billing, and durable image
object behavior. Sub2API Plus v0.1.173+custom.004 is the new code baseline, but
its migration filenames diverge from the production lineage after migration
186 and its unmodified tree does not implement the active GotoCC contracts.

Starting the unmodified Plus binary on the production database is unsafe. It
can commit new Plus migrations and then fail on an equivalent schema change
whose old production filename is unknown to the Plus runner. Replacing the
frontend or services without local adaptations would also make existing data
unreachable through supported product flows.

## What Changes

- Keep the complete immutable Sub2API Plus v0.1.173+custom.004 source tree,
  repository identity, migrations, dependency graph, release history, and
  license notices as the implementation baseline.
- Add a fail-closed migration-lineage preflight that recognizes only reviewed,
  byte-identical legacy/Plus migration pairs and verifies the live schema
  before recording equivalent Plus filenames.
- Preserve the existing JWT and refresh-session contract and define bounded
  Redis compatibility and cache-rebuild behavior without flushing the shared
  database.
- Reimplement the active GotoCC local contracts in the current Plus handler,
  service, repository, Ent, Wire, frontend, and embedded-asset architecture.
- Keep the GotoCC homepage brand and behavior while adapting it to Plus
  frontend structure rather than replacing Plus infrastructure with the old
  file tree.
- Keep the official Model Plaza as the only model marketplace. The retired
  TokenFlux marketplace remains absent and its historical LC identifier is not
  reused.
- Regenerate Ent, Wire, modules, and embedded frontend assets from the
  resulting Plus candidate and verify both Plus and GotoCC test suites.

## Impact

- Persistent data: migration-equivalence records are inserted only after a
  complete fail-closed preflight; later Plus migrations remain forward-only.
- Public APIs: reusable invitation codes, Jingmeng/Grok video compatibility,
  team lifecycle, image routing, durable image object URLs, Model Plaza
  compatibility, and public homepage statistics are restored on the Plus
  baseline.
- Billing and authorization: team actor, billing owner, team scope, member
  allowance, API-key ownership, and read-only image history rules remain
  explicit security boundaries.
- Frontend: GotoCC branding and homepage behavior remain, but component,
  routing, state, localization, and build integration follow Plus.
- Operations: production database, Redis, configuration, deployment, restart,
  and remote push remain out of scope until a separately authorized clone
  rehearsal and production change window.

## Non-goals

- Do not merge the old `src/sub2api` tree or use its generated Ent, Wire, or
  embedded frontend output as conflict resolutions.
- Do not import official Sub2API v0.1.174 or later into this migration.
- Do not restore the retired TokenFlux `ModelMarketplaceView` or
  `/api/v1/marketplace/models` implementation.
- Do not modify production data, Redis, `.env`, deployment configuration, or
  running services in this change.

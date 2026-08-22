## Why

GotoCC production runs `0.1.177+custom.002` with active local contracts that
are absent from the immutable Plus `v0.1.178+custom.001` release. Deploying the
Plus artifact directly would remove supported product behavior.

## What Changes

- Start from the immutable Plus tag `v0.1.178+custom.001` and prepare local
  candidate `0.1.178+custom.002`.
- Reapply every active LC contract and keep retired LC-005 absent.
- Preserve deployed GotoCC migration filenames and checksums while adding the
  four distinct Plus v0.1.178 migrations.
- Compose the v0.1.178 asynchronous-image deletion and exact-key behavior with
  durable object ownership and URL renewal, without exposing storage keys.
- Preserve atomic invitation consumption and regenerate Ent, Wire, and the
  embedded frontend from the combined semantic source.

## Impact

- Startup will perform schema and data writes from migrations 224, 225, 226,
  and 228. Production deployment therefore needs a separate explicit
  confirmation and database-aware rollback plan.
- No `.env`, Compose, systemd, PostgreSQL/Redis package, DMIT, or network
  boundary change is included.
- Candidate preparation does not publish a tag, image, Release, or deployment.

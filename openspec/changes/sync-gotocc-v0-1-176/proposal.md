## Why

GotoCC production runs `0.1.173+custom.004` with active local contracts that
are not present in the immutable Plus `v0.1.176+custom.001` tag. Deploying the
tag directly would remove supported invitation, team, homepage, key-import,
video, Images, and durable image-object behavior.

## What Changes

- Start from Plus tag `v0.1.176+custom.001` peeled commit
  `7b156736bdf0c6f017bf356c1e2eeba37e3a0c23`.
- Reapply the frozen GotoCC production baseline commit
  `0889cfbaf34ffaaab107ef5e96d8cf7e8a82fe90` as semantic source changes.
- Preserve every active LC, keep LC-005 absent, and regenerate Ent, Wire, and
  embedded frontend assets from the combined source.
- Publish no tag, image, branch, deployment, database write, or remote push.

## Impact

- Persistent data gains the additive upstream migration
  `220_group_model_pricing.sql`; existing local migrations `220` through `225`
  keep their filenames and checksums.
- API-key creation combines upstream quota/expiry validation with GotoCC
  personal/team scope attribution.
- The resulting local candidate version is `0.1.176+custom.002` because the
  upstream input tag already owns `.001`.

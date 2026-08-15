## Baselines

| Role | Identity |
| --- | --- |
| Deployed source freeze | `0889cfbaf34ffaaab107ef5e96d8cf7e8a82fe90` |
| Plus input tag | `v0.1.176+custom.001` |
| Plus peeled commit | `7b156736bdf0c6f017bf356c1e2eeba37e3a0c23` |
| Official input | `v0.1.176` / `e803e3851c0a7e222cfadeafad7b8636ab959d11` |
| Local candidate | `0.1.176+custom.002` |

## Integration

The candidate is created directly from the Plus tag. GotoCC changes are
applied at semantic source level. Generated Ent files, Wire output, and
embedded frontend assets are regenerated; they are not copied from the old
candidate.

The only textual conflict is API-key creation. Upstream request validation
runs before GotoCC scope resolution. Personal keys continue to bill their
actor. Team keys keep the actor as `api_keys.user_id`, resolve the team owner
as billing user, and persist the team ID.

## Migration Order

The runner keys migrations by complete filename and sorts only the pending
files before execution. Production already records
`220_reusable_invitation_codes.sql` through
`225_restore_openai_video_prices.sql`; it does not record the distinct
`220_group_model_pricing.sql`. The latter therefore executes once as a new
additive migration under the existing advisory lock. Fresh databases execute
both `220_` files in lexical filename order.

No existing migration is renamed, edited, or marked equivalent. The new group
columns are additive and ignored by the old binary, so binary rollback remains
technically compatible with this schema change. A production rollback still
requires the separately approved drain and verification procedure.

## Verification

The candidate must pass markers, targeted, full, and release gates in order.
The release manifest must identify the input tag, all source commits, generated
fingerprints, Linux/amd64 artifact SHA-256, every LC status, and `NOT DEPLOYED`.
Database/Redis clone rehearsal and production deployment remain separate
authorization gates.

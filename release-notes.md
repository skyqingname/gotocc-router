Sub2API Plus v0.2.0+custom.004

## Highlights

- Completes the owned update channel with immutable binary and pricing assets published together. Supersedes the incomplete custom.003 release.

- Preserved GotoCC teams, permanent image objects, video terminal billing, and configurable audit policies on the upstream v0.2.0+custom.002 baseline.

- Added client-disconnect lifecycle tracking, ordered streak enforcement, automatic user disabling, and administrator event review.
- Added durable Content Moderation session blocks and persisted redacted moderation input for administrator review.

## Changed

- Separated the gray owned-release badge from the upstream status badge; unadapted upstream releases turn the latter amber and can be refreshed and reviewed independently.
- Added GPT-6 Astra official default pricing: $10 input, $50 output, $1 cache read and $12.50 cache write per million tokens, with published long-context and Fast/Flex price rules.
- Prompt Audit now scans the official client-controlled transcript; latest-turn blocking includes the nearest preceding assistant/model output, while Content Moderation remains limited to direct-user content.
- Empty IP last-seen times no longer display as permanent bans for unhit automatic blocks.

## Fixed

- Publish the locally built Linux/amd64 archive and both pricing assets together, then make the complete release immutable.

- Added explicit confidence JSON parsing with a configurable inclusive threshold; custom scoring prompts no longer fail the Qwen3Guard response parser. Legacy configurations keep their original format.
- Node probes now execute and parse a model response, and latest-turn audit includes tool results from that turn.

- Hardened usage settlement after client disconnects so accepted requests retain billing and lifecycle outcomes without silently dropping queued work.
- Closed session-block and disconnect-risk settlement holes so PostgreSQL remains the session-block source of truth and admitted OpenAI WS turns still settle after disconnect.

## Compatibility and migration

- Audit response format and confidence threshold are versioned settings. Existing formats and production enable/block switches remain unchanged until an administrator saves the new policy.

Database migrations 247 through 252 add client-disconnect lifecycle state and events, usage completion metadata, durable Content Moderation session blocks, and persisted moderation input.

## Known issues

- This release provides the locally verified Linux/amd64 binary package used by the production service. Container images and other platform archives are not published for this version.

- Local request verification uses synthetic models and data; real-provider behavior and production-volume migration duration require deployment-specific verification.

## Upstream baseline

Plus release: v0.2.0+custom.002
Plus commit: cd1d8438cbe19358936605af7e6b20954283bf15
Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918

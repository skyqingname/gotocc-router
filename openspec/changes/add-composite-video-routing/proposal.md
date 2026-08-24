# Change: Support NewAPI unified video aliases

## Why

Infinite Canvas sends `grok-imagine-video` creation to `/videos/generations`, while the production video group is intentionally an OpenAI-compatible NewAPI group containing every video model. The gateway currently treats that alias as Grok-only and returns a local 404 before the request can reach NewAPI.

## What Changes

- Allow OpenAI groups to enter their existing video handler through `/videos/generations` create, status, and content aliases.
- Normalize those aliases to NewAPI's canonical `/v1/videos` task surface before building the upstream request.
- Accept the existing `video` billing mode in channel create/update request validation so per-second prices can be maintained through the admin API.
- Preserve the native Grok handler for real Grok groups and keep edits/extensions Grok-only.
- Keep the admin Composite route contract unchanged.

## Impact

- Affected code: video gateway dispatch, OpenAI video URL normalization, tests, and GotoCC migration documentation.
- Data: no schema, migration, environment variable, or Redis contract change.
- Configuration: the production group remains OpenAI; account/model/pricing changes remain separately authorized if live audit finds drift.

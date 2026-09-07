GoToCC 0.2.1+custom.002

## Changes

- Adapted the complete Plus v0.2.1+custom.001 tree (39f6e2908975636956c184bbc084e90c8b392f74), based on official Sub2API v0.2.1, while retaining active GotoCC contracts.
- Integrated reviewed PR #5: explicit smart API Key routing and per-model group priority. Existing keys remain fixed; smart keys choose only groups available to the payer and keep the selected group's billing, audit and resource ownership.
- Fixed downstream client disconnects being reported as generic upstream 502 failures. The handler records a downstream network event with the actual HTTP status, avoids writing another error to the closed connection and excludes the disconnect from account-health failure observations. Partial usage settlement and lifecycle records remain intact.
- Retained upstream request IDs, Codex model manifest configuration, max-reasoning pricing support, and image response improvements from Plus.

- Fixed a 30-second whole-evaluation deadline incorrectly shared by all prompt-audit chunks. Each endpoint attempt now receives its configured timeout, while parent cancellation still applies.
- Added an administrator text test using saved audit configuration and the real evaluation path, with pass/block/error, confidence, reason, endpoint and timing results. No user violation events or business-model calls are created.
- Aggregated confidence evidence now corresponds to the highest recorded score.

- Changed CCS imports and one-click OpenAI/Codex HTTP/WS configuration defaults to `gpt-6-astra`, including the review model. Both entry points share `client-access-defaults.json`.

- Node probes now call the actual audit model directly, without requiring model-list permission. Authentication failures are explicit, and credential clearing is honored.

## Migration and compatibility

Existing production SQL through 252 is unchanged. PR migrations 253/254 add API Key routing mode and original batch-image group. Imported upstream migrations are renamed 255–259 with unchanged SQL contents: factory monitor Astra entry, upstream request ID column and partial concurrent index, max-reasoning multiplier, and group Codex manifest JSON.

ALTER TABLE statements take table locks; routing-mode constraints scan existing keys, and the upstream request ID index scans usage_logs concurrently with additional disk I/O and index/WAL space. Historical usage request IDs and batch groups stay NULL; there is no bulk business-data backfill. The monitor update only affects factory configuration with updated_by IS NULL.

Smart routing policy uses the existing settings table; new response affinity entries use shared Redis with the existing TTL. The auth snapshot version advances to include smart routing, team billing and Codex manifest fields. No Redis flush or new environment variables are required.

Before an application downgrade, disable smart keys or explicitly return them to fixed groups. Keep all added schema and task ownership records. After new writes, swapping the binary is not a data rollback. Do not overlap old/new writers.

## Local acceptance

The Linux/amd64 archive and its runtime pricing resources are built locally for manual acceptance. No new paid image/video probes are used. The underlying cause of downstream connection closure in the historical incident remains unproven; this fix corrects error attribution and handling, not the remote network path. This package has not been published or deployed.

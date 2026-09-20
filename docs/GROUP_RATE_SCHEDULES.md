# Group rate schedules — implementation draft

## Status and scope

This change adds real Go and TypeScript implementations, a reusable Vue editor,
and unit test source. It is **not a completed group-billing feature**. The editor
is not mounted in GroupsView; the new contract is not accepted by existing group
APIs, persisted in the database, hydrated into auth caches, or consumed by the
production billing path. Existing peak-rate behavior is unchanged.

The pull request is for source review only. No Docker tests, lint, typecheck,
application build, production probe, or official submit-pr matrix has run in the
editing environment. Do not mark it ready or merge it as an operational feature.

## Implemented configuration contract

The following is input to the new package/editor, **not an existing HTTP API**:

```json
{
  "enabled": true,
  "timezone": "Asia/Shanghai",
  "rules": [
    {"id": "night", "enabled": true, "start": "22:00", "end": "06:00", "multiplier": 0.5},
    {"id": "peak", "enabled": true, "start": "18:00", "end": "22:00", "multiplier": 1.5}
  ]
}
```

Asia/Shanghai is an example, not a detected production timezone. An empty timezone
inherits the explicitly supplied server timezone; missing/invalid timezones are
errors. A machine-local timezone or browser-local timezone is never substituted.
The runtime needs IANA timezone data (the browser needs the corresponding Intl
support); deployment validation must verify availability of configured zones.

Rules recur daily with inclusive starts and exclusive ends. A start later than
its end crosses midnight. 24:00 is allowed only as an end; 00:00–24:00 explicitly
means all day. Equal times are rejected. Historical one-digit hours such as 1:30
are accepted and normalized internally. Up to 64 rules are accepted. Identifiers
must be unique ASCII letters/digits/underscore/hyphen, 1–64 characters long.
Enabled rules may not overlap, including either half of an overnight window.
Disabled rules are still validated, but do not participate in matching/overlaps.
Disabling the schedule preserves its configuration and resolves to the base rate.

Multipliers are additional factors, following the existing single-window peak
factor convention. The UI labels them as coefficients and previews the effective
value. For a resolved base of 0.4, a window factor of 0.5 yields 0.2. No match
means factor 1, not effective rate 1. An already resolved user override is passed
as the base. Absolute-rate replacement is not implemented or assumed.

Factors/bases must be finite and non-negative. Zero explicitly permits a free
window. Overflow and positive-product underflow to zero are errors. These checks
do not replace the eventual persistence layer's decimal precision/range checks.
This package calculates multipliers, not monetary charges, and introduces no
floating-point money settlement or new ledger.

Repeated DST hours match the same wall-clock rules both times. A skipped hour
has no real instants to match. The caller supplies the trusted request instant;
there is no time.Now(), database access, scheduler mutation, or network call in
the Go evaluator.

## Request snapshot contract

Compile validates a copied configuration once and returns an immutable schedule.
Resolve returns a value snapshot containing the received instant in UTC, resolved
timezone, normalized configuration SHA-256, matched rule ID, base, factor, and
effective multiplier. Reordering equivalent rules does not change the hash.
Editing the original configuration cannot mutate a compiled schedule or snapshot.
The configuration hash identifies input; it is not a signature or validation proof.
The frontend preview is never an authoritative billing input.

## Integration still required before this becomes usable

- Add the persistent Group contract, forward-only migration and legacy conversion;
  regenerate Ent and Wire from semantic sources instead of editing generated files.
  Assign the migration number against the latest main. Map the old subscription
  window once with its actual server timezone; never stack old and new factors.
- Add group create/update/duplicate/batch validation and DTO fields; mount the
  editor under the base multiplier in create/edit forms. Saving a group and its
  schedule must be atomic. Include explicit empty-vs-omitted update semantics.
- Hydrate schedules through every group/auth/scheduler cache path and propagate
  invalidations. Update public channels/model/plan price displays. Do not show a
  preview as if the backend is already charging it.
- Capture the trusted ingress instant and produce the verified snapshot after
  security audit but before account selection/admission/billing side effects.
  Reuse it for profit checks, user-rate precedence, retries and settlement.
  Do not silently apply token schedules to independent image/video prices.
- Exercise ordinary and subscription groups, migration/rollback limitations,
  multi-instance cache invalidation, group copy, concurrency and end-to-end money
  behavior in the repository Docker environment. Preserve HTTP/WS audit ordering.
- Run the official submission process with fresh base/head before ready-for-review.
  No validation success status or machine-readable submit-pr proof was fabricated
  for this draft. No main, releases, tags, production settings, or upstream code
  are changed by these standalone additions.

## Test sources provided (execution pending)

Go tests cover boundaries, gaps, adjacent windows, overnight and all-day windows,
all 1,440 minutes of a day, zero/invalid/overflow/underflow factors, duplicate IDs,
overlap, explicit/fallback timezones, repeated/skipped DST hours, normalization,
immutable snapshots and concurrent reads. TypeScript tests cover corresponding
preview behavior. Vue tests cover bilingual labels, effective-rate preview,
immutable form updates, disabling without deleting rules, adding/removing rows,
and invalid/overlapping-rule reporting.

Use the existing documented checks **inside the repository's Docker validation
container**, not the host:

```sh
# From backend:
go test -tags=unit ./internal/pkg/rateschedule

# From the repository root:
pnpm --dir frontend run lint:check
pnpm --dir frontend run typecheck
pnpm --dir frontend run test:run
```

These focused checks do not replace the full official submit-pr matrix.

# Contributing to Sub2API Plus

Thank you for contributing. Repository-wide mandatory rules are in
[`AGENTS.md`](AGENTS.md); this guide provides the working commands and review
flow.

## Toolchain

Use the versions declared by the repository:

- Go: `backend/go.mod`
- Node.js and pnpm: `frontend/package.json`
- Application release: `backend/cmd/server/VERSION`
- Build and CI parameters: `gotocc-build.json`

Install frontend dependencies with pnpm only and synchronize
`frontend/pnpm-lock.yaml` when they change. Go dependency changes synchronize
`backend/go.mod` and `backend/go.sum`.

## Requirement-based Test Design

The owner's execution policy in `AGENTS.md` takes precedence: do not add or run
test suites in this workspace. The following is the retained contract for
existing upstream test sources, not a command to create or execute them. Preserve
their meaning and keep stubs/mocks compiling when interfaces change. Use actual
behavior, builds, type/static checks and existing read-only generator checks for
task verification.

For existing unit, integration, UI, and tooling tests, derive scenarios and expected results independently from
actual business requirements, acceptance criteria, and authoritative contracts.
Reading implementation code to find entrypoints, dependencies, and test setup
is allowed; treating its current output or branches as the definition of correct
behavior is not.

For each changed behavior:

1. Identify the requirement and its source before choosing assertions. Record
   the relevant requirement, documented rule, or official protocol/source
   reference in the test or fixture so reviewers can check the expectation
   independently. Resolve unclear requirements instead of silently declaring
   current behavior correct.
2. Use independent expected values or invariants. Do not call the code under
   test, reuse its generated output, or copy its algorithm to calculate the
   expected result. Protocol fixtures must have an authoritative source;
   snapshots require review against that source or the business requirement.
3. Cover applicable success, failure, boundary, and required/forbidden side
   effects. Verify observable behavior at the boundary where the requirement
   matters: for example, capture actual outbound requests for header privacy,
   or check that a rejected payment leaves balances and orders unchanged.
   Helper return values or mock call counts alone cannot prove those outcomes.
   Mocks may isolate external dependencies but must not replace the behavior
   the test claims to verify.
4. When implementation conflicts with the requirement, fix the implementation.
   Never weaken assertions, remove cases, or update expected values/snapshots
   merely to make tests pass. A legitimate requirement change must update its
   owning documentation and tests together.
5. Defect regressions must detect the original incorrect behavior and pass with
   the fix. Where feasible, demonstrate this against the pre-fix implementation
   or by restoring the specific defect in isolation; otherwise document the
   limitation and the evidence used. A failure caused only by a missing symbol
   or broken setup does not demonstrate detection of the business defect.

Passing tests and coverage percentages alone do not establish business
correctness. Review the requirement-to-assertion mapping and any untested
outcomes. The AGENTS.md validator protects this rule against removal or
weakening; it cannot determine whether a test's expected behavior is correct.
Provider identity tests additionally follow the
[requirement-based outbound regressions](docs/OUTBOUND_IDENTITY.md#requirement-based-regression-coverage).

## Development Checks

Selection dropdowns use `frontend/src/components/common/Select.vue` and the shared
panel in `SearchableSelect.vue` for both closed and expanded appearance. Do not
introduce native HTML `select` controls or use `.input` for dropdowns: native
expanded menus cannot reliably share the system's theme and option styling.
Use typed option values, disabled options, and `kind: 'group'` headers. The default
`searchable: 'auto'` preserves local search above five options; explicit
`searchable: false` hides only the search input. Clearable and custom
`selected`/`option` slot controls preserve integrated selected content and rich options.
Keep existing search-result popovers and their styling for user, account, model,
and proxy selectors. Search remains inside the dropdown; do not expose a
separate search input above or alongside a second selection field.
Checkbox/multi-selection and other composite controls retain their existing
dropdown layouts and interactions. Keep checkboxes inside the dropdown and
allow repeated selection without reopening it. Keep combined auto-refresh
enable/interval actions in their original menu. Usage history and live requests
are the explicit exception: show a temporary auto-refresh switch directly, with
the interval choices retained in the integrated refresh menu. Do not replace composite
controls with plain selectors or split them into separate visible fields.
The shared dropdown owns keyboard navigation and focus, including disabled/group
skipping, selection, Escape, and Tab. Keep trigger styles in `frontend/src/style.css`
and expanded-menu styles in `SearchableSelect.vue`, including light/dark surfaces,
borders, typography, hover, focus, selected, error and disabled states. Keep spacing
shared, and do not add local appearance replacements. Use `triggerClass` only for
compact height, text sizing or font; layout classes belong on the wrapper.
Individual pages may adjust layout and compact width, height or text size, leaving
enough space for the longest supported translation and the dropdown arrow.
Verify the expanded panel itself in both themes and narrow viewports, including
menus inside dialogs and scrollable cards; checking only the closed trigger is insufficient.

The product build entry compiles the frontend and server with the embedded
frontend. Its parameters are in `gotocc-build.json`; CI uses this entry.
Oracle's final dual-architecture packages use the separate maintenance
workspace entry described in [RELEASING.md](docs/RELEASING.md). Neither entry
automatically regenerates release-channel constants:

```bash
python3 tools/gotocc_build.py --output backend/bin/sub2api
```

GitHub runs `.github/workflows/ci.yml` (GoToCC Build) on pull requests and on
manual dispatch. It runs the same entry point, then type-checks every backend
package with its unit and integration tests:

```bash
go -C backend vet -tags=unit ./...
go -C backend vet -tags=integration ./...
```

There is no full test matrix or promotion gate. Do not run test suites; use
focused checks and actual behavior evidence, and keep existing tests compiling
when a signature or interface changes.

The [seven-provider matrix](docs/SEVEN_PROVIDER_TEST_MATRIX.md) maps the critical
provider scenarios to their observable assertions. The full validation profile
also runs its unit, integration and frontend gates. These execute the required tests and
reject missing, skipped, failed or unfinished cases, including all fourteen
seven-provider failover scenarios and forty-two Chat proxy-binding scenarios;
they do not replace the complete suites.
The integration gate uses the same disposable PostgreSQL/Redis environment as
the repository integration suite. Keep the matrix and gate manifest synchronized
when adding sending entrypoints or changing the owning business contracts.

The separate S3 storage gate requires disposable PostgreSQL, Redis and MinIO
reachable from the validation container. Set `SUB2API_TEST_POSTGRES_DSN`,
`SUB2API_TEST_REDIS_ADDR`, an explicit nonzero `SUB2API_TEST_REDIS_DB`, and
`SUB2API_ASYNC_TEST_S3_ENDPOINT` for those temporary services. MinIO must use the
fixture credentials `minio` / `minio-test-secret`. Inside the validation container,
use the same database for the migration smoke and the storage workflow:

```bash
export SUB2API_ASYNC_TEST_DATABASE_URL="$SUB2API_TEST_POSTGRES_DSN"
python3 tools/check_seven_provider_coverage.py --profile storage
```

This gate runs the existing migration/idempotency smoke before the real
asynchronous-image MinIO workflow, with the integration build tag and uncached
execution. Missing configuration, missing tests and skipped/failed tests reject
the gate; migration failure stops the workflow. Ordinary `unit`/`integration`
coverage profiles and the shared full/CI profile do not provision MinIO or run
this separate storage gate. Set the two `SUB2API_ASYNC_TEST_*` variables inside
the validation container; the launcher does not forward them. Remove all
temporary service containers and their writable state after validation.

The separate [real Chromium management suite](frontend/e2e/README.md) runs
`python3 tools/run_seven_provider_browser.py` as root inside a disposable
validation container with explicit disposable PostgreSQL/Redis settings and
`SUB2API_E2E_ISOLATED=1`. Its runner requires all fourteen browser scenarios to
execute successfully; missing/skipped cases fail. The ordinary frontend gate
checks Vue/API lifecycle regressions and does not claim browser execution.
For shared styling, dialog and dropdown changes, run
`pnpm --dir frontend run test:layout` in the validation container as well. This
independent [component layout suite](frontend/layout-tests/README.md) uses real
Chromium geometry and hit testing without backend accounts or services; it does
not replace the management business suite.
Authorization regressions must use redacted account DTOs and the initiating
`oauth_credential_owner`, include both owner-replacement isolation and unchanged
owner success controls, and retain pending grants across unrelated observation
updates. A late HTTP response test proves UI isolation; an already-issued OAuth
apply needs separate server owner/CAS and persistence assertions. Keep these
layers explicit in the [supported business inventory](docs/SEVEN_PROVIDER_SUPPORTED_BUSINESS.md)
and matrix rather than inferring provider authorization or billing from browser
proxy-management cases.

Provider/account changes, outbound paths, client versions, dependency upgrades
and upstream merges must preserve the trusted identity triple. Follow the
[mandatory outbound identity maintenance contract](docs/OUTBOUND_IDENTITY.md#mandatory-maintenance-contract)
and provide its applicable regression evidence before merging. Do not weaken
identity checks to accept upstream behavior. The repository AGENTS.md validator
protects these rules as well as the existing Codex and audit rules:

```bash
python3 skills/compress-cli/scripts/compress_cli.py check AGENTS.md
```

Backend `unit` tests stay in-process (mocks, memory SQLite, miniredis,
httptest). `integration` tests require Docker Postgres/Redis or an explicit
external DSN.

## Generated Code

Parameter and version generation follows
[Generated artifacts](docs/GENERATED_ARTIFACTS.md). Identify the generator,
review every affected output and run supported read-only checks before fixing
the package source. Upstream baseline metadata is independent of owned version
assignment. Do not assume a successful build proves generated values are current.

After changing `backend/ent/schema`, regenerate Ent and Wire:

```bash
cd backend
go generate ./ent
go generate ./cmd/server
```

Commit the generated output. Do not edit generated Ent or Wire files directly.

## Database Changes

Read [`backend/migrations/README.md`](backend/migrations/README.md) before
adding a migration. Existing migrations are immutable. Use the next unique
numeric prefix and create a forward-only migration.

## Documentation and Localization

- Update English and Chinese frontend locales together.
- Run `pnpm --dir frontend run check:i18n` in the build image for locale
  schemas, static keys, dynamic API enum labels, message compilation, and matching
  interpolation parameters. The frontend build runs this check automatically.
- Keep the three README core section IDs aligned.
- Put detailed operational content in `docs/` or `deploy/`.
- Add user-visible changes to the release notes.
- New configurable parameters belong in root JSON with explicit values, not
  new environment injection. Preserve existing upstream configuration formats.
  Configuration changes need matching consumers, generated outputs, actual
  behavior evidence and owning documentation. Update
  `deploy/` examples when deployment configuration changes.
- Prefer maintained repository scripts and Make targets when documenting
  workflows. Verify native tool syntax, supported versions, and execution
  environments before documenting additional commands.

## Specifications

Use a local OpenSpec change to plan cross-cutting features or changes to public
APIs, persistent data, security boundaries, or multi-module behavior. The
`openspec/changes/` directory is intentionally untracked and must not be
included in pull requests. Start from the tracked example under
`openspec/examples/` when useful.

Record durable behavior in the owning documentation; preserve applicable
existing test contracts without adding or running suites. Use
pull request descriptions and commit history for change rationale. Small fixes
and documentation-only changes do not require an OpenSpec plan.

## Pull Requests

Keep changes focused and use the existing commit style, such as `feat:`,
`fix:`, `test:`, `docs:`, and `chore:`. Complete the pull request checklist
and include:

- The problem and intended behavior
- Important implementation or compatibility decisions
- Verification actually performed and its unverified scope
- Migration, configuration, documentation, and release-note impact

Release publication is a separate maintainer action. A pull request must not
create or move release tags.

Never push `main` directly. Publication merges the accepted candidate branch
into `main` through a pull request, as described in `docs/RELEASING.md`.

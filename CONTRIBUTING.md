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

This is a mandatory repository-wide contract for unit, integration, UI, and
tooling tests. Derive test scenarios and expected results independently from
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

Plain selection dropdowns use native HTML `select` controls and shared styles in
`frontend/src/style.css`. Reuse `frontend/src/components/common/Select.vue` for
typed values, disabled options, and option groups. Its default `searchable: 'auto'`
preserves local search when there are more than five options; explicit
`searchable: false` keeps a plain dropdown native. Searchable, clearable, and
custom `selected`/`option` slot controls use the existing popover in
`SearchableSelect.vue`, preserving integrated selected content and rich options.
Keep existing search-result popovers and their styling for user, account, model,
and proxy selectors. Search remains inside the dropdown; do not expose a
separate search input above or alongside a second selection field.
Checkbox/multi-selection and other composite controls retain their existing
dropdown layouts and interactions. Keep checkboxes inside the dropdown and
allow repeated selection without reopening it. Keep combined auto-refresh
enable/interval actions in their original menu. Do not replace composite
controls with plain native selects or split them into separate visible fields.
Native controls own keyboard navigation and focus for plain dropdowns.
Keep horizontal padding and arrow placement in the shared styles; individual
pages may adjust control width and height.

One entry point builds the frontend and the server with the embedded frontend,
for local release packages and for CI alike. Its parameters are in
`gotocc-build.json`; local packages run it inside the build image defined by
`deploy/Dockerfile.validation`:

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

There is no full test matrix or promotion gate. Run a focused test only to
diagnose an observed problem, and keep existing tests compiling when a
signature or interface changes.

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
- Configuration changes need defaults, storage or environment bindings,
  relevant tests, and owning documentation appropriate to their source. Update
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

Record durable behavior in the owning documentation and automated tests. Use
pull request descriptions and commit history for change rationale. Small fixes
and documentation-only changes do not require an OpenSpec plan.

## Pull Requests

Keep changes focused and use the existing commit style, such as `feat:`,
`fix:`, `test:`, `docs:`, and `chore:`. Complete the pull request checklist
and include:

- The problem and intended behavior
- Important implementation or compatibility decisions
- Tests performed
- Migration, configuration, documentation, and release-note impact

Release publication is a separate maintainer action. A pull request must not
create or move release tags.

Never push `main` directly. Publication merges the accepted candidate branch
into `main` through a pull request, as described in `docs/RELEASING.md`.

# Contributing to Sub2API Plus

Thank you for contributing. Repository-wide mandatory rules are in
[`AGENTS.md`](AGENTS.md); this guide provides the working commands and review
flow.

## Toolchain

Use the versions declared by the repository:

- Go: `backend/go.mod`
- Node.js and pnpm: `frontend/package.json`
- Application release: `backend/cmd/server/VERSION`
- Release and lint tools: `.tool-versions`

Install frontend dependencies with pnpm only:

```bash
pnpm --dir frontend install --frozen-lockfile
```

## Local development and acceptance

Build application artifacts in the existing local Docker environment with the
pinned toolchain. Diagnose concrete problems with focused commands as needed.
The release criterion is the final package running locally and the owner's
manual acceptance. Do not run repeated full matrices or the old submit-pr /
release-finalization chain as routine release gates. Documentation and script
syntax checks can run on the host without starting an application test stack.

Keep the accepted local environment and reusable dependency caches. Remove old
worktrees or build outputs only under an explicit cleanup list; do not prune
unrelated containers, volumes, or project resources.

Publication reuses the same accepted package and follows
[`docs/RELEASING.md`](docs/RELEASING.md). It does not rebuild after acceptance or
wait for GitHub builds. The vircs operations workspace provides the build,
preview and upload commands; the owner completes the online update.

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
- Keep the three README core section IDs aligned.
- Put detailed operational content in `docs/` or `deploy/`.
- Add user-visible changes to the release notes.

## Specifications

Use a local OpenSpec change to plan cross-cutting features or changes to public
APIs, persistent data, security boundaries, or multi-module behavior. The
`openspec/changes/` directory is intentionally untracked and must not be
included in pull requests. Start from the tracked example under
`openspec/examples/` when useful.

Record durable behavior in the owning documentation. Use
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

Never push `main` directly. PRs remain available for source collaboration, but
they are not a prerequisite for publishing a locally accepted tag. Reuse the
actual acceptance evidence instead of restarting the full validation matrix.

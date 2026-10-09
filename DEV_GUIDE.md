# Sub2API Plus Development Guide

This guide contains local setup and troubleshooting notes. Mandatory repository
rules are in [`AGENTS.md`](AGENTS.md), and contribution checks are in
[`CONTRIBUTING.md`](CONTRIBUTING.md).

## Toolchain

Use repository-declared versions:

- Go: `backend/go.mod`
- Node.js and pnpm: `frontend/package.json`
- Build and CI parameters: `gotocc-build.json`

Do not copy tool versions into additional policy documents.

## Local Services

Development requires PostgreSQL and Redis. Use local services or the development
Compose file:

```bash
docker compose -f deploy/docker-compose.dev.yml up -d
```

Keep local credentials in ignored `.env` or config files. Do not add developer
machine paths, passwords, or production data to tracked documentation.

## Install and Build

One entry point builds the frontend and the server with the embedded frontend.
Local release packages run it inside the build image from
`deploy/Dockerfile.validation`, and CI runs the same script:

```bash
python3 tools/gotocc_build.py --output backend/bin/sub2api
# or
make build
```

## Run in Development

Backend:

```bash
cd backend
go run ./cmd/server
```

Frontend:

```bash
pnpm --dir frontend run dev
```

Copy configuration examples to ignored local files before editing them. See
[`deploy/README.md`](deploy/README.md) for service configuration.

## Verification

CI (`.github/workflows/ci.yml`) builds through the entry point above and then
type-checks every backend package with its unit and integration tests:

```bash
go -C backend vet -tags=unit ./...
go -C backend vet -tags=integration ./...
```

Run a focused test only to diagnose an observed problem. `unit` is in-process;
`integration` uses Docker or a real DSN.

## Code Generation

After editing `backend/ent/schema`:

```bash
cd backend
go generate ./ent
go generate ./cmd/server
```

Commit generated Ent and Wire output. Do not edit it directly.

## Common Problems

### Frozen lockfile failure

If `frontend/package.json` changes, run `pnpm install` in `frontend` and commit
the resulting `pnpm-lock.yaml`.

### npm/pnpm installation conflict

Remove only the repository's `frontend/node_modules` after confirming the path,
then reinstall with pnpm. Do not use npm or yarn for this project.

### Interface compilation failure

When a Go interface gains a method, update all production implementations and
test stubs/mocks before running the broader test suite.

### Configuration ignored from the environment

For configuration sourced from YAML or environment variables, register a
default or explicit binding so Viper can reach the field. Update relevant tests
and deployment examples. Database-backed settings follow their own defaults,
persistence, and documentation contract.

### Database migration failure

Do not edit an applied migration to repair it. Restore the released content and
create a new compensating migration. See
[`backend/migrations/README.md`](backend/migrations/README.md).

### Forwarded client IP or proxy behavior

Review [`deploy/EDGE_SECURITY.md`](deploy/EDGE_SECURITY.md). Do not trust broad
proxy CIDRs or raw forwarded headers without an enforced edge boundary.

## Repository Layout

```text
sub2api-plus/
├── backend/        Go service, Ent schemas, migrations, and tests
├── frontend/       Vue application and en/zh locales
├── deploy/         Installers, Compose files, and operations documentation
├── docs/           Provider, protocol, and maintainer documentation
├── openspec/       Specifications for cross-cutting changes
└── tools/          Build entry point and generators
```

## Related Documents

- [Contributing](CONTRIBUTING.md)
- [Documentation index](docs/README.md)
- [Release process](docs/RELEASING.md)
- [Upstream mapping](UPSTREAM.md)

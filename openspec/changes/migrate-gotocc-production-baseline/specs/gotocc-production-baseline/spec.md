## ADDED Requirements

### Requirement: Plus release remains the complete implementation baseline

The candidate SHALL start from immutable Sub2API Plus release
`v0.1.173+custom.004` and SHALL preserve its repository identity, tracked file
history, dependency graph, migration history, license, notice, and upstream
mapping. The old production source tree SHALL NOT be merged as a parent or used
as a generated-code source.

#### Scenario: local behavior conflicts with a Plus core implementation

- **WHEN** both source trees implement the same core concern differently
- **THEN** the candidate retains the Plus implementation and architecture
- **AND THEN** integrates the approved local behavior through a current Plus
  extension point.

### Requirement: Legacy migration equivalence fails closed before execution

The migration runner SHALL evaluate the complete reviewed legacy-lineage state
after acquiring its advisory lock and before executing any new migration. It
SHALL record a Plus migration as equivalent only when the exact legacy
filename, legacy database checksum, current Plus file checksum, and resulting
schema all match one closed rule. It SHALL record all applicable equivalents
atomically and SHALL reject every mismatch before any new Plus migration is
executed.

#### Scenario: reviewed migration pair is equivalent

- **WHEN** the legacy filename has its reviewed checksum, the Plus filename is
  absent, the Plus file has its reviewed checksum, and the schema matches
- **THEN** the runner records the Plus filename without rerunning its SQL
- **AND THEN** normal migration execution continues.

#### Scenario: schema or checksum differs

- **WHEN** any reviewed filename exists with an unexpected checksum or its
  required schema differs
- **THEN** startup fails before executing or recording any new Plus migration.

#### Scenario: clean Plus database has no legacy filename

- **WHEN** neither filename in a reviewed pair exists
- **THEN** the compatibility preflight does not synthesize a record
- **AND THEN** normal Plus migration execution applies the target migration.

### Requirement: Existing sessions survive a compatible cutover

The candidate SHALL retain the persisted JWT secret and the existing refresh
token key and JSON contracts. Deployment procedures SHALL preserve the Redis
volume and SHALL NOT use `FLUSHDB`. Derived cache invalidation SHALL be bounded
to reviewed prefixes or normal expiration.

#### Scenario: old refresh token is read by the candidate

- **WHEN** an unexpired refresh token record was serialized by the production
  implementation and the JWT secret is unchanged
- **THEN** the candidate reads the record and can complete the normal refresh
  flow without requiring the user to sign in again.

### Requirement: Active GotoCC contracts remain usable on Plus

The candidate SHALL implement every active local contract, including reusable
invitation codes, Jingmeng/Grok video compatibility, one-click key import,
Images model passthrough, GotoCC homepage, team lifecycle and billing, Plus
Model Plaza compatibility, and durable image objects. Preserving old tables
without application read/write behavior SHALL NOT satisfy this requirement.

#### Scenario: existing local data is present after migration

- **WHEN** reusable invitation, team, or durable image object rows already
  exist
- **THEN** the corresponding authorized Plus API and UI can read and operate
  on those rows with the same ownership and billing boundaries.

### Requirement: Retired TokenFlux marketplace stays retired

The candidate SHALL keep Plus Model Plaza at `/model-plaza` and
`/api/v1/model-plaza` as the only model marketplace. It SHALL NOT restore
`ModelMarketplaceView`, the TokenFlux curated list or price conversion, or the
`/api/v1/marketplace/models` model-list API. LC-005 SHALL remain a retired audit
identifier and SHALL NOT be reused or removed by renumbering later contracts.

#### Scenario: compatibility route is used

- **WHEN** a browser opens `/models`
- **THEN** it reaches the Plus Model Plaza compatibility destination
- **AND THEN** no retired TokenFlux model-list API is invoked.

### Requirement: GotoCC homepage is adapted to Plus frontend architecture

The homepage SHALL retain GotoCC branding, business copy, public aggregate
statistics, provider/model summary, primary interactions, responsive behavior,
and the local `logo.png`. Its components, routing, state, localization, and
build integration SHALL follow the current Plus frontend. It SHALL NOT show
Plus default branding or retired TokenFlux branding.

#### Scenario: user opens the default homepage

- **WHEN** no administrator HTML or URL override is active
- **THEN** the desktop and 390px mobile views render the GotoCC homepage
- **AND THEN** public statistics contain only approved aggregate fields
- **AND THEN** text, controls, and media do not overlap or overflow.

### Requirement: Production mutation remains separately authorized

Candidate implementation and local verification SHALL NOT connect to or write
the production PostgreSQL database or Redis, change `.env`, deploy, restart a
production service, or push a remote branch. Those operations require explicit
separate authorization after an isolated clone rehearsal.

#### Scenario: local candidate reaches source-complete state

- **WHEN** all local source and test tasks pass
- **THEN** the candidate remains local and production remains unchanged until
  the separate approvals are recorded.

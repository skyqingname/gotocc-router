## ADDED Requirements

### Requirement: Active GotoCC contracts survive the v0.1.183 rebase

The candidate SHALL preserve LC-001 through LC-004 and LC-006 through LC-012
while retaining the Plus and official v0.1.183 behavior. LC-005 SHALL remain
absent.

#### Scenario: local candidate passes its release gate

- **WHEN** candidate verification completes
- **THEN** every active LC has a specific passing result
- **AND THEN** retired TokenFlux marketplace behavior is absent.

### Requirement: Model visibility follows live scheduling

Model-plaza membership SHALL come from schedulable accounts in the requested
group and platform. Pricing and wildcard defaults MAY enrich or expand that
inventory but SHALL NOT expose an inactive group or unschedulable concrete
platform.

#### Scenario: an active group has account models but no channel price

- **WHEN** the model plaza lists the group
- **THEN** its schedulable models remain visible without fabricated pricing
- **AND THEN** models from inactive accounts remain absent.

### Requirement: Migration lineage remains append-only

The candidate SHALL keep every deployed migration filename and byte sequence
unchanged and SHALL add Plus migrations 229-233 under their published full
filenames.

#### Scenario: the production database starts the candidate

- **WHEN** migration preflight accepts the deployed lineage
- **THEN** only unrecorded filenames 229-233 are applied once
- **AND THEN** interrupted concurrent indexes are verified before retry.

### Requirement: Immediate rollback has a bounded compatibility window

An automatic health-gate rollback MAY restore the deployed 0.1.178 binary
before operators write 0.1.183-only pricing or plugin data. Later rollback
SHALL preserve the matched PostgreSQL, Redis, configuration, binary, and plugin
artifact state and SHALL NOT drop forward migrations as an incidental action.

#### Scenario: the first post-start health gate fails

- **WHEN** no plugin or new multiplier operation has occurred
- **THEN** the deployment restores the frozen 0.1.178 binary
- **AND THEN** repeats the complete runtime and route verification.

### Requirement: Local upgrade remains non-production

The candidate SHALL pass `markers -> targeted -> full -> release`, and its
manifest SHALL state `NOT DEPLOYED` until a separate deployment confirmation.

#### Scenario: local release completes

- **WHEN** the release artifact and manifest are produced
- **THEN** no public snapshot, server file, database, cache, or service has
  changed.

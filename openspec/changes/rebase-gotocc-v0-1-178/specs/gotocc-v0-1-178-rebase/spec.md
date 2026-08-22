## ADDED Requirements

### Requirement: Active GotoCC contracts survive the v0.1.178 rebase

The candidate SHALL preserve LC-001 through LC-004 and LC-006 through LC-011
while retaining Plus v0.1.178 behavior. LC-005 SHALL remain absent.

#### Scenario: local candidate passes its release gate

- **WHEN** candidate verification completes
- **THEN** every active LC has a specific passing result
- **AND THEN** retired TokenFlux marketplace behavior is absent.

### Requirement: Reused migration prefixes remain independent

The candidate SHALL retain deployed GotoCC migration files without editing or
renaming them. New Plus files that reuse prefixes SHALL be tracked by their
distinct full filenames and exact checksums.

#### Scenario: production has recorded GotoCC 224 and 225 migrations

- **WHEN** the v0.1.178 candidate starts
- **THEN** the matching GotoCC filenames are skipped
- **AND THEN** the four distinct Plus migrations are applied once.

### Requirement: Image object keys remain private

The candidate SHALL store exact object keys only in server-controlled Redis and
PostgreSQL fields. Task and URL-renewal JSON SHALL NOT expose storage keys.

#### Scenario: a user completes and renews an asynchronous image

- **WHEN** the task result and renewed object URL are returned
- **THEN** both contain the durable object ID and usable URL
- **AND THEN** neither response contains `storage_key`.

### Requirement: Local upgrade hook is the release gate

The candidate SHALL pass `markers → targeted → full → release`. The release
manifest SHALL state `NOT DEPLOYED`; production migrations require a separate
explicit deployment confirmation.

#### Scenario: local release completes without production mutation

- **WHEN** all four local verification layers pass
- **THEN** the Linux/amd64 candidate package and manifest are produced locally
- **AND THEN** the manifest records `NOT DEPLOYED`
- **AND THEN** no production migration or service replacement occurs.

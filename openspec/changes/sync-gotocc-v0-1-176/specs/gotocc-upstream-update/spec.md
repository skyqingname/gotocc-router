## ADDED Requirements

### Requirement: GotoCC upgrade starts from an immutable Plus tag

The candidate SHALL start directly from Plus tag `v0.1.176+custom.001` and
SHALL preserve the complete Plus implementation before applying local semantic
changes. It SHALL NOT use old generated Ent, Wire, or embedded frontend output
as integration inputs.

#### Scenario: generated code is needed after integration

- **WHEN** GotoCC schema or provider definitions are installed on the target
  Plus source
- **THEN** Ent and Wire are regenerated from the candidate source.

### Requirement: Active local contracts survive the upstream update

The candidate SHALL preserve LC-001 through LC-004 and LC-006 through LC-010,
and SHALL keep retired LC-005 absent. Upstream features and fixes SHALL remain
present unless they violate a documented local ownership or security boundary.

#### Scenario: candidate completes local verification

- **WHEN** the release gate is evaluated
- **THEN** every active LC has a specific passing marker and targeted test
- **AND THEN** the retired marketplace page and model-list API remain absent.

### Requirement: API-key validation precedes scope resolution

The candidate SHALL validate quota and expiration inputs before creating
either personal or team API keys. Team scope SHALL still preserve actor,
billing-owner, and team attribution.

#### Scenario: invalid team-key quota is submitted

- **WHEN** a team-key request contains a negative, non-finite, or otherwise
  invalid quota or expiration value
- **THEN** creation fails before any team or API-key record is written.

### Requirement: Same-prefix migrations remain independently immutable

The candidate SHALL retain both `220_group_model_pricing.sql` and
`220_reusable_invitation_codes.sql` under their released filenames. Migration
identity SHALL remain the complete filename plus checksum; neither file SHALL
be renamed, rewritten, or synthesized as equivalent to the other.

#### Scenario: production has already applied local migration 220

- **WHEN** `220_reusable_invitation_codes.sql` is recorded and
  `220_group_model_pricing.sql` is not
- **THEN** the runner applies only the missing group-pricing migration under
  the advisory lock.

### Requirement: Production remains unchanged during candidate preparation

Candidate preparation SHALL NOT upload or deploy artifacts, restart services,
modify `.env`, write production PostgreSQL or Redis, or push remote refs.

#### Scenario: local release artifact is complete

- **WHEN** all local gates pass
- **THEN** the manifest states `NOT DEPLOYED`
- **AND THEN** production waits for separate rehearsal and deployment approval.

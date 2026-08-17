## ADDED Requirements

### Requirement: Active GotoCC contracts survive the v0.1.177 rebase

The candidate SHALL preserve LC-001 through LC-004 and LC-006 through LC-011
while retaining the v0.1.177 Plus implementation. LC-005 SHALL remain absent.

#### Scenario: local candidate passes its release gate

- **WHEN** the candidate verification completes
- **THEN** every active LC has a specific passing result
- **AND THEN** retired TokenFlux marketplace behavior is absent.

### Requirement: Same-prefix migrations remain independently immutable

The candidate SHALL retain the deployed GotoCC `221_add_teams.sql`,
`222_harden_team_lifecycle.sql`, and
`223_add_team_attribution_indexes_notx.sql` without renaming or editing them.
The release policy SHALL accept these paths only with their audited SHA-256.

#### Scenario: an upgraded production copy has recorded GotoCC migrations

- **WHEN** the v0.1.177 candidate starts
- **THEN** it skips the matching recorded GotoCC filenames
- **AND THEN** it applies the distinct upstream 221/222/223 filenames once.

### Requirement: Team email links only use trusted fallback origins

Team invitation, reissue, and ownership-transfer emails SHALL use the explicit
configured frontend URL when present. When it is absent, the public origin of
the configured `api_base_url` MAY be used. A request Origin MAY be used only
when it exactly matches a configured non-wildcard CORS origin or is an HTTPS
same-origin request with an exact request-host match. Wildcard CORS and an
untrusted Origin SHALL NOT become email-link destinations.

#### Scenario: production has api_base_url but no frontend_url

- **WHEN** `frontend_url` is empty and `api_base_url` is an absolute HTTP(S) URL
- **THEN** the invitation email link uses that URL's scheme and host.

#### Scenario: a same-origin owner sends an invitation without frontend_url

- **WHEN** `frontend_url` and `api_base_url` are absent and an HTTPS browser
  request has an Origin equal to the request host
- **THEN** the invitation email link uses that origin.

#### Scenario: a cross-origin request attempts to send an invitation

- **WHEN** `frontend_url` and `api_base_url` are absent and the Origin matches
  neither a configured non-wildcard CORS origin nor the request host
- **THEN** no invitation is created and the API reports that no frontend URL is
  available.

### Requirement: Local upgrade hook is the release gate

The candidate SHALL pass the existing Sub2API customization hook
`markers → targeted → full → release`. Isolated production-copy rehearsal is
not part of that hook and SHALL NOT block local packaging. Schema, data, or
cache-contract changes still require an explicit production-deploy
confirmation after the hook passes.

#### Scenario: local source gates pass

- **WHEN** the Linux/amd64 candidate package is built
- **THEN** its manifest states `NOT DEPLOYED`
- **AND THEN** production remains unchanged pending separate authorization.

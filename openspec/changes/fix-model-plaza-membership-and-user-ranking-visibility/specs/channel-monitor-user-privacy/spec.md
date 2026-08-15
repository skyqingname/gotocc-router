## ADDED Requirements

### Requirement: User ranking must be administrator-only in the channel monitor

The channel monitor SHALL render the user-ranking tab only for administrators.
Regular users MUST NOT see the entry or trigger its data request through legacy
query parameters.

#### Scenario: Regular user opens the channel monitor

- **WHEN** a regular user opens `/monitor`
- **THEN** the detail tabs MUST contain models and error reasons only
- **THEN** the page MUST NOT request the user-ranking endpoint

#### Scenario: Regular user opens a legacy ranking link

- **WHEN** a regular user opens `/monitor?tab=users`
- **THEN** the active detail tab MUST fall back to models
- **THEN** the URL MUST be normalized away from the user-ranking view
- **THEN** the page MUST request model details and MUST NOT request user ranking

#### Scenario: Administrator opens the ranking link

- **WHEN** an administrator opens `/monitor?tab=users`
- **THEN** the user-ranking tab MUST remain available and active
- **THEN** the page MAY request the administrator-visible ranking data

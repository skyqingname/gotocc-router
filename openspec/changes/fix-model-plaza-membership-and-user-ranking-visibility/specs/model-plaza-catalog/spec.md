## ADDED Requirements

### Requirement: Model Plaza membership must follow schedulable models

The system SHALL derive Model Plaza model membership from active groups with
schedulable accounts and the gateway's group model source. Pricing and mapping
records MUST NOT be the sole membership source.

#### Scenario: Schedulable group has no channel pricing

- **WHEN** an active group has a schedulable account and gateway models but no channel pricing entries
- **THEN** the Model Plaza MUST include the group and those models
- **THEN** missing prices MUST NOT remove the models

#### Scenario: Group has no schedulable accounts

- **WHEN** an active group has no schedulable account
- **THEN** the Model Plaza MUST omit that group even if stale pricing exists

### Requirement: Platform defaults and wildcards must be concrete

The system SHALL use the platform's built-in model list when the gateway has no
explicit account mapping. A suffix wildcard SHALL expand only to matching
models from that platform list, and the wildcard itself MUST NOT be exposed.

#### Scenario: No explicit account mapping

- **WHEN** a schedulable OpenAI group has no account model mapping
- **THEN** the Model Plaza MUST list the built-in OpenAI models

#### Scenario: Wildcard mapping

- **WHEN** a group model source contains `gpt-5.6-*`
- **THEN** the response MUST contain matching concrete built-in model IDs
- **THEN** the response MUST NOT contain `gpt-5.6-*`

### Requirement: Display pricing must use explicit precedence

The system SHALL enrich each available model using group custom pricing,
channel custom pricing, then bundled official pricing. Entries without any
actual price SHALL allow fallback to the next source. When no source contains a
price, the model SHALL remain visible with null pricing.

#### Scenario: Group custom price exists

- **WHEN** group and channel prices both match a model
- **THEN** the displayed paid price MUST use the group price

#### Scenario: Only official price exists

- **WHEN** an available model has no group or channel price and the bundled catalog contains it
- **THEN** the displayed paid price MUST use the bundled official price before the group multiplier

#### Scenario: No price exists

- **WHEN** an available model is absent from group, channel, and bundled pricing
- **THEN** the model MUST remain in the response with null pricing

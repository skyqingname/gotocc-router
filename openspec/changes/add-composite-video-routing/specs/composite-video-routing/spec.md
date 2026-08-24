## ADDED Requirements

### Requirement: OpenAI video aliases use the NewAPI canonical create path

The gateway SHALL accept `/videos` and `/videos/generations` for an OpenAI group and SHALL forward both to the account Base URL's canonical `/v1/videos` endpoint.

#### Scenario: Canvas submits the Grok-named model through the unified group

- **WHEN** an OpenAI-group API key posts `grok-imagine-video` to `/v1/videos/generations`
- **THEN** the request enters the OpenAI-compatible video handler
- **AND THEN** the upstream request path is `/v1/videos`.

#### Scenario: native Grok group creates a task

- **WHEN** a Grok-group API key posts a Grok video model to `/v1/videos` or `/v1/videos/generations`
- **THEN** the request enters the Grok video generation handler.

### Requirement: OpenAI video alias reads use the canonical task surface

The gateway SHALL normalize generation-alias task reads to NewAPI's canonical task and content resources.

#### Scenario: aliased task is polled

- **WHEN** an OpenAI-group API key gets `/v1/videos/generations/task-123`
- **THEN** the upstream request path is `/v1/videos/task-123`.

#### Scenario: aliased task content is downloaded

- **WHEN** an OpenAI-group API key gets `/v1/videos/generations/task-123/content`
- **THEN** the upstream request path is `/v1/videos/task-123/content`.

### Requirement: Unified grouping does not create a Composite video contract

The admin Composite route API and UI SHALL remain unchanged. Video model names exposed by the same NewAPI account SHALL NOT require a local provider route.

#### Scenario: administrator reviews route scopes

- **WHEN** an administrator opens Composite route controls
- **THEN** no new `videos` endpoint scope is present for this change.

### Requirement: Unified video pricing remains writable

The channel create and update API SHALL accept `billing_mode: "video"`, matching the billing mode already supported by the service and admin frontend.

#### Scenario: administrator saves a per-second video rule

- **WHEN** an administrator submits an OpenAI channel pricing entry with `billing_mode: "video"`
- **THEN** request validation accepts the entry and passes it to the channel service.

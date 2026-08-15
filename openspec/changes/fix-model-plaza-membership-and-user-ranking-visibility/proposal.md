## Why

The Plus Model Plaza currently derives model membership from channel mapping and
pricing declarations. A schedulable group therefore disappears when its
accounts can serve platform-default models but no channel price or mapping has
been entered. The user channel monitor also exposes a cross-user ranking tab to
regular users, despite that view being intended for administrators.

## What Changes

- Build Model Plaza membership from active groups with schedulable accounts and
  the same group model source used by the gateway.
- Expand configured wildcard models against the platform catalog and use the
  platform defaults when no explicit account mapping exists.
- Treat group pricing, channel pricing, and built-in official pricing as price
  enrichment only; an available model remains visible when none has a price.
- Hide the user-ranking tab from regular users, make legacy `?tab=users` links
  fall back to the model tab, and avoid the ranking request in that case.
- Keep the ranking available to administrators.

## Capabilities

### New Capabilities

- `model-plaza-catalog`: defines schedulable model membership and display-price
  precedence for the public Model Plaza.
- `channel-monitor-user-privacy`: defines role-based visibility and request
  behavior for the channel-monitor user ranking.

## Impact

- **Backend**: Model Plaza service composition, handler dependency injection,
  schedulable model aggregation, and focused unit tests.
- **Public API**: `/api/v1/model-plaza` returns more active groups and models;
  its response schema does not change.
- **Frontend**: `/monitor` tab availability and legacy query handling.
- **Persistent data and configuration**: none.
- **Production**: no deployment, restart, or data write is part of this change.

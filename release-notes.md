Sub2API Plus v0.1.176+custom.003

## Highlights

- Make Model Plaza membership follow schedulable group models, so usable
  models remain visible even when no channel price has been entered.
- Resolve displayed paid prices in group, channel, bundled official, then
  unconfigured order without removing unpriced models.
- Hide the channel-monitor user-ranking entry from regular users, including
  legacy `?tab=users` links, while retaining administrator access.

## Changed

- Expand suffix-wildcard model mappings only against platform defaults and
  deduplicate the resulting concrete model IDs case-insensitively.
- Preserve composite-platform price isolation and existing image-tier display
  pricing when enriching the schedulable catalog.
- Treat image-input-only custom prices as configured prices instead of
  incorrectly falling back to the bundled catalog.

## Compatibility and migration

- No database migration or environment-setting change.
- The Model Plaza response schema, exclusive-group visibility, user-specific
  multipliers, gateway `/models` routing, and administrator ranking behavior
  remain unchanged.

## Known issues

None.

## Upstream baseline

Official release: v0.1.176
Official commit: e803e3851c0a7e222cfadeafad7b8636ab959d11

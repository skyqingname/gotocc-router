# Model Plaza Catalog

Model Plaza is the public, group-oriented catalog served by
`GET /api/v1/model-plaza`. It shows which models a customer can use in each
active group and the prices that would be applied before the group or
user-specific multiplier.

## Membership

Model membership follows the live gateway inventory, not pricing records:

1. Load active groups that have at least one active schedulable account.
2. Read explicit model mappings from schedulable accounts in that group.
3. When no account has an explicit mapping, use the built-in model list for
   the group's platform.
4. Expand suffix wildcards, such as `gpt-5.6-*`, only against that platform's
   built-in list and return concrete model IDs.
5. Keep models with no configured or official price in the response with
   `pricing: null`.

Composite groups inspect their schedulable concrete platforms separately.
Model names are deduplicated case-insensitively in the same provider order as
the gateway, and channel pricing is looked up only in the selected concrete
platform.

## Display Pricing

For each available model, the `pricing` field uses the first source that
contains an actual price:

1. group model pricing;
2. channel model pricing;
3. the runtime official catalog, with the bundled catalog as its offline
   fallback;
4. `null` when none of the sources contains a price.

An empty group or channel pricing entry does not block the next source.
Image-input-only prices count as configured prices. Image models continue to
use the existing group image-tier composition, so group 1K/2K/4K overrides
and independent image multipliers match the billing display.

`official_pricing` remains a separate reference field. It does not determine
whether the model appears in the catalog.

## Visibility

- Anonymous visitors see non-exclusive groups when public access is enabled.
- Authenticated users also see exclusive groups explicitly granted to them.
- Authenticated users may receive `user_rate_multiplier` for a group.
- Missing pricing does not hide a group or model.

The response schema and `/model-plaza` route are stable. No database or
environment configuration is required for this membership behavior.

## Operational Checks

After changing account mappings, group pricing, channel pricing, or bundled
pricing data, verify:

- the group has a schedulable account;
- concrete wildcard matches appear and wildcard tokens do not;
- unpriced models remain visible with `pricing: null`;
- group prices override channel prices, and channel prices override official
  fallback prices;
- composite groups do not leak prices across concrete platforms.

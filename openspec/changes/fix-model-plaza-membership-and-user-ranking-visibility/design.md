## Model Membership

Model membership and pricing have separate responsibilities:

1. Load active groups and keep only groups with at least one schedulable
   account.
2. Ask the gateway model source for the group's currently configured models.
3. When the gateway returns no explicit mapping, use the built-in model list
   for the group's platform.
4. Expand suffix wildcards only against that built-in list, deduplicate without
   case sensitivity, and never return the wildcard token itself.
5. Keep an available model even when every pricing source is absent.

This restores the prior GotoCC catalog behavior while adapting it to Plus
`0.1.176` group-level model pricing.

## Display Pricing

For each live model, display pricing is resolved in this order:

1. matching group-level model pricing;
2. matching channel pricing;
3. bundled official pricing fallback;
4. `nil` when no source covers the model.

Group and channel entries with no actual price fields do not block the next
fallback. Image pricing is passed through the existing group image-tier
composition so the displayed per-image tiers and independent image multiplier
stay aligned with billing. Official reference pricing remains a separate field
in the response.

The existing public/exclusive visibility and user-specific multiplier rules
remain in the handler. The API schema is unchanged.

## User Ranking Visibility

The channel monitor exposes three detail tabs only to administrators. Regular
users receive only `models` and `errors` in the rendered tab list. Tab parsing
is role-aware: `users` is accepted only for an administrator. If a regular user
arrives through an old `?tab=users` URL, the active tab becomes `models`, the
URL is normalized, and the page loads the model endpoint rather than the user
ranking endpoint.

The backend ranking endpoint and administrator dashboards are outside this
change. This is an intentional UI entry and request-flow restriction, matching
the requested boundary.

## Compatibility

- No database migration or environment setting is introduced.
- Existing Model Plaza consumers continue to receive the same JSON fields.
- `/models` compatibility routing remains unchanged.
- Anonymous and authenticated exclusive-group filtering remains unchanged.
- Administrator ranking access remains unchanged.

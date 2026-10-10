# Per-key smart routing priorities

The administrator enables `allow_user_override` in the existing smart routing
policy dialog (Groups → Smart routing priorities). It defaults to false for
new installations and existing stored policies. `api-key-routing-defaults.json`
is the documented initial policy payload; the existing settings database owns
the runtime value. No environment variables or schema migration are needed.

Users can customize default ordering and exact/terminal-wildcard model rules
inside the API Key create/edit form. Drafts persist only when the form is saved.
Reset to administrator defaults writes a JSON null preference. Cancelling
discards the draft. Existing preferences survive toggling the permission off.

`GET /api/v1/keys/:id/routing-policy` returns permission, the saved preference,
and currently accessible groups. `PUT` accepts null or an object containing
`default_group_order` and `model_rules` only. Both require key ownership; writes
use the authenticated actor, require auto routing and an enabled permission,
validate all group permissions and reject malformed or oversized payloads.
Team keys use the existing team's payer group permissions.

User exact/longest-prefix model order precedes user default order, followed by
the administrator model/default order and stable group sort. Preferences only
reorder authorized candidates; they never grant access, change billing, skip
audit or introduce quota/error failover. Model catalogs and request routing use
the same policy. Required-group requests remain locked to the original group.
The toggle is read on every new resolution: disabling it ignores preferences
immediately for new resolutions, without interrupting existing pinned tasks.

Preferences reuse the indexed settings repository under
`api_key_routing_preference:<immutable key ID>`. They contain only model/group
ordering, never API key secrets. Deleted keys are inaccessible; their ordering
metadata is inert and IDs are never reused. No new repository interface or
generated Ent/Wire changes are required.

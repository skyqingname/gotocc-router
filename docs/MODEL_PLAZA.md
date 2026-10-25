# Model Plaza Visibility

## GoToCC billing credits

Site balances, quotas, price cards and usage costs display the **$** accounting
symbol. The numeric official price is preserved: USD 5 becomes $5 and CNY 2
becomes $2. The symbol denotes the site balance unit, not an exchange to USD. No USD/CNY conversion is applied. Per-token prices are
still scaled by 1,000,000 for the per-million display.

[`billing-credits.json`](../billing-credits.json) owns the internal unit, `display_symbol`, and maintained
CNY face-value cards. `python3 tools/generate_billing_credits.py` regenerates the
Go defaults and distributed pricing catalog. DeepSeek uses all-day peak prices:
2 input / 8 output / 0.04 cached input credits per million tokens. Its existing
Pro-to-Flash routing time remains in effect. Legacy Kimi K2/K2 Thinking and
Doubao embedding-vision cards retain their original CNY figures instead of the
previous converted USD figures. Other official USD cards keep their numeric
values; a provider having a domestic site does not implicitly select a different
regional price card.

Registered face-value defaults are used by billing and official plaza prices;
the models.dev snapshot cannot overwrite them with converted or off-peak prices.
Saved group/channel prices and effective multipliers retain precedence. Existing
balances, quota counters, historical usage and frozen task prices are not scaled
or rewritten. The legacy `*_usd` JSON/database field names remain unchanged,
but site amounts keep their existing numeric values. `/v1/usage` returns
`unit: "$"` for client display. Internal catalog `billing_unit: "credits"` remains
a machine identifier. Shared UI formatters and default notifications read the
symbol from the root parameter; payment receipts keep their actual currency.

Payment-provider charges and upstream-provider balance snapshots retain their
real currency. New recharge principal is 1:1 (USD 1 or CNY 1 adds 1 credit), and
subscription prices use their numeric face value in the selected payment
currency. Recharge multipliers and subscription FX are retired. Promotions and
fees remain separate items. Existing orders retain their frozen credited/pay
amounts and refund ratio; they are never repriced using the new policy.

## Group display translations

Administrators may set a group's `i18n` map with locale keys and optional
`name`/`description` fields. Keys use normalized BCP 47 casing; duplicate
normalized keys and names longer than 100 characters are rejected. Values
are trimmed and empty entries removed. An omitted map on update preserves
translations; an empty object clears them. Duplicating a group copies translated
descriptions but clears translated names so the new name remains distinguishable.

User-facing group, API-key, usage, subscription, redeem, checkout, model-plaza
and available-channel responses localize display text using `Accept-Language`.
Model Plaza appends `Vary: Accept-Language` while preserving CORS variation.
The highest-priority language matches the full locale,
then its primary language. Each missing translated field falls back to the
original field; a missing language does not select a lower-priority language.
User DTOs omit the translation map. Administrator forms retain the original
fields and map. IDs, access, routing, pricing, quotas and log identity are unchanged.

Changing the interface language refreshes localized read data without reloading
the document. Open forms, payment selections and unsaved drafts remain intact;
checkout refresh changes group display names only. Migration 286 supplies an
empty map for existing groups, preserving their current display until configured.

## Availability

Model Plaza is an optional group-and-pricing showcase at
`GET /api/v1/model-plaza`. It is disabled by default. An administrator must
enable it in System Settings before the route and header entry are available.

When enabled, sign-in is required by default. This remains true when
`model_plaza_require_auth` has no stored value. Anonymous visibility requires
an administrator to explicitly save `model_plaza_require_auth=false`; the
administrative UI presents an exposure warning before that setting is used.

## Data Exposure

Anonymous visitors see only non-exclusive groups, including their names,
models, pricing, and rate multipliers. A signed-in user may additionally see
exclusive groups listed in that user's `AllowedGroups` relationship. This is a
showcase relation, not authorization to call a model: it does not check an
active subscription or grant gateway access by itself.

An invalid supplied JWT is rejected by optional authentication middleware; it
does not silently fall back to an anonymous request. In backend mode, the
existing backend-mode user guard remains in effect. Global IP access control is
applied before this route and cannot be bypassed through optional JWT handling.

## Official reference refresh

The models.dev snapshot is frozen during preparation, bundled, and read offline.
The provider order in `model-plaza-catalog.json` selects Z.AI before Zhipu AI for
shared GLM IDs, matching the existing official USD billing cards. Zhipu AI's
converted USD prices do not replace the Z.AI native USD figures. Registered CNY
face-value cards continue to take precedence without currency conversion.

The 2026-10-25 snapshot has 298 provider entries (four added, twenty removed by
the source, 137 modified). Retiring a source entry does not remove configured
accounts, group membership, saved pricing, or historical usage.
`model-pricing-defaults.json` also updates the exact backend price cards for
`gemini-pro-latest`, Kimi K3/K3.1 Pro/Highspeed, GLM-4.5V and GLM-4.6V.
Kimi's 5-minute and 1-hour cache write prices follow its official pricing page.
Other changed models.dev prices remain plaza references; the snapshot does not
change billing defaults automatically.

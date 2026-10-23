# 0.2.17+custom.003 — PR review and combined release

Based on published GoToCC 0.2.17+custom.002 and Plus v0.2.17+custom.001 (official Sub2API v0.2.17).

- Login and registration show an input-adjacent service-terms hint. The terms dialog keeps keyboard focus inside and returns to the input after acceptance or rejection.
- Gateway redeem codes support one use, a shared-use limit, or unlimited users; each user can redeem a shared code once. Invitation codes, payment-order codes, fixed create-and-redeem codes and Canvas cards retain their single-use contracts.
- Shared redemption counts, balances, rebates and cashback are committed together. Existing redemption records survive deletion requests and cannot have their status, expiry or subscription group rewritten through batch editing. Synthetic affiliate-transfer history omits code-only usage limits. Omitted limits remain single-use at both HTTP and internal service entry points.
- Admin usage records show the public service host, including gotocc.xyz, api.gotocc.xyz and customer domains. Historical and detached asynchronous records without a captured host show a dash.
- Risk-control results distinguish reviewed pass/hit from unreviewed timeout, cancellation or unavailable endpoints. This includes the change accepted on 2026-10-20.
- Both binaries are built with Go 1.27.3 from the same fixed source and embedded frontend.

Migrations 310 and 311 add service_host, redeem-code limits and redemption usage records. After shared codes are created or redeemed, old binaries do not preserve their semantics: they can consume a shared code as a single-use code or omit shared history. Keep current data and use a forward fix; restoring an old dump would discard later activity.

No production installation or restart is included in this GitHub release. No paid upstream probes or test suites were run for this release.

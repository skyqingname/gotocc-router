# 0.2.18+custom.001 — Plus update, official prices and model fingerprints

Based on Plus `v0.2.18+custom.001` (`eb2f23d6`) and official Sub2API `v0.2.18`. Includes GoToCC PRs #41–#45 on top of the previous owned `0.2.17+custom.003` release.

- Integrates the latest Plus stream interruption and usage handling, Vertex account quotas, Gemini image permissions, scheduling snapshots and dependency updates. GoToCC retains its configured client identities, team wallets, automatic-key routing, local image storage and owned update channel.
- Site balances, quotas and model prices display `$` as the accounting symbol. Official numeric prices are preserved without USD/CNY conversion: USD 1 and CNY 1 each represent $1. Explicit group and channel prices remain higher priority.
- New recharge principal and subscription charges use their face value in the selected payment currency. Existing balances, frozen order amounts, currencies and refund semantics are not recalculated.
- Fixes risk-control hit lists and details when score maps are null, preserving the distinction between reviewed, unreviewed and unscored records. Restores validation of channel time-pricing periods.
- Refreshes the bundled model-plaza directory and six exact backend price cards, including Kimi K3.1 Pro/Highspeed, Kimi K3 cache TTL pricing and Gemini Pro Latest context tiers. GLM reference prices use Z.AI native USD values; maintained CNY face-value cards retain their precedence.
- Updates the offline model-fingerprint reference from 60 to 63 identities, adding Kimi K3.1 Pro, Gemini 3.5 Flash Lite and MiniMax M3.1 Flash. Includes the upstream sequence-shape correction and paired float32 reference artifact.

Migration 312 enables image generation for existing Gemini/Antigravity groups; composite groups keep their configuration. Existing migrations are unchanged. Preserve current data and use a forward correction after new payments or usage: replacing an older binary does not undo a data migration or restore the previous pricing contract, and restoring an older database would discard later activity.

Both Linux/arm64 and Linux/amd64 packages contain the same verified frontend and fixed source, built with Go 1.27.3. Local builds, type/static checks, price APIs, billing/payment fixtures, risk-control UI and offline fingerprint Worker checks were completed during integration. Fingerprint samples came from the upstream reference collection and are not an independent accuracy evaluation. No paid upstream probes or test suites were run for this release.

Publishing this release does not install or restart production.

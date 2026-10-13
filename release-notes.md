Sub2API Plus v0.2.15+custom.002

基于已发布的 `0.2.14+custom.004`，合入 Plus `v0.2.15+custom.001`（官方 `v0.2.15`，`f2669c8cf62555cd92389b3f55920e9e6e7c6ff2`）。保留全部 GoToCC 自有功能。

- Combine optional V1 active probes with V3 service status. Probe outcomes participate directly in V3 state and incident decisions while users see only the V3 monitor; the separate V1 user monitor remains available only in V1 mode. Probe requests follow the same credential-owner outbound identity contract as platform requests.
- Show deduplicated model status within each platform, without group or monitor names in the user-facing model and incident views. Healthy traffic cannot hide a failed probe.
- Improve the available-channel catalog with at most five cards per row, wider spacing and rounder corners. Service tiers such as Fast/priority, Flex and Ultrafast are collapsed by default and can be expanded independently.
- Add an optional external redeem-code purchase entry in payment settings. Recharge/subscription selection pages show its card at the bottom, with a right-aligned purchase action; operators choose an embedded page or a new tab. The built-in payment switch still controls recharge/subscription navigation.
- Let administrators opt balance and subscription redeem codes into order creation. Redemption atomically grants the benefit and creates a completed order, with separate payment/redemption source labels and filters and the existing balance/subscription order types.

- 同步官方 v0.2.15：新增 Cline、Command Code 兼容供应商接入及用量探测；平台选项、展示名与多协议账号表单统一由后端平台清单派生。
- 修正 OpenAI WebSocket 逐轮分组计价：长连接的后续轮次按当前分组价格计费和过利润门，与 HTTP 一致。智能 Key 逐轮路由与站长客户价格沿用原规则。
- 运维 TPS 分布改为非媒体单请求平均速率，提供样本数、平均值及 P5/P10/P50；既有吞吐量 `tps` 定义不变。
- 同步拒绝响应、历史函数调用、开发者角色、工具选择、思考签名等兼容处理，以及多处管理界面、异步状态和对话框修复。

- Balance redemption-order currencies are CNY and USD, defaulting to CNY, with no exchange-rate conversion: a face value of 100 and CNY order currency credits 100 USD and records a 100 CNY order.
- Subscription redemption codes freeze the selected plan's price, currency, name, group and duration when generated. Later plan price changes or deletion do not rewrite those snapshots.
- Payment and redemption orders share order management and statistics; totals remain separate per currency. Redemption orders have no provider payment, recharge promotion, fee or refund, including forced administrator refunds.
- New-tab purchases use opener isolation. Embedded purchases use a protected, sandboxed page and the configured origin in CSP, without adding application credentials or user identifiers to the third-party URL.

- 自有 Video 平台继续独立于上游平台清单，Video 分组、账号、调度和 `/v1/videos` 行为不变。
- 上游新增的 CI 验证工具不属于 GoToCC 构建流程，已加入 `upstream-removed-paths.json`。
- 构建镜像升级到 Go 1.27.2。
- 修复 Command Code / OpenCode 账号未显式设置协议时，编辑保存会把自动分流改为 Chat Completions 的问题。
- 修复渠道 Video 配置错误列出组合分组的问题，Video 继续只绑定 Video 分组。
- 修复 Cline 余额或订阅探测覆盖仍有效的组织限额等独立冷却原因，避免余额恢复后提前解除限额。

- Align the purchase button beside the purchase-card title and description, and label it "Go to purchase" / "前往购买".
- Use the shared plan-duration formatter in subscription administration so translated day/week/month labels replace raw translation keys.
- Preserve the disabled purchase entry and default new-tab mode in admin settings when payment configuration is unavailable.
- Invalidate subscription entitlement caches only after the redemption transaction commits; failed order or audit writes leave the code and benefits unchanged.

## Compatibility and migration

Upgrade applies forward-only migrations `282_redeem_order_snapshots.sql` and `283_channel_monitor_v3_probe_sources.sql`. Existing redeem codes default to no order, existing payment orders retain their payment source and currency resolution, and existing V3 observations keep their history and incident IDs as request observations.

Extra V1 probes in V3 and the external purchase entry default to disabled. Existing monitoring mode and the built-in payment navigation gate are preserved. Only balance and subscription codes support optional order creation; internal payment-fulfillment codes remain opted out. Redemption orders are non-refundable and receive no recharge promotions.

## Known issues

Third-party sites can prevent iframe embedding through their own browser security policies; the embedded purchase page provides a new-tab option.

V3 treats a missing probe or one older than five minutes as unknown, so longer probe intervals can leave unknown gaps. A probe targeting this application's gateway can also produce a request observation; the two sources are combined for status, without claiming deduplicated business-request counts across sources.

## Upstream baseline

Official release: v0.2.15
Official commit: f2669c8cf62555cd92389b3f55920e9e6e7c6ff2

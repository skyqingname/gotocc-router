# GoToCC 0.2.13+custom.004

基于 Plus `v0.2.13+custom.001`，在已发布的 GoToCC `0.2.13+custom.003` 上补齐 Codex 走 WebSocket 所需的核心行为，供入口打通 WebSocket 透传后使用；并新增影策画布的账号与余额桥接接口。

## Codex WebSocket

- WebSocket 连接按 `gateway.stream_keepalive_interval`（默认 10 秒）向客户端发送 ping，与 HTTP 流式 keepalive 使用同一参数。Cloudflare 对约 125 秒无流量的 WebSocket 直接断开，上游长时间推理和回合间隔都会触发；Codex 断线后先用 WebSocket 重试 5 次、每次重新整包发送，再回退 HTTPS。
- `mode_router_v2_enabled` 关闭时，账号显式配置的 WebSocket 模式 `http_bridge` 同样生效：客户端走 WebSocket，核心以 HTTP 请求上游。此前这类第三方中转账号被按 `ctx_pool` 直连上游 WebSocket，握手 404 后整轮失败。
- 首个 `response.create` 的等待时限默认 600 秒（原 30 秒），单条消息上限默认 256 MiB（原 64 MiB），与 HTTP 请求体上限一致。Codex 建连后立即发送带完整上下文的预热请求，大上下文在慢速上行下需要数分钟才能传完。

## 影策画布账号与余额桥接

- 新增管理员接口 `/api/v1/admin/canvas-bridge/*`，由影策画布前置的桥接服务凭管理员 API Key 调用：
  - `verify`：按邮箱和密码校验 GoToCC 账号，返回账号状态、是否站长客户，以及付款人（团队成员为团队负责人，其余为本人）。
  - `transfers`：按划转号从付款人余额单向划入画布，同一划转号重复提交只入账一次；余额不足、账号停用或团队暂停时拒绝；团队成员的划转计入其日/周/月额度。
  - `transfers/:id/reverse`：退回一笔划转，余额加回付款人并释放成员额度，只在画布侧无法入账时使用。
- 余额记录新增「影策画布划转」「影策画布划转退回」两种类型，用户与管理员的余额记录直接显示，备注写明对应的画布用户。
- 无数据库迁移，划转记录复用兑换记录表；审计日志对校验接口的密码字段脱敏。

## 上线顺序

- 本版本上线后再在 DMIT 入口 Nginx 透传 `Upgrade`/`Connection`。入口未改时，Codex 仍收到 426 并立即改走 HTTP，与此前一致。
- 无数据库迁移、无 Redis 格式变化、无账务变化。回退旧程序前先撤回入口的 WebSocket 透传。生产由用户通过自有更新通道安装，发布不等于已上线。

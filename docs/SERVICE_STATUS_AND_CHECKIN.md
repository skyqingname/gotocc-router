# 签到中心与服务状态

## 签到中心 `/checkin`

每天签到一次，奖励立即加到个人余额（`users.balance`，走现有 `AdjustBalance`，不计入累计充值）。本自然月累计签到满 7 / 14 / 30 天的那一次签到，当天额外再得里程碑奖励；按自然月累计，不要求连续。日期和月份边界按站点时区（`timezone` 配置）。

开启条件：本自然月内有过任意金额的入账，来源是 `redeem_codes` 中 `type` 在参数列表里且 `value > 0` 的单次兑换与管理员加余额（记在 `used_by`/`used_at`）、共享兑换码的兑换记录（`redeem_code_usages`），以及状态在参数列表里的支付订单（`payment_orders.completed_at`）。站长客户的额度由站长管理，接口返回 `CHECKIN_UNAVAILABLE`，前端路由在 `reseller-defaults.json` 的 `customer_hidden_routes` 里。

规则全部在根目录 `daily-checkin.json`：`daily_reward`、`milestones[]`（`days`、`reward`）、`unlock.redeem_code_types`、`unlock.payment_order_statuses`。修改后执行 `python3 tools/generate_daily_checkin.py`，生成 `backend/internal/service/daily_checkin_config_gotocc.go`。

| 方法与路径 | 行为 |
| --- | --- |
| `GET /api/v1/checkin?month=YYYY-MM` | 本月是否开启（`unlocked`、`lock_reason`）、今天是否已签、今天签到能拿多少；`month`（默认本月，不能晚于本月）每天的签到与金额、该月天数与所得、累计所得、三个里程碑的达成、还差几天和本月是否还来得及 |
| `POST /api/v1/checkin` | 签今天；返回本次奖励、触发的里程碑、本月第几天、新余额。未充值 `CHECKIN_RECHARGE_REQUIRED`（403），今天已签 `CHECKIN_ALREADY_DONE`（409） |

一次签到在同一事务里写 `daily_checkins` 并加余额，提交后失效该用户的认证缓存与余额缓存。`(user_id, checkin_date)` 唯一，同一天的并发请求只有一次写入，其余返回 409。

迁移 `346_daily_checkins.sql` 新建 `daily_checkins` 表（只新增，不改旧表）。

## 服务状态 `/monitor`

按分组的被动检测，只看真实用户请求，不发探测，也不依赖渠道监控开关（管理员的渠道监控页与它无关）。原 V1/V2/V3 三种用户展示和「可用渠道」页已删除，`/available-channels` 重定向到 `/monitor`；每张分组卡带倍率和可用模型。

口径（窗口内，按请求的最终结果）：

- 成功：一条用量记录（`usage_logs`），中途换号后成功的也算；中途断流（`completion_status=incomplete`）算失败。客户端主动断开（`client_disconnected`）和被上游内容策略拦下（`request_type=4`）的不计入。
- 失败：客户端收到错误（`status_code >= 400`）且归因为平台或上游（`error_owner` 为 `provider`/`platform`）的运维错误记录（`ops_error_logs`），不含业务限制（余额、并发、参数、鉴权、内容审核等）；唯独「没有可用账号」虽被标成业务限制，阶段是 `routing`，仍算平台失败。同一请求号已有用量记录的错误是换号前的中间失败，以最终结果为准；同一请求号的多条错误只算一次。
- 当前状态看最近 N 次请求的成功率：≥ 正常阈值为「正常」，≥ 波动阈值为「波动」，更低为「异常」，窗口内没有请求为「暂无请求」。
- 状态条每格一个时段，最后一格是当前时段；格子按该时段的成功率用同样的阈值着色。
- 首字中位数只看成功请求；缓存命中率 = 缓存读 ÷（新输入 + 缓存读 + 缓存写），没有输入时显示「—」。

参数在根目录 `service-status.json`：`recent_requests`、`normal_success_rate`、`degraded_success_rate`、`bucket_count`、`bucket_minutes`、`cache_seconds`（服务端全站共用的统计缓存）、`refresh_seconds`（页面自动刷新）。修改后执行 `python3 tools/generate_service_status.py`，生成 `backend/internal/service/service_status_config_gotocc.go`；前端直接读取根 JSON。

| 方法与路径 | 行为 |
| --- | --- |
| `GET /api/v1/service-status` | 当前用户能用的分组（与可用渠道同一套可见性，专属分组只给被授权的人）及各自状态、倍率、指标和状态条 |
| `GET /api/v1/service-status/groups/:id/models` | 该分组可调用的模型，取自可用渠道同一份目录 |

倍率是该用户实际生效的倍率（含用户专属倍率）。站长客户拿到的分组已换成站长给他的客户价，不含站长成本。管理员代看用户时两个接口也在 `/api/v1/admin/support/users/:user_id/` 下提供只读版本。

# 内容审计在线测试

管理员在内容审计页的 Worker 卡片右侧输入文字、选择已保存的审核节点并点击「开始测试」。前置拦截及关闭模式下，同一入口显示在审核记录上方。

- 使用已保存的协议、模型、代理、超时、Jev 审核标准及分类阈值。设置弹窗中未保存的草稿不参与测试。
- 每次选取所选节点的一枚已保存 Key 直接调用，不执行池内故障转移或重试。暂停/冷却节点与冻结 Key 仍可诊断；显示本次 Key 的掩码。
- 成功时展示每个返回类别的分数、对应阈值和命中状态。判定复用 `evaluateModerationScores`，不使用供应商的 `flagged` 替代本站阈值。
- 「命中阈值」与「当前允许模型前置拦截」分别展示。后者依据系统风控开关、内容审计开关、节点启用/暂停状态、全局模式、文本 API 模式和关键词策略计算。真实请求仍受分组/模型范围、抽样、节点可用性及白名单影响；本测试不模拟关键词、历史 hash 或会话封禁。
- 失败展示 HTTP 状态（未收到响应时为 0）、耗时和错误全文；HTTP 非 2xx、网络/超时、响应解析失败及缺少评分都显示未评分，绝不伪造为 0 分通过。响应原文只作文本渲染。
- 不写入审计记录、违规计数、hash/会话封禁、邮件或用户处罚，不更新节点/Key 健康状态，不保存本次输入；后台操作日志只记录调用元信息，请求体沿用既有省略机制。测试会实际调用上游，可能产生供应商费用。

## 管理接口

`POST /api/v1/admin/risk-control/test-text`，使用既有管理员认证。

```json
{"endpoint_id":"saved-endpoint-id","prompt":"需要审核的文字"}
```

纯空白或超过既有送审上限的文本返回 400，不静默截断；不存在的节点返回 404，未配置 Key 返回 400。正常调用返回标准 `code/data` 信封，`data` 包含：

| 字段 | 含义 |
| --- | --- |
| `endpoint_id`, `endpoint_name`, `model`, `api_key_masked` | 实际调用目标与 Key 掩码 |
| `http_status`, `latency_ms` | 上游 HTTP 状态和调用耗时 |
| `blocking_enabled` | 已保存配置是否允许此节点对范围内文本请求做模型前置拦截 |
| `audit_result` | 成功时的分类评分、阈值快照、最高分与阈值命中；失败为 null |
| `error` | 失败详情；成功为空字符串 |

OpenAI Moderations 和 TypeSafe Jev 共用原协议调用。错误响应完整读取；非 2xx、非 JSON、空评分/缺失 Jev 答案保留响应原文。已有健康状态摘要仍按原逻辑处理，在线测试不写这些摘要。调用沿用 `callModerationOnceWithInput` 的可信出站身份、已保存代理与超时。浏览器不再用通用 30 秒限制截断这条请求，离开面板会取消调用。

上游调用失败时输出 `content_moderation.online_test_failed` 结构化日志，包含请求ID、固定接口路径、审核协议、阶段、稳定错误分类、输入字节数、HTTP状态与耗时。日志不包含输入、Key或上游错误正文；错误原文仍只返回管理员页面。

实现：`content_moderation_online_gotocc.go`（service/handler）、`ContentAuditOnlineTest.vue`、`riskControl.ts`。无数据库迁移，无新增业务参数，无发布版本变更。

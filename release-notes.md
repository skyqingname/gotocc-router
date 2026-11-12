# GoToCC 0.2.20+custom.003

## Responses 压缩兼容性

- OAuth / setup-token Responses 的最终 HTTP/plugin 出站 Zstd 窗口限制为 512 KiB，保留等级 3，修复已复现的大图片请求通用 400。
- 压缩等级、窗口、生成器和 Go 常量配套管理。JSON、工具历史、身份来源、审核与资金流程保持原语义；不新增自动重试或未压缩回退。
- 同一合成请求原编码两次返回 400，修改后两次完整成功；这不证明所有历史 400 同因。重复大图片请求的传输体积可能增加，原失败用户会话未重放。

## Claude advisor

- 原生 Messages 保留 `advisor-tool-2026-03-01`、`advisor_20260301` 工具定义、`server_tool_use(name=advisor)`、`advisor_tool_result` 和 `usage.iterations`，advisor 模型名与加密内容不受主模型/工具名改写影响。
- 每个 `advisor_message` 按自己的模型价格与冻结分组倍率独立记录和扣费；主请求仍只按外层 usage 计费。保留 Actor、个人/团队/客户付款主体和平台成本语义。
- 子记录关联父请求及段索引，复用原子幂等结算。缺少 advisor 价格时按主请求实际选定价格模型收费并告警；缺失 iterations（包括中途断连）不新增 advisor 错误或费用。
- 现有管理员 beta 策略和账号 header 覆写仍有效；供应商支持需按具体渠道验证。只返回普通 `tool_use` 的供应商不视为完成 advisor 协议，也不会生成虚构 advisor 账单。

## 验证与兼容

- advisor 13 个顶层测试、17 个含子用例结果通过；相关回归 264 个通过结果。另有 4 条既有 DeepSeek 金额断言失败，已在未修改的 `.002` 基线复现，未修改该部分逻辑或断言。
- 真实独立 PostgreSQL 验证主请求/多段 advisor 分别入账、缺价回退、关联及重复结算余额不变；真实供应商流式/非流式均返回 advisor 内容与正确模型，主/子费用经独立计算核对。
- 同一固定源码和既有前端提供真实 Linux ARM64、AMD64 包。本版无新增依赖、数据库迁移或后台配置开关，默认价卡保持 `.002` 原样。
- 上游基线保持 `LuckyKuang/sub2api-plus@v0.2.20+custom.001 / c54cae37813b21077afeefcf9b252a54d052380c`。
- 未验证供应商的 advisor 兼容性、原失败会话恢复和生产自然 400 变化仍待观察。正式发布不代表已安装到生产；本轮不执行生产部署或重启。

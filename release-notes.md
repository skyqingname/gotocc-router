# GoToCC 0.2.18+custom.005

- Chat Completions 经 Responses 上游转换时新增 `gotocc_timing.forward_duration_ms`：本站从转发起点至最终 JSON/结束块序列化前实测的链路耗时，包含上游等待和传输，不改既有用量计时或计费。
- 上游明确提供响应创建、完成时间戳时，独立返回 `gotocc_timing.upstream_response`，包含原始 `created_at`、`completed_at` 及二者差值 `duration_ms`。缺少或倒序的时间戳不产生上游历时，不以本站测量补齐。
- 普通 JSON 同时返回对应 `Server-Timing` 指标；流式在 finish chunk 返回耗时对象，不在初始响应头中预填尚未发生的完整耗时。

两组时间来源独立。供应商整秒时间戳换成毫秒单位并不意味着毫秒精度，其响应生命周期也不代表纯模型推理时间；本版不填写或冒充 `openai-processing-ms`。原生 Responses 和 raw Chat 直转协议保持原样。

验证：ARM64/AMD64 候选构建、相关 Go 静态检查和生成器只读检查通过。普通/流式回放输出通过；一次授权真实请求 HTTP 200，本站实测25.674秒，上游时间戳差1秒，两者分别输出，原始时间戳、用量和过滤数组核对一致。该数值属于本地候选验证链路，不代表生产性能。未新增或运行测试套件，也未追加收费调用。

保留 `.004` 的零值用量明细、Azure `content_filters` 和管理员使用记录布局修复。Plus 上游基线仍为 `v0.2.18+custom.003` / `84e21908d1651f0f1482587844ce76ef82c935ef`。无新增数据库迁移、账号配置或计费策略变更。双架构包由同一固定源码和前端生成；发布不包含生产安装或重启。

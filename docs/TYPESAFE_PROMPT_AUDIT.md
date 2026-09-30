# TypeSafe Jev 提示词审核

在“安全审计 → Prompt Audit”选择 `TypeSafe Jev · Noul`。填写 TypeSafe 节点的 Base URL、专用 API Key、`jev-latest` 模型、超时与单段输入上限，选择风险类别、阈值和分组范围，保存后使用文本审核区验证。默认地址、模型、策略和输入参数集中在根目录 `prompt-audit-defaults.json`。

依据 [TypeSafe API](https://docs.typesafe.ai/api) 与 [模型文档](https://docs.typesafe.ai/models)，调用 `POST /v1/systemone`，使用 Bearer 认证。待审文本只放在 `state.content`；审核策略及每类判断问题放入 `questions`。一次请求同时提交全部启用类别的 Noul 问题，读取 `answers.<类别>.noul`。

任一类别概率达到 `confidence_threshold` 即按原有阻断策略处理，默认 0.8。Noul 是该类别成立的概率，不是 Choice/Score 的 confidence。为兼容事件结构，最大类别概率仍保存在现有 confidence 字段；文本测试显示“最高风险概率”，原因只列达到阈值的类别，全部低于阈值时明确显示未命中，不伪造模型生成的解释。事件 `scanner_version` 记录 API 实际返回的模型版本，文本测试也显示该版本。

同步、异步、节点探测与文本审核共用同一个实现，沿用原来的提取、分段、优先级、每节点/每段超时和聚合。HTTP 401/422/429/5xx、超时、缺失类别或非法概率均保留为调用失败，不冒充审核通过。这里的 API 适配不证明用户实际审核政策的准确率，须用用户样例进行人工验收。

## 旧配置切换

已有 Qwen3Guard/评分 JSON 配置保留原协议，避免升级时将旧供应商密钥发往 TypeSafe。选择 Jev 保留已有自定义审核标准，仅在使用内置默认模板时换成 Jev 默认标准；它会在编辑草稿中把旧节点转换为 TypeSafe、填入新默认地址和模型，并清空旧节点凭据；用户填写 TypeSafe Key 后保存才生效。新安装无配置时默认 Jev。线上启用由管理员明确操作，发布代码不自动改生产审核设置。

官方资料：
- [介绍](https://docs.typesafe.ai/introduction)
- [API Reference](https://docs.typesafe.ai/api)
- [Primitives](https://docs.typesafe.ai/primitives)

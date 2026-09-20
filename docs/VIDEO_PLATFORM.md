# Video 平台

Video 是独立的分组与账号平台，账号类型为 API Key，供应商地址必须明确填写。Video 分组同时支持 Video 账号和既有 OpenAI 兼容 API Key 账号。它复用既有异步任务、账务预占、原账号轮询和成功交付后结算。既有 OpenAI 视频分组和历史任务继续使用原协议。

## 管理配置

1. 创建 Video 分组。在“添加账号”选择 Video，填写供应商地址与 API Key，再绑定 Video 分组；账号列表和筛选显示 Video。已有 OpenAI API Key 账号仍可直接绑定或复制到 Video 分组。
2. 在渠道管理添加 Video 平台，绑定该组。
3. 在“Video 模型与协议”逐个添加公开模型名、上游模型名、启停和调用协议。
4. 在同一渠道下方的模型定价中配置该模型的按次或按秒价格及分辨率档位。

每个渠道独立保存 `features_config.video_models`。模型选项的初始值来自根目录 `video-platform-defaults.json`，保存后的完整配置是运行真源。

支持两类协议：

- `openai`：JSON/multipart 视频请求保留原正文与文件，支持配置创建、查询和内容路径。
- `custom_json`：客户端使用 JSON 及公开媒体 URL，配置 canonical 字段到上游 JSON 点路径的映射，例如 `prompt → input.prompt`、`seconds → parameters.duration`。不包含需要厂商签名 SDK 或可执行插件的协议。

默认参数只填充未传入的值；可按模型声明参数类型、必填和枚举允许值。数值范围可通过同一配置中的 `min/max` 声明。公开名与上游模型名不同时，报价遵循渠道设置的请求模型/渠道映射模型/上游模型计费基准。计费仍读取 canonical `seconds` 和 `resolution`，应给它们配置明确值或默认值，再映射到供应商字段。模型映射由公开模型的 `upstream_model` 确定，不根据名称猜测协议。Video 账号不预置文本模型；按需手动填写上游模型白名单，供应商支持 /v1/models 时也可同步。账号菜单不提供文本连通测试或文本定时测试；视频生成验证使用已配置渠道的 Videos API。

路径从账号 API 站点根地址开始，显式包含所需前缀；查询/内容路径中的 `{task_id}` 自动替换。字段路径指定任务 ID、状态和结果视频 URL。创建响应省略状态时由明确配置的 `create_status` 指定状态，未配置则拒绝不完整响应；供应商状态映射为 `pending|processing|completed|failed|cancelled|expired`。

若供应商没有 content 接口，留空 `content_path` 并设置 `video_url_field`。内容获取时重新查询任务获得当前结果 URL，再读取媒体；账号凭据不会传给媒体存储域名。只验证媒体可开始读取，不声称已检查整个文件。

## API 与任务

统一入口仍为 `POST /v1/videos`、`GET /v1/videos/:id`、`GET /v1/videos/:id/content`，以及既有别名。`GET /v1/models` 返回该渠道已启用、且有可调度 API Key 账号支持的公开模型。智能 Key 的视频路由支持 Video 分组，其他协议不借此进入视频组。

原始 canonical 提示词完成审核后，才填充无内容默认参数、适配请求、选择账号、预占及提交。创建前校验参数与价格；任务保存原账号、模型、价格及完整 provider 配置。迁移 276 添加 `openai_video_tasks.provider_config`；旧行 NULL，沿用原 OpenAI 实现。修改渠道配置不会改变旧任务的查询方式。

余额任务先预占，成功且内容可读后捕获；失败/取消/过期释放。订阅仅成功终态消费。任务归属、Key/团队额度和幂等规则沿用现有视频生命周期。平台配额沿用 OpenAI 兼容账号计量。

设计参考：[影策后端](https://github.com/ddcat-ai/open-ai-canvas)，采用渠道、模型配置、协议请求与归一化结果的分层；未复制其可执行插件运行时。

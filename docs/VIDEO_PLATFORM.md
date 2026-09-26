# Video 平台与影策视频协议

## 配置入口

在“分组管理 → 创建/编辑 Video 分组 → Video 模型与协议”配置。为每个公开模型选择影策协议、上游模型及参数。协议目录可按厂商、名称和请求路径筛选；接口选项展示完整路径，包括 `/v1/videos`、`/v1/video/generations`、`/v1/videos/generations` 及其他厂商路径。

选中协议后带入影策的请求、查询、结果解析、鉴权及参数定义。常规参数支持默认值、允许值、启停、类型、数值范围和步长；协议扩展项放在按协议 ID 分组的 `provider_options` 对象中。工作流 ID、Google 项目和地区、厂商 Action/Version 等按该协议接口说明填写，不猜测账号或模型的能力。高级接口定义可针对该分组模型覆盖 create/poll/result/auth/response，修改后保存才生效。查看只读说明不需要提交。

账号管理选择 Video，填写上游地址与 API Key；腾讯/火山签名协议还需要 Secret Key。SecretId / Access Key 填在 API Key 栏，Secret Key 有独立输入框，查询只返回已配置标志。Video 分组也兼容原 OpenAI API Key 账号。账号绑定决定该组使用的渠道凭据；渠道定价页保留价格设置，协议配置以分组 `video_models` 为唯一真源。

原 OpenAI JSON/multipart 和自定义 JSON 字段映射配置继续可用；已有渠道配置由 migration 278 复制到其关联 Video 分组，不重写已创建任务的 provider_config。

## 继承来源与运行

影策来源锁定为 `ddcat-ai/open-ai-canvas` commit `b8eefdad709b192ba9686e3ffa76dbd0512b1ca2`。导入其全部 39 个声明 video capability 的官方 provider；`backend/internal/pkg/yingceprotocol` 是其协议表达式与解析引擎，`video-protocol-catalog.json` 保存原始定义。MIT 授权保留在 `LICENSE.yingce`。火山 V4 使用官方 SDK `v1.0.253` 的独立签名实现，Apache 授权保留在 `LICENSE.volcengine`；腾讯 TC3 沿用影策实现。运行时无需安装 SDK、拉取插件或下载协议。

从已固定的影策源码更新目录：

```bash
python3 tools/import_yingce_video.py --source /absolute/path/to/pinned/open-ai-canvas --commit PINNED_COMMIT
```

该脚本同步声明式引擎、协议目录和编译资源。`video-protocol-url-defaults.json` 保存影策的版本前缀规则；`originPath` 使用站点根路径，否则遵循影策 Base URL/显式版本前缀的拼接语义。导入新来源后仍须适配宿主传输、参数、身份和账务契约并重新构建。

影策协议的创建与查询返回本地 `video-local:…` ID，原供应商 ID 仅用于原账号上游查询。Veo 等层级资源任务名在上游路径中保留路径段，客户端不用把带斜杠的供应商 ID 拼到网关路由。

公共入口统一为 `/v1/videos`、`/v1/videos/:id` 和 `/v1/videos/:id/content`，支持原复数 generations 别名及新增 `/v1/video/generations` 创建/查询/content 别名。别名在鉴权、审核及路由前归一到同一视频任务入口。

客户端可使用 canonical `seconds/aspect_ratio/generate_audio`，也接收影策对应的 `duration/aspectRatio/generateAudio`。`images/videos/audios` 接收 URL 或带 role/order 的素材对象，兼容 `image_urls/video_urls/audio_urls`。multipart 文件转换为协议所需引用；要求公网 URL 的厂商需由客户端提供可访问 URL，不包含新增的对象存储托管服务。

审核继续先于账号选择、预占和上游请求。提交时冻结选中协议完整模板、模型、非素材轮询参数及原账号；创建、轮询、结果下载均使用该快照。HTTP 请求使用既有传输和账号身份，再按协议添加 Bearer、专用头、Google API Key、TC3 或火山 V4 签名。公共存储域名不接收账号凭据；同站点受认证结果接口按协议鉴权。API Key/签名密钥不放入协议快照。

余额先预占，成功且结果可读后捕获；失败/取消/过期释放。订阅成功终态才消费。定价、站长差价、归属、Key/团队额度及幂等沿用现有任务生命周期。导入接口定义与完成真实上游调用是不同证据：每家实际模型、余额、权限和参数仍以该渠道为准，未配置凭据的协议不声称已通过实调用。

## Migration 278

在 277 后追加 `groups.video_models JSONB NOT NULL DEFAULT '{}'`，将原关联渠道的 video_models 复制到 Video 分组。原 channel JSON 保留仅供历史追溯；新运行时和编辑页均读取分组。复制分组同时复制配置；认证快照版本更新以重新载入分组协议。

DDL 短时锁 groups；回填产生 JSON 数据和 WAL，增量与既有 Video 配置数量相关。旧任务无需回填，继续读取原 provider_config。旧程序不理解分组协议和新的影策任务快照，启用后不能仅回滚二进制；保留数据前滚修复，或停全部 writer 后按另行确认的匹配集合恢复。不得用旧 dump 覆盖新增业务数据。

## 已导入协议目录

| 协议 | 创建接口 | 鉴权 |
| --- | --- | --- |
| Agnes Video 2.5 / 2.5 Flash (`agnes-video`) | `POST /videos` | `bearer` |
| Agnes Video V2.0 (`agnes-video-v20`) | `POST /videos` | `bearer` |
| AutoDL ComfyUI 视频 (`autodl-comfyui`) | `POST /api/v1/comfyui/comfyui_workflow/{{model}}` | `header` |
| 百度千帆视频 (`baidu-video`) | `POST /v2/video/generations` | `bearer` |
| DashScope Wan Video (`dashscope-wan-video`) | `POST /api/v1/services/aigc/video-generation/video-synthesis` | `bearer` |
| DashScope Wan 3.0 Video (`dashscope-wan3-video`) | `POST /api/v1/services/aigc/video-generation/video-synthesis` | `bearer` |
| fal.ai Queue Video (`fal-queue-video`) | `POST /{{model}}` | `bearer` |
| Google Gemini Veo (`gemini-veo`) | `POST /v1beta/models/{{model}}:predictLongRunning` | `google-api-key` |
| Kling Video (`kling-video`) | `POST /v1/videos/generations` | `bearer` |
| Luma Dream Machine (`luma-dream-machine`) | `POST /dream-machine/v1/generations` | `bearer` |
| 万有引力 Wan 3.0 视频（现有渠道） (`lxmone-wan-videos`) | `POST /v1/videos` | `bearer` |
| 万有引力 Wan 3.0 视频（S 渠道） (`lxmone-wan-channel-s`) | `POST /v1/videos` | `bearer` |
| 万有引力 Seedance 视频 (`lxmone-seedance-videos`) | `POST /v1/videos` | `bearer` |
| 万有引力 MiniMax H3 Max 视频 (`lxmone-h3-max-videos`) | `POST /v1/videos` | `bearer` |
| 万有引力 SD 视频 (`lxmone-sd-videos`) | `POST /v1/videos` | `bearer` |
| 万有引力 SD Mini 视频 (`lxmone-sd-mini-videos`) | `POST /v1/videos` | `bearer` |
| 万有引力 Grok Imagine 视频 (`lxmone-grok-videos`) | `POST /v1/videos/generations` | `bearer` |
| 万有引力 MiniMax H3 工作流 (`lxmone-h3-workflow`) | `POST /v1/videos/generations` | `bearer` |
| METASO MiniMax H3 (`metaso-h3`) | `POST /api/minimax/v2/video_generation` | `bearer` |
| MiniMax Hailuo Video V2 / H3 (`minimax-video`) | `POST /v2/video_generation` | `bearer` |
| NewAPI Media Task Channel 1 (`newapi-channel-1`) | `POST /v1/videos` | `bearer` |
| NewAPI Video Generations Channel 2 (`newapi-channel-2`) | `POST /v1/video/generations` | `bearer` |
| Novita Video (`novita-video`) | `POST /v3/video/create` | `bearer` |
| OpenAI Videos / Sora (`newapi`) | `POST /v1/videos` | `bearer` |
| Pika via fal.ai (`pika-via-fal`) | `POST /{{model}}` | `bearer` |
| PixVerse Video (`pixverse-video`) | `POST /openapi/v2/video/text/generate` | `bearer` |
| Replicate Predictions Video (`replicate-prediction-video`) | `POST /v1/predictions` | `bearer` |
| RollDek WAN 3.0 Video (`rolldek-wan-video`) | `POST /v1/videos` | `bearer` |
| RunningHub Workflow (`runninghub-workflow`) | `POST /task/openapi/create` | `bearer` |
| Runway Video (`runway-video`) | `POST /v1/image_to_video` | `bearer` |
| Seedance Compatible /videos (`seedance-videos-compatible`) | `POST /v1/videos` | `bearer` |
| 腾讯混元视频 (`tencent-hunyuan-video`) | `POST /` | `tc3` |
| Vertex AI Veo (`vertex-veo`) | `POST /v1/projects/{{request.providerOptions.vertex-veo.project}}/locations/{{request.providerOptions.vertex-veo.location}}/publishers/google/models/{{model}}:predictLongRunning` | `bearer` |
| Vidu Video (`vidu-video`) | `POST /v1/videos` | `bearer` |
| Volcengine Ark Agent Plan Seedance (`volcengine-ark-agent-plan-video`) | `POST /api/plan/v3/contents/generations/tasks` | `bearer` |
| Volcengine Ark Seedance (`volcengine-ark-video`) | `POST /api/v3/contents/generations/tasks` | `bearer` |
| Volcengine Jimeng Video (`volcengine-jimeng-video`) | `POST /` | `volcengine-v4` |
| xAI Video (`xai-video`) | `POST /v1/videos/generations` | `bearer` |
| 智谱 CogVideoX (`zhipu-cogvideox`) | `POST /v4/videos/generations` | `bearer` |

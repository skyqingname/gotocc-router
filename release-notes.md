# GoToCC 0.2.13+custom.007

基于 GoToCC `0.2.13+custom.006`，上游基线保持 Plus `v0.2.13+custom.001`。

## 上游输出前内部错误自动换号

- OpenAI 上游在任何输出之前以裸 `error` 帧报告服务端内部错误（`internal_error` / `server_error`，例如 Codex 后端的 `response protection is unavailable`）时，网关按现有换号流程改用同组其它账号重试，不再把失败直接返回给客户端。
- 已开始输出的请求、网络安全策略拒绝、上下文超长等判定保持不变；这类失败此前计费为 0，换号同样不重复计费。

## 异步图片编辑张数上限可配置

- multipart 异步图片编辑的输入图片张数上限改为运行参数 `async_image.edit_max_input_images`，默认 16（原固定 4）；总上传大小 40 MiB 和 mask 规则不变。
- 超限报错文字带出当前上限。

## Codex 一键接入默认 WebSocket 配置

- OpenAI 分组「使用密钥」默认选中「Codex CLI (WebSocket)」页签，生成的 `config.toml` 带 `supports_websockets = true`，Codex 续轮在同一连接只上传增量；原 HTTP 模板仍在相邻页签。
- 默认页签由产品根目录 `client-access-defaults.json` 的 `openai_codex_client_tab` 控制（`codex-ws` 或 `codex`）。

## 兼容性

- 无数据库迁移；新增运行参数有默认值，现有 `config.yaml` 无需修改。
- 已在用的 Codex 配置不会自动改变，需用户在自己的 `config.toml` 里加 `supports_websockets = true` 或重新复制。
- 生产通过自有版本通道更新；本地验收与 GitHub 发布不代表生产已安装。

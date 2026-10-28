# OpenAI Images 上游编辑协议

API Key 图片接口保留调用者的模型名。JSON 图片编辑接受
`images[].image_url`，其中每项可为公网图片 URL 或 data URL；可选遮罩为
`mask.image_url`。multipart 编辑继续接受 `image` / `image[]` 与 `mask` 文件。

## Sourcesdata 的 multipart 适配

产品根参数 [`upstream-request-adaptations.json`](../../upstream-request-adaptations.json)
统一保存本轮有错误证据的请求改写。`image_edits_multipart_hosts` 目前只有
`api.sourcesdata.com`：这是图片编辑接口的内容类型约束，不按型号筛选。
唯一生成入口为 `tools/generate_upstream_request_adaptations.py`，输出
`upstream_request_adaptations_gen_gotocc.go`；修改参数后重新生成并执行 `--check`。
没有新增环境变量或生产数据库字段。

匹配的 JSON `/v1/images/edits` 请求在审计、账号选择和凭据解析之后，由共用出站
入口 `applyUpstreamRequestAdaptations` 在最终身份准备/压缩/发送之前完成转换：
下载或解码参考图，以原顺序写入图片文件字段；遮罩同样转换为文件。
原始 `prompt` 字符串及 `model`、`n`、`size`、`quality`、`output_format`、
`response_format` 等字段保留。扩展对象/数组作为原 JSON 文本表单值传递。
multipart writer 同时生成正文和带 boundary 的 Content-Type。

图片 URL 获取复用现有图片下载路径的账号代理、可信客户端身份、公开目标策略和
单图限制。获取素材不携带供应商认证头。任何获取/编码错误均在生成请求前返回；
不在失败后换一种协议再次生成。原始请求结构继续用于审计和计费，模型名、付款
主体和费用语义不因编码转换而变化。

data URL 与内联图片结果共用 Base64 规范化：先移除已有末尾填充，再按剩余长度
补齐 `=`。已带一个或两个填充字符的合法图片不会因按旧长度补位而解码失败；
省略填充的输入继续按原规则补齐。

原生 multipart、文生图及其他上游沿用原协议。
HTTP 自动透传开关不承担此图片编码转换；当前 API Key Images 分支统一按图片
协议处理，开启 `openai_passthrough` 仍会应用已配置的编辑适配。
同步与异步编辑复用同一 `ForwardImages` 路径。

该适配处理 JSON 编辑被上游按 `missing_prompt` 或 `content_type` 拒绝的问题。
供应商 524、生成耗时和任务完成状态属于独立故障，转换成功不代表生成一定完成。

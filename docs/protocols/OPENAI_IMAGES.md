# OpenAI Images 上游编辑协议

API Key 图片接口保留调用者的模型名。JSON 图片编辑接受
`images[].image_url`，其中每项可为公网图片 URL 或 data URL；可选遮罩为
`mask.image_url`。multipart 编辑继续接受 `image` / `image[]` 与 `mask` 文件。

## 按上游账号声明图片编辑格式

OpenAI 官方图片编辑接口同时支持 JSON 和 multipart。第三方兼容接口可能只接受
multipart，因此格式能力属于持有 API 地址和凭据的账号，不由域名或模型名推断。
[官方接口说明](https://developers.openai.com/api/reference/resources/images/methods/edit)。

OpenAI API Key 账号的 `extra.image_edits_multipart` 是布尔值。管理员创建、编辑
账号时可设置「图片编辑转为文件表单」：

- `true`：该账号所有模型的 JSON 图片编辑转为 multipart；换 API 地址仍然生效。
- `false`或未配置：保留客户端请求格式。默认值来自根
  [`upstream-request-adaptations.json`](../../upstream-request-adaptations.json)的
  `image_edits_multipart.default`，当前为false。

新账号持久化显式默认值；旧编辑客户端更新extra但遗漏该字段时保留已有值。
创建、更新、额外字段合并和批量更新入口拒绝该字段的非布尔值。
复制账号和导入导出沿用既有extra合同，不按新的地址重新判断格式。

迁移329只为已知Sourcesdata旧入口及美区直连入口、未显式配置该能力的OpenAI
API Key账号写入true，保留其他字段与管理员已有选择。域名清单只存在于这份冻结的
升级迁移输入，运行时不再匹配域名。`tools/generate_upstream_request_adaptations.py`
同时生成Go默认值与迁移SQL；已有`--check`核对两者。迁移发布后不修改其输入或SQL。
没有新增数据库列或环境变量。更新只发生在安装本发行并执行迁移时，发布本身不修改生产。

## 请求转换边界

启用账号的 JSON `/v1/images/edits` 请求在审计、账号选择和凭据解析之后，由共用
出站入口 `applyUpstreamRequestAdaptations` 在最终身份准备/压缩/发送之前转换。
读取的是实际凭据归属账号的能力，适用于该账号的所有图片编辑模型及同步/异步请求。
下载或解码参考图，以原顺序写入图片文件字段；遮罩同样转换为文件。
原始 `prompt` 字符串及 `model`、`n`、`size`、`quality`、`output_format`、
`response_format` 等字段保留。扩展对象/数组作为原 JSON 文本表单值传递。
multipart writer 同时生成正文和带boundary的Content-Type。

图片URL获取复用现有图片下载路径的账号代理、可信客户端身份、公开目标策略和
单图限制。获取素材不携带供应商认证头。获取/编码失败在生成请求前返回，不在失败后
切换协议重试。原始请求继续用于审计和计费，付款主体和异步归属保持不变。

data URL与内联图片结果沿用Base64规范化：先移除已有末尾填充，再按剩余长度
补齐`=`，支持已填充与未填充的合法图片。

原生multipart、文生图、未启用账号及OAuth图片协议保持原行为。
HTTP自动透传开关不承担图片编码转换；开启`openai_passthrough`仍应用账号能力。

## 验证与恢复边界

本地出站捕获可以验证Content-Type/boundary、原始提示词、图片/遮罩字节、地址更换
及账号开关效果，不能替代供应商真实生成。供应商524、任务完成与实际扣费是独立事项。
旧二进制不读取此账号能力，回退二进制会丢失新域名的适配；应保留最新数据和配置，
按供应商实际协议前向修复，不用旧数据库覆盖新业务。

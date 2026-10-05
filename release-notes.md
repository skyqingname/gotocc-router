# GoToCC 0.2.13+custom.003

基于 Plus `v0.2.13+custom.001`，在已发布的 GoToCC `0.2.13+custom.002` 上定向修复客户站 API 端点。

- 客户站默认 API 端点直接使用 `reseller-sites.json` 的客户域名，正式地址为 `https://ai.gotocc.xyz`。
- 客户站不再显示主站配置的备用端点；API 密钥页复制、使用密钥配置与 CC Switch 导入共用客户地址。
- 主站的 API 地址及备用端点配置保持原行为。

本次无数据库迁移、账务变更或新增运行参数。沿用现有双域名配置；本地客户入口为 `http://localhost:18080`。生产由用户通过自有更新通道安装，发布不等于已上线。

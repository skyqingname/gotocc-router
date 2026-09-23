# GoToCC 0.2.7+custom.003

上游基线：Plus `v0.2.7+custom.001`，commit `7b2d38cc501b0e73302c2609ee79f39f75d3c5e8`；官方 Sub2API `v0.2.7`，commit `aea725f2ea644d5592d0bbb1d63b607efa7e200a`。本版基于已发布的 GoToCC `0.2.7+custom.002`，保留其功能与修复。

## GPT-6 默认价格修正

OpenAI 官方 Standard 价格，单位美元／百万 token：

| 模型 | 输入 | 缓存读取 | 缓存写入 | 输出 |
| --- | ---: | ---: | ---: | ---: |
| `gpt-6-sol` | 2 | 0.20 | 2.50 | 10 |
| `gpt-6-luna` | 0.10 | 0.01 | 0.125 | 0.50 |

同步更新随版定价目录；远端目录若保留旧价，这两项默认价仍采用随版官方价格。超过 272K 输入 token 的请求对整单输入及缓存按 2 倍、输出按 1.5 倍计价；Fast 为 Standard 的 2 倍，Batch/Flex 为一半。已明确配置的分组、渠道价格保持优先，不改写已有账目。价格依据：[OpenAI GPT-6 Sol](https://developers.openai.com/api/docs/models/gpt-6-sol)、[OpenAI GPT-6 Luna](https://developers.openai.com/api/docs/models/gpt-6-luna)。

## Claude Opus 5.5 默认价格

官方模型 ID 为 `claude-opus-5-5`。Standard 输入／输出／缓存读取／5 分钟缓存写入／1 小时缓存写入分别为 $4／$20／$0.20／$5／$8 每百万 token。Fast 为上述价格的 2 倍，Batch 为一半；完整 1M 上下文使用相同单价。随版默认价覆盖远端旧目录条目，已明确配置的分组、渠道售价保持优先，不改写已有账目。价格依据：[Anthropic Claude Opus 5.5](https://platform.claude.com/docs/en/models/opus-5-5/overview)。

## 升级与回滚

相对 `0.2.7+custom.002`，本版不新增 SQL migration、配置项、Redis 数据格式或历史资金回填。二进制与定价资源组成同一发行集合；回退时保持集合匹配。若线上仍是更早的生产版本，跨版本历史迁移及其回滚限制仍适用，详见上一版发行说明；发生前向迁移或新业务写入后，不能仅替换旧二进制或用旧 dump 覆盖当前数据。

发行包包含 Linux/amd64 压缩包、更新器要求的清单、定价 JSON 和定价清单。GitHub 发布不代表生产已更新；用户从自有版本面板更新并按提示重启。

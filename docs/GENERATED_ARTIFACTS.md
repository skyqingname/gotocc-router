# 参数、生成物与运行证据

根 JSON 是可维护输入，运行程序可能读取编译后的 Go 常量、前端资源或打包文件。修改输入不等于消费者已更新。执行范围遵循 [AGENTS.md](../AGENTS.md)；不新增或运行测试套件，不进行本地 SHA sum。

## 按变更选择生成步骤

1. 从消费者查出实际输入及生成器，阅读生成器的读取、写入、联网行为和命令参数。
2. 修改真源，执行受影响生成器，审阅全部输出。不要直接修改派生结果；不要批量刷新与任务无关的目录、价格或指纹数据。
3. 生成器有 `--check` 时使用已有只读检查；没有时按生成脚本核对输出，不猜测参数。
4. 输入与输出一起固定到包的源码提交，随后编译、核对包外文件并运行候选，验证消费者得到的值。

常见入口如下；完整输出范围以实际脚本为准。

| 输入 | 生成入口 | 主要消费者 |
| --- | --- | --- |
| `release-channel.json` | `tools/generate_release_channel.py` | `backend/internal/releasechannel/channel_gen.go`、安装脚本、通道相关文本与镜像引用 |
| `billing-credits.json` | `tools/generate_billing_credits.py` | `backend/internal/service/billing_credits_defaults_gotocc.go`、随包价格目录 |
| `content-audit-jev.json` | `tools/generate_content_audit_jev.py` | 后端嵌入的 `content_audit_jev_gen.json`、统一审核类别/阈值/评分映射；前端直接读取根 JSON |
| `upstream-request-adaptations.json` | `tools/generate_upstream_request_adaptations.py`，支持 `--check` | `backend/internal/service/upstream_request_adaptations_gen_gotocc.go`；图片编辑按上游 host 选择 multipart，原生 Messages 按最终 Opus 5.5 模型转换旧思考参数 |
| `model-fingerprint.json` | `tools/generate_model_fingerprint.py` | 后端指纹参数及前端 `parameters.gen.json` |
| `model-plaza-catalog.json` | `tools/generate_model_catalog.py` | `backend/internal/pkg/modelcatalog/catalog_gen.json`；该生成器会联网下载数据 |
| 其他路由、返佣、卡密、站长、审计根参数 | 对应 `tools/generate_*.py` | 依据本次改动逐项追踪，不能假设所有脚本都有检查模式 |
| `backend/ent/schema/` | 现有 Ent/Wire 生成入口 | Ent 与依赖注入代码，见 [贡献说明](../CONTRIBUTING.md#generated-code) |

## 发行通道

在产品源码根目录，修改通道输入后执行已有命令：

```sh
python3 tools/generate_release_channel.py
python3 tools/generate_release_channel.py --check
```

生成器也维护安装脚本、README、部署说明和部分前端通道文本。审阅完整差异，确认只有根配置要求的变化；不要手工删回派生更新以缩小 diff。它是生成步骤，不是测试套件。

`UPSTREAM.md` 记录来源仓库的身份与镜像引用，不参与自有通道镜像替换，避免把上游同名 tag 误写为自有镜像。

上游适配完成时就同步 `upstream_baseline` 和派生常量；自有版本号仍按用户确认的发版安排确定。若后续发行身份整理再次修改运行参数，重复受影响生成步骤及验证，再固定包源码。

当前 `tools/gotocc_build.py` 和 Oracle 维护入口的 `scripts/package_plus.py` 直接编译，不会自动运行发行通道生成器。构建和 CI 成功不能证明这份生成文件是最新的。本说明要求 Agent 执行现有步骤，没有新增自动构建动作。

## 验证实际消费者

更新服务使用编译进程序的 `releasechannel.UpstreamBaseline`，不是每次读取磁盘上的 `release-channel.json`。因此包里的 JSON 正确、主程序 `-version` 正确仍可能出现上游提示错误。

在最终候选的管理员版本弹层，或已有的只读 `GET /api/v1/admin/system/check-updates` 响应中核对：

- `current_version` 与包版本一致；`release_repository` 指向自有通道。
- `upstream_repository` 与固定源码配置一致。
- `upstream_baseline` 与已适配上游和生成常量一致。
- `upstream_latest_version`、`upstream_warning` 和 `upstream_has_update` 对应本次查询结果；同一基线且查询成功时不应提示更新。

上游检查失败或验证期间新版本发布时记录实际结果，不强行要求提示为灰色。检查不包含安装、更新或重启。发布步骤见 [RELEASING.md](RELEASING.md)。

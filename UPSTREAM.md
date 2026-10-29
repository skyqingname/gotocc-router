# Upstream Mapping

This file is the authoritative mapping of Plus tags to official baselines and
publication status. Release naming and procedures live in
[RELEASING.md](docs/RELEASING.md).

## Integrated Baseline

The current GoToCC tree integrates the complete fixed Plus release
`LuckyKuang/sub2api-plus@v0.2.19+custom.001`: tag object
`e1e1862294281e333c32370ade896e1fa0e26f8f`, peeled commit
`6599d4b29e125277232dba75ad5b84d5b14799c4`. Its official baseline is
`Wei-Shaw/sub2api@v0.2.19`, commit `dece26153621959ccd7635814f4b2d08304e681a`.
The owned release is `skyqingname/gotocc-router@v0.2.19+custom.001`; identical
tag text in these repositories does not identify the same source or package.

## GoToCC source integration (2026-10-29)

This integration retains the active GoToCC customizations and retired features
from owned `0.2.18+custom.007`, and includes PR #55 (client cancellation before
session-route errors) and PR #56 (native Opus 5.5 legacy thinking adaptation).
Sourcesdata multipart edits, Grok reasoning adaptation, timing/route metadata,
local image storage, owned release channels and the existing brand remain.

The Plus update adds frozen pricing generations and composite rate policies,
user/account/unfinished-task admission, scoped administrator keys, opt-in
image-count billing, Gemini retry/discovery improvements and GPT-6.1 Terra
pricing. The inherited pinned Codex identity is 0.165.0; credential-owner and
global-setting precedence remain unchanged. This is a fixed-release import,
not automatic identity-version synchronization.

Team wallets retain their own balance namespace, actor and payer identities.
Reseller credits and platform cost retain separate reservations and settlement;
selected-account repricing updates both reservations. Reseller retail rates
remain explicit and frozen, and team key quotes use the team's authorized
pricing owner. Automatic routing retains its group/key/funding owner and
per-turn validation. GoToCC credit pricing uses original-currency numeric
values and DeepSeek all-day peak prices. DeepSeek Pro now retains its own
price family as specified by the fixed upstream; historical accepted task
prices and ledger rows are not recalculated.

Only incoming SQL files are numbered **324–328**; existing migrations through
323 and prior duplicate historical prefixes remain immutable. Ent and Wire
outputs are regenerated from the merged inputs. The release-channel generator
compiles the actual Plus baseline `v0.2.19+custom.001`, independently of the
owned repository/tag identity. Publication and production activation remain
separate operations; this source document does not assert deployment.

## Removed Upstream Paths

GoToCC builds through `tools/gotocc_build.py`, checks pull requests with
`.github/workflows/ci.yml`, and publishes from the operations workflow. The
upstream CI, release and validation tooling listed in
`upstream-removed-paths.json` is not part of this tree; the operations
`prepare` step removes those paths again after every upstream merge.

## Plus Baseline Version

```text
Git/GitHub: v0.2.19+custom.001
Application: 0.2.19+custom.001
GHCR: ghcr.io/luckykuang/sub2api-plus:v0.2.19-custom.001
```

## Release Mapping

| Custom Release | Official Release | Official Commit | Status |
| --- | --- | --- | --- |
| `v0.1.164+custom.001` | `v0.1.164` | `cd8bb98c44303b2c8f04c0da340447c992f0cb7d` | historical |
| `v0.1.164+custom.003` | `v0.1.164` | `cd8bb98c44303b2c8f04c0da340447c992f0cb7d` | historical |
| `v0.1.164+custom.004` | `v0.1.164` | `cd8bb98c44303b2c8f04c0da340447c992f0cb7d` | historical |
| `v0.1.164+custom.005` | `v0.1.164` | `cd8bb98c44303b2c8f04c0da340447c992f0cb7d` | historical |
| `v0.1.165+custom.001` | `v0.1.165` | `e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29` | published |
| `v0.1.165+custom.002` | `v0.1.165` | `e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29` | published |
| `v0.1.165+custom.003` | `v0.1.165` | `e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29` | published |
| `v0.1.165+custom.004` | `v0.1.165` | `e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29` | published |
| `v0.1.166+custom.001` | `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | published |
| `v0.1.166+custom.002` | `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | published |
| `v0.1.166+custom.003` | `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | published |
| `v0.1.166+custom.004` | `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | published |
| `v0.1.166+custom.005` | `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | published |
| `v0.1.166+custom.006` | `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | published |
| `v0.1.166+custom.007` | `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | invalid |
| `v0.1.166+custom.008` | `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | published |
| `v0.1.166+custom.009` | `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | published |
| `v0.1.166+custom.010` | `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | published |
| `v0.1.168+custom.001` | `v0.1.168` | `99c8e4bf7564823bafbab369acab6539e734c1bb` | published |
| `v0.1.169+custom.001` | `v0.1.169` | `26d894ef4f50645a4bf1030e378ac892f17d0223` | published |
| `v0.1.169+custom.002` | `v0.1.169` | `26d894ef4f50645a4bf1030e378ac892f17d0223` | published |
| `v0.1.170+custom.001` | `v0.1.170` | `c043c24774228ba891ddf90d783aa6dc7d0855b5` | published |
| `v0.1.170+custom.002` | `v0.1.170` | `c043c24774228ba891ddf90d783aa6dc7d0855b5` | published |
| `v0.1.171+custom.001` | `v0.1.171` | `f0e7a9c7a23a7d02fb159b62fa809621eb0475a6` | published |
| `v0.1.172+custom.001` | `v0.1.172` | `155c494964c3ea6ecc31f52679525c1034bf0f16` | published |
| `v0.1.173+custom.002` | `v0.1.173` | `29009f0b2ea14edf3b11ae2564fb617ff91a03b4` | published |
| `v0.1.173+custom.003` | `v0.1.173` | `29009f0b2ea14edf3b11ae2564fb617ff91a03b4` | published |
| `v0.1.173+custom.004` | `v0.1.173` | `29009f0b2ea14edf3b11ae2564fb617ff91a03b4` | published |
| `v0.1.176+custom.001` | `v0.1.176` | `e803e3851c0a7e222cfadeafad7b8636ab959d11` | published |
| `v0.1.176+custom.002` | `v0.1.176` | `e803e3851c0a7e222cfadeafad7b8636ab959d11` | published |
| `v0.1.177+custom.001` | `v0.1.177` | `073e92d17178a1ccdb0a27017f572f10c9c7ab62` | published |
| `v0.1.177+custom.002` | `v0.1.177` | `073e92d17178a1ccdb0a27017f572f10c9c7ab62` | published |
| `v0.1.177+custom.003` | `v0.1.177` | `073e92d17178a1ccdb0a27017f572f10c9c7ab62` | published |
| `v0.1.178+custom.001` | `v0.1.178` | `e0c48a19ed794a565e3858662520afe0a1f9f0ba` | published |
| `v0.1.178+custom.002` | `v0.1.178` | `e0c48a19ed794a565e3858662520afe0a1f9f0ba` | published |
| `v0.1.178+custom.003` | `v0.1.178` | `e0c48a19ed794a565e3858662520afe0a1f9f0ba` | published |
| `v0.1.178+custom.004` | `v0.1.178` | `e0c48a19ed794a565e3858662520afe0a1f9f0ba` | published |
| `v0.1.178+custom.005` | `v0.1.178` | `e0c48a19ed794a565e3858662520afe0a1f9f0ba` | published |
| `v0.1.183+custom.001` | `v0.1.183` | `e8cb019fabf8b55199436229044cbf9aa7a82564` | published |
| `v0.1.183+custom.002` | `v0.1.183` | `e8cb019fabf8b55199436229044cbf9aa7a82564` | published |
| `v0.1.183+custom.003` | `v0.1.183` | `e8cb019fabf8b55199436229044cbf9aa7a82564` | published |
| `v0.1.183+custom.004` | `v0.1.183` | `e8cb019fabf8b55199436229044cbf9aa7a82564` | published |
| `v0.2.0+custom.001` | `v0.2.0` | `aa236488351eb71e120fc2b6fb32e36b0374c918` | published |
| `v0.2.0+custom.002` | `v0.2.0` | `aa236488351eb71e120fc2b6fb32e36b0374c918` | published |
| `v0.2.0+custom.003` | `v0.2.0` | `aa236488351eb71e120fc2b6fb32e36b0374c918` | published |
| `v0.2.1+custom.001` | `v0.2.1` | `578785ee7fb35030b094b69624efe25670a36f5f` | published |
| `v0.2.1+custom.002` | `v0.2.1` | `578785ee7fb35030b094b69624efe25670a36f5f` | published |
| `v0.2.1+custom.003` | `v0.2.1` | `578785ee7fb35030b094b69624efe25670a36f5f` | published |
| `v0.2.4+custom.001` | `v0.2.4` | `5de5e2bed035d43591a2e10e51f420ef6a84eb98` | published |
| `v0.2.4+custom.002` | `v0.2.4` | `5de5e2bed035d43591a2e10e51f420ef6a84eb98` | published |
| `v0.2.4+custom.003` | `v0.2.4` | `5de5e2bed035d43591a2e10e51f420ef6a84eb98` | published |
| `v0.2.4+custom.004` | `v0.2.4` | `5de5e2bed035d43591a2e10e51f420ef6a84eb98` | published |
| `v0.2.4+custom.005` | `v0.2.4` | `5de5e2bed035d43591a2e10e51f420ef6a84eb98` | published |
| `v0.2.4+custom.006` | `v0.2.4` | `badfad8b7248b8aac0e6b503a06e392aa31cb294` | withdrawn |
| `v0.2.5+custom.001` | `v0.2.5` | `86f93c28ee34cc74b629dafb748bd5ac5ca8c5ea` | published |
| `v0.2.7+custom.001` | `v0.2.7` | `aea725f2ea644d5592d0bbb1d63b607efa7e200a` | published |
| `v0.2.8+custom.001` | `v0.2.8` | `fd80b08c90b55edcad5b00171b53f08721d30da1` | published |
| `v0.2.8+custom.002` | `v0.2.8` | `fd80b08c90b55edcad5b00171b53f08721d30da1` | published |
| `v0.2.9+custom.001` | `v0.2.9` | `4c00df2e0183e2c70b7fa8ba45914205e36aad0c` | published |
| `v0.2.10+custom.001` | `v0.2.10` | `2f3fed2fdb0787141294cec81487a5df30426f7f` | published |
| `v0.2.11+custom.001` | `v0.2.11` | `96f4c115c9749078f90cbf210a01d39baf3f53b6` | published |
| `v0.2.11+custom.002` | `v0.2.11` | `96f4c115c9749078f90cbf210a01d39baf3f53b6` | published |
| `v0.2.13+custom.001` | `v0.2.13` | `3040209f205472038c1ba745a1bedd2edd9053b1` | published |
| `v0.2.14+custom.001` | `v0.2.14` | `0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d` | published |
| `v0.2.14+custom.002` | `v0.2.14` | `0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d` | published |
| `v0.2.14+custom.003` | `v0.2.14` | `0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d` | published |
| `v0.2.14+custom.004` | `v0.2.14` | `0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d` | published |
| `v0.2.15+custom.001` | `v0.2.15` | `f2669c8cf62555cd92389b3f55920e9e6e7c6ff2` | published |
| `v0.2.15+custom.002` | `v0.2.15` | `f2669c8cf62555cd92389b3f55920e9e6e7c6ff2` | published |
| `v0.2.15+custom.003` | `v0.2.15` | `ab80abc6d1f929a6864c8804fd077b509d21fda2` | published |
| `v0.2.17+custom.001` | `v0.2.17` | `060d0160417782cd962a0c7f460f6cd4565d4916` | published |
| `v0.2.18+custom.001` | `v0.2.18` | `ac1b8f1271be0d674fa6f074403c1410f30a3a81` | upstream published |
| `v0.2.18+custom.002` | `v0.2.18` | `ac1b8f1271be0d674fa6f074403c1410f30a3a81` | owned GoToCC release; see GitHub Release |
| `LuckyKuang/sub2api-plus@v0.2.18+custom.003` | `v0.2.18` | `ac1b8f1271be0d674fa6f074403c1410f30a3a81` | upstream published |
| `skyqingname/gotocc-router@v0.2.18+custom.003` | `v0.2.18` | `ac1b8f1271be0d674fa6f074403c1410f30a3a81` | owned GoToCC release; integrates Plus `.003`; see GitHub Release |
| `skyqingname/gotocc-router@v0.2.18+custom.004` | `v0.2.18` | `ac1b8f1271be0d674fa6f074403c1410f30a3a81` | owned GoToCC release; retains Plus `.003`; response metadata and admin usage layout |
| `skyqingname/gotocc-router@v0.2.18+custom.005` | `v0.2.18` | `ac1b8f1271be0d674fa6f074403c1410f30a3a81` | owned GoToCC release; retains Plus `.003`; separate gateway and upstream-reported response timing |
| `skyqingname/gotocc-router@v0.2.18+custom.006` | `v0.2.18` | `ac1b8f1271be0d674fa6f074403c1410f30a3a81` | owned GoToCC release; retains Plus `.003`; actual gateway route metadata and version-panel positioning |
| `skyqingname/gotocc-router@v0.2.18+custom.007` | `v0.2.18` | `ac1b8f1271be0d674fa6f074403c1410f30a3a81` | owned release; image edits and Grok compatibility |
| `LuckyKuang/sub2api-plus@v0.2.19+custom.001` | `v0.2.19` | `dece26153621959ccd7635814f4b2d08304e681a` | upstream published |
| `skyqingname/gotocc-router@v0.2.19+custom.001` | `v0.2.19` | `dece26153621959ccd7635814f4b2d08304e681a` | owned integration; publication status is authoritative in the owned GitHub Release |

`v0.2.15+custom.003` records the integrated official main snapshot in the
Official Commit column. Its tagged official baseline remains
`v0.2.15` / `f2669c8cf62555cd92389b3f55920e9e6e7c6ff2`.

`v0.2.17+custom.001` records official tag `v0.2.17` / peeled
`060d0160417782cd962a0c7f460f6cd4565d4916`. Official tag VERSION is `0.2.16`;
Plus writes `0.2.17+custom.001`.

`v0.2.18+custom.001` records official tag `v0.2.18` / peeled
`ac1b8f1271be0d674fa6f074403c1410f30a3a81`. Official tag VERSION is `0.2.17`;
Plus and the owned GoToCC artifact both use `0.2.18+custom.001`; repositories
and their release assets remain separate.

`LuckyKuang/sub2api-plus@v0.2.18+custom.002` added the seven-provider gateway work, but is withdrawn after
its post-publication audit found that binary archives placed fallback pricing at
the wrong path and the installer omitted resources. Its immutable tag and assets
are preserved. Docker's separate resource-copy path is outside this archive defect.

`LuckyKuang/sub2api-plus@v0.2.18+custom.003` publishes the binary archive and
installer correction on the same official baseline. It was published on
2026-10-26; GoToCC imports its fixed tag above. The upstream
withdrawal does not apply to the separate owned GoToCC `.002` release.

`v0.2.19+custom.001` records official tag `v0.2.19` / peeled
`dece26153621959ccd7635814f4b2d08304e681a`. Official tag VERSION is `0.2.18`;
Plus writes `0.2.19+custom.001`. Official migrations 247/248 already entered
Plus as immutable 302/303 during the preceding snapshot import; this tag adds
no duplicate migrations. Seven-provider agent identities, audit boundaries,
strict proxy ownership, frozen billing and the Plus UI remain authoritative.

`v0.2.4+custom.006` is marked withdrawn because official `v0.2.5` was imported before that snapshot overlay was published. Do not reuse or retag `.006`.

`v0.1.166+custom.007` is marked invalid because its tag contains embedded and
documented version `0.1.166+custom.006`. Remote Release and OCI artifact status
still require a maintainer audit. Do not reuse or retag `.007`.

## Repository Roles

- `origin`: [GoToCC](https://github.com/skyqingname/gotocc-router), the owned
  installation, update, rollback and release source.
- [Sub2API Plus](https://github.com/LuckyKuang/sub2api-plus) supplies the fixed
  Plus baseline; [official Sub2API](https://github.com/Wei-Shaw/sub2api) supplies
  its underlying official baseline.

Update this mapping in the same change as an upstream integration. Keep tag,
embedded version and OCI naming synchronized through the release workflow;
published tags and artifacts are immutable.

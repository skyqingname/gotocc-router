# 创作台动态图标

仅供 GoToCC 创作台使用，共 25 个实际使用的语义图标。`StudioIcon.vue` 使用 `registry.ts`，`useIconAnimation.ts` 管理悬停、键盘聚焦、减少动态效果和卸载清理；其它页面继续使用现有图标。

图形与逐元素动画改编自 TokenFlux/TokenRouter 固定提交 `f383062dd5fd534f462887acb70ea241ce0d8278`，保留其 SVG、关键帧和缓动。动态图形来源为 Lucide Animated `072c38b1b04ea738d90a084485ccaad4b890ddca`（MIT），补充图形来源为 Lucide `66d8f9fc394b8530377e5f6112f0b8908ba01280`。许可见同目录 `LICENSE.lucide-animated` 与 `LICENSE.lucide`，每个 artwork 文件保留来源链接；发行包根目录也包含两份许可。

动画由固定版本 `motion-v 2.5.1` 执行并随前端打包，运行时不访问外部 CDN。简单图形统一使用 `createShapeIcon` 的原版轻微缩放；未注册的名称由 TypeScript 报错，不替换成其它图形。加载旋转和菜单箭头旋转由外层 CSS 控制，不叠加图标悬停动画。

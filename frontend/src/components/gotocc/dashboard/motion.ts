import type { CSSProperties } from 'vue'

// 仪表盘动效时长，参数取自 TokenRouter（LGPL-3.0）。脚本直接引用常量，样式读取 dashboardMotionVars 注入的 CSS 变量。
// 系统开启「减少动态效果」时，各组件跳过对应动画。

// 指标数字从旧值滚动到新值的时长。
export const COUNT_UP_MS = 900
// 趋势图重新取数后从左到右描线的总时长。
export const CHART_REVEAL_MS = 800
// 指标卡迷你走势线的描线时长。
export const SPARKLINE_DRAW_MS = 700
// 页面各区块首次进入时上移淡入的时长，相邻区块错开一个步长。
export const DASH_RISE_MS = 420
export const DASH_RISE_STEP_MS = 60
// 热力图单个格子的入场时长；相邻两列错开一个步长，最晚一列不超过上限。
export const HEATMAP_CELL_ENTER_MS = 320
export const HEATMAP_WAVE_STEP_MS = 8
export const HEATMAP_WAVE_MAX_MS = 480
// 模型占比条伸展的时长，相邻两行错开一个步长。
export const TOP_MODEL_BAR_MS = 600
export const TOP_MODEL_BAR_STEP_MS = 60

export const dashboardMotionVars = {
  '--dash-rise-ms': `${DASH_RISE_MS}ms`,
  '--dash-rise-step-ms': `${DASH_RISE_STEP_MS}ms`,
  '--dash-sparkline-ms': `${SPARKLINE_DRAW_MS}ms`,
  '--dash-heatmap-enter-ms': `${HEATMAP_CELL_ENTER_MS}ms`,
  '--dash-top-model-bar-ms': `${TOP_MODEL_BAR_MS}ms`,
} as CSSProperties

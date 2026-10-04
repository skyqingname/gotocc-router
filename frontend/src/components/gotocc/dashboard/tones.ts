import { computed } from 'vue'
import { useTheme } from '@/composables/useTheme'

// 仪表盘配色。每类指标固定一个颜色，指标卡、实时状态条和图表共用，同一指标处处同色。
// 类名写成完整字符串，保证 Tailwind 能扫描到。
export const METRIC_TONES = {
  requests: { icon: 'text-primary-600 dark:text-primary-400', tile: 'bg-primary-500/10' },
  tokens: { icon: 'text-sky-600 dark:text-sky-400', tile: 'bg-sky-500/10' },
  cost: { icon: 'text-amber-600 dark:text-amber-400', tile: 'bg-amber-500/10' },
  cacheHitRate: { icon: 'text-violet-600 dark:text-violet-400', tile: 'bg-violet-500/10' },
  latency: { icon: 'text-rose-600 dark:text-rose-400', tile: 'bg-rose-500/10' },
} as const

export type MetricTone = keyof typeof METRIC_TONES

// Token 分项与命中率折线的语义色，与上面的指标色同一色系。
export const SERIES_COLORS = {
  input: '#0ea5e9',
  output: '#10b981',
  cacheCreation: '#f59e0b',
  cacheRead: '#8b5cf6',
  cacheHitRate: '#8b5cf6',
} as const

export const CHART_TICK_FONT_SIZE = 10

// 图表随深浅主题取色：品牌线取主色 600 / 400，卡片底色与 .card 一致，刻度和网格取中性灰。
export function useChartColors() {
  const { isDark } = useTheme()
  const colors = computed(() => (isDark.value
    ? { brand: '#818cf8', surface: '#282828', text: '#d4d4d4', muted: '#a3a3a3', grid: '#404040', tooltip: '#404040' }
    : { brand: '#4f46e5', surface: '#ffffff', text: '#525252', muted: '#a3a3a3', grid: '#e5e5e5', tooltip: '#171717' }))
  return { isDark, colors }
}

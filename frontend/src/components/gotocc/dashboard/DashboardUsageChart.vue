<template>
  <div class="space-y-4" :aria-busy="loading">
    <!-- 指标卡展示所选范围的合计与环比，点击后在下方折线图中查看该指标 -->
    <div class="dash-rise-item grid grid-cols-2 gap-4 lg:grid-cols-4" :style="{ '--rise-i': 1 }" role="tablist" :aria-label="t('gotocc.dashboard.usageChart.metricsLabel')">
      <button
        v-for="item in metricTabs"
        :key="item.key"
        type="button"
        role="tab"
        :aria-selected="metric === item.key"
        class="metric-card card flex min-w-0 flex-col gap-3 overflow-hidden p-4 text-left"
        :class="metric === item.key ? '!border-primary-500 ring-1 ring-primary-500/30 dark:!border-primary-400' : 'hover:!border-gray-300 dark:hover:!border-dark-500'"
        @click="selectMetric(item.key)"
      >
        <div class="flex min-w-0 items-center gap-2">
          <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md" :class="[METRIC_TONES[item.key].tile, METRIC_TONES[item.key].icon]">
            <Icon :name="item.icon" size="sm" />
          </span>
          <span class="truncate text-xs font-medium text-gray-500 dark:text-dark-400">{{ item.label }}</span>
        </div>
        <div class="flex min-w-0 items-end justify-between gap-3">
          <Skeleton v-if="!loaded" :width="96" :height="32" />
          <span v-else class="min-w-0 truncate text-2xl font-semibold tracking-tight tabular-nums text-gray-900 dark:text-white">
            <!-- 滚动中的数字只做视觉展示，读屏读取最终值 -->
            <span aria-hidden="true">{{ item.animatedValue }}</span>
            <span class="sr-only">{{ item.value }}</span>
          </span>
          <UsageSparkline
            v-if="loaded"
            class="metric-sparkline h-8 w-20 shrink-0"
            :class="metric === item.key ? 'text-primary-600 dark:text-primary-400' : 'text-gray-300 dark:text-dark-500'"
            :values="item.series"
          />
        </div>
        <Skeleton v-if="!loaded" :width="120" :height="20" />
        <span v-else class="flex min-w-0 items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
          <template v-if="item.delta">
            <span class="inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-0.5 font-medium tabular-nums" :class="DELTA_CLASSES[item.delta.tone]">
              <Icon v-if="item.delta.tone !== 'flat'" :name="item.delta.tone === 'up' ? 'arrowUp' : 'arrowDown'" size="xs" />
              {{ item.delta.text }}
            </span>
            <span class="metric-delta-label truncate">{{ t('gotocc.dashboard.usageChart.vsPrevious') }}</span>
          </template>
          <span v-else class="truncate">{{ t('gotocc.dashboard.usageChart.noPrevious') }}</span>
        </span>
      </button>
    </div>

    <!-- 折线图与模型排行并排，窄于 xl 时排行排到图表下方 -->
    <div class="dash-rise-item grid grid-cols-1 gap-4 xl:grid-cols-3" :style="{ '--rise-i': 2 }">
      <div ref="chartCardRef" class="card flex min-w-0 flex-col p-4 xl:col-span-2">
        <div class="mb-4 flex min-h-7 flex-wrap items-center justify-between gap-2">
          <div class="flex items-baseline gap-2">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ chartTitle }}</h3>
            <span class="text-xs text-gray-500 dark:text-dark-400">{{ granularityHint }}</span>
          </div>
          <!-- 图例兼开关：Token 按分项开关，其余指标开关上一周期对比线 -->
          <div class="flex flex-wrap gap-2">
            <template v-if="metric === 'tokens'">
              <button
                v-for="series in tokenSeries"
                :key="series.key"
                type="button"
                class="legend-chip"
                :class="hiddenTokenSeries.includes(series.key) ? 'legend-chip-off' : 'legend-chip-on'"
                :aria-pressed="!hiddenTokenSeries.includes(series.key)"
                @click="toggleTokenSeries(series.key)"
              >
                <span class="h-2 w-2 rounded-full" :class="{ 'opacity-30': hiddenTokenSeries.includes(series.key) }" :style="{ backgroundColor: series.color }"></span>
                {{ series.label }}
              </button>
            </template>
            <template v-else>
              <span class="legend-chip legend-chip-on">
                <span class="h-0.5 w-3 rounded-full" :style="{ backgroundColor: mainColor }"></span>
                {{ t('gotocc.dashboard.usageChart.currentPeriod') }}
              </span>
              <button type="button" class="legend-chip" :class="showPrevious ? 'legend-chip-on' : 'legend-chip-off'" :aria-pressed="showPrevious" @click="showPrevious = !showPrevious">
                <span class="w-3 border-t-2 border-dashed" :style="{ borderColor: colors.muted }" :class="{ 'opacity-30': !showPrevious }"></span>
                {{ t('gotocc.dashboard.usageChart.previousPeriod') }}
              </button>
            </template>
          </div>
        </div>

        <div class="relative h-64 sm:h-80">
          <Skeleton v-if="!loaded" height="100%" />
          <div v-else-if="!hasUsage" class="flex h-full flex-col items-center justify-center gap-2 text-sm text-gray-500 dark:text-dark-400">
            <Icon name="chart" size="lg" class="text-gray-300 dark:text-dark-600" />
            {{ t('gotocc.dashboard.usageChart.empty') }}
          </div>
          <template v-else>
            <!-- 切换指标或重新取数时重建图表，与首次打开播放同一套描线动画 -->
            <Line ref="lineRef" :key="chartKey" :data="chartData" :options="chartOptions" :plugins="chartPlugins" />
            <!-- 未结束时段的末端呼吸点，位置由图表插件在每次绘制后写入 -->
            <span v-if="livePoint" class="pointer-events-none absolute" :style="{ left: `${livePoint.x}px`, top: `${livePoint.y}px`, color: mainColor }" aria-hidden="true">
              <span class="absolute -left-1.5 -top-1.5 h-3 w-3 animate-ping rounded-full bg-current opacity-60 motion-reduce:animate-none"></span>
              <span class="absolute -left-1 -top-1 h-2 w-2 rounded-full bg-current ring-2 ring-white dark:ring-dark-800"></span>
            </span>
          </template>
        </div>
      </div>

      <DashboardTopModels class="xl:col-span-1" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePreferredReducedMotion } from '@vueuse/core'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  type Chart,
  type ChartDataset,
  type Plugin,
  type ScriptableLineSegmentContext,
  type TooltipItem,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import type { TrendDataPoint } from '@/types'
import DashboardTopModels from './DashboardTopModels.vue'
import UsageSparkline from './UsageSparkline.vue'
import { formatCostAuto, formatUsd } from './format'
import { CHART_REVEAL_MS, COUNT_UP_MS } from './motion'
import { CHART_TICK_FONT_SIZE, METRIC_TONES, SERIES_COLORS, useChartColors } from './tones'
import { cacheHitRateOf } from './usageData'
import { injectUsageState, type UsageMetric } from './usageState'
import { useCountUp } from './useCountUp'

// 指标卡与趋势图，结构与动效参照 TokenRouter（LGPL-3.0）：未结束时段画虚线，标出峰值与均值，可叠加上一周期对比线。

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip)

type TokenSeriesKey = 'input_tokens' | 'output_tokens' | 'cache_creation_tokens' | 'cache_read_tokens'
type MetricIcon = 'swap' | 'cube' | 'dollar' | 'database'
type UsageDataset = ChartDataset<'line', Array<number | null>> & { previous?: boolean }

const DASH = [4, 4]
// 峰值与均值标签胶囊的尺寸。
const LABEL_HEIGHT = 18
const LABEL_PADDING_X = 6
const LABEL_GAP = 10

const DELTA_CLASSES = {
  up: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  down: 'bg-red-500/10 text-red-600 dark:text-red-400',
  flat: 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300',
}

type Delta = { text: string; tone: keyof typeof DELTA_CLASSES }

const { t } = useI18n()
const { colors } = useChartColors()
const reducedMotion = usePreferredReducedMotion()
const { metric, activeRange, loading, loaded, current, previous, previousTotals, totals, selectMetric } = injectUsageState()

const hiddenTokenSeries = ref<TokenSeriesKey[]>([])
const showPrevious = ref(true)
const lineRef = ref<{ chart?: Chart } | null>(null)
const chartCardRef = ref<HTMLElement | null>(null)
const livePoint = ref<{ x: number; y: number } | null>(null)

const chartTitle = computed(() => t('gotocc.dashboard.usageChart.trendTitle', { metric: t(`gotocc.dashboard.usageChart.metrics.${metric.value}`) }))
const granularityHint = computed(() => t(activeRange.value.granularity === 'hour' ? 'gotocc.dashboard.usageChart.byHour' : 'gotocc.dashboard.usageChart.byDay'))

// 环比百分比；上一周期为 0 时无法比较。
const percentDelta = (now: number, before: number | undefined): Delta | null => {
  if (before === undefined || before <= 0) return null
  const percent = Math.round(((now - before) / before) * 100)
  if (percent === 0) return { text: '0%', tone: 'flat' }
  return { text: `${percent > 0 ? '+' : ''}${percent}%`, tone: percent > 0 ? 'up' : 'down' }
}

// 命中率按百分点比较。
const pointDelta = (now: number | null, before: number | null | undefined): Delta | null => {
  if (now === null || before === null || before === undefined) return null
  const change = Math.round((now - before) * 10) / 10
  if (change === 0) return { text: '0 pt', tone: 'flat' }
  return { text: `${change > 0 ? '+' : ''}${change.toFixed(1)} pt`, tone: change > 0 ? 'up' : 'down' }
}

const formatPercent = (value: number | null): string => (value === null ? '—' : `${value.toFixed(1)}%`)

const metricValueOf = (point: TrendDataPoint, key: UsageMetric): number | null => {
  if (key === 'tokens') return point.total_tokens
  if (key === 'cost') return point.actual_cost
  if (key === 'cacheHitRate') return cacheHitRateOf(point.input_tokens, point.cache_creation_tokens, point.cache_read_tokens)
  return point.requests
}

const animatedRequests = useCountUp(() => totals.value.requests, COUNT_UP_MS)
const animatedTokens = useCountUp(() => totals.value.tokens, COUNT_UP_MS)
const animatedCost = useCountUp(() => totals.value.cost, COUNT_UP_MS)
const animatedHitRate = useCountUp(() => totals.value.cacheHitRate ?? 0, COUNT_UP_MS)

const metricTabs = computed(() => {
  const now = totals.value
  const before = previousTotals.value
  const seriesOf = (key: UsageMetric) => current.value.map((point) => metricValueOf(point, key))
  return [
    { key: 'requests' as const, icon: 'swap' as MetricIcon, label: t('gotocc.dashboard.usageChart.metrics.requests'), value: formatNumber(now.requests), animatedValue: formatNumber(Math.round(animatedRequests.value)), delta: percentDelta(now.requests, before?.requests), series: seriesOf('requests') },
    { key: 'tokens' as const, icon: 'cube' as MetricIcon, label: t('gotocc.dashboard.usageChart.metrics.tokens'), value: formatTokensK(now.tokens), animatedValue: formatTokensK(animatedTokens.value), delta: percentDelta(now.tokens, before?.tokens), series: seriesOf('tokens') },
    { key: 'cost' as const, icon: 'dollar' as MetricIcon, label: t('gotocc.dashboard.usageChart.metrics.cost'), value: formatCostAuto(now.cost), animatedValue: formatCostAuto(animatedCost.value), delta: percentDelta(now.cost, before?.cost), series: seriesOf('cost') },
    { key: 'cacheHitRate' as const, icon: 'database' as MetricIcon, label: t('gotocc.dashboard.usageChart.metrics.cacheHitRate'), value: formatPercent(now.cacheHitRate), animatedValue: now.cacheHitRate === null ? '—' : formatPercent(animatedHitRate.value), delta: pointDelta(now.cacheHitRate, before?.cacheHitRate), series: seriesOf('cacheHitRate') },
  ]
})

const tokenSeries = computed(() => [
  { key: 'input_tokens' as const, label: t('gotocc.dashboard.usageChart.series.input'), color: SERIES_COLORS.input },
  { key: 'output_tokens' as const, label: t('gotocc.dashboard.usageChart.series.output'), color: SERIES_COLORS.output },
  { key: 'cache_creation_tokens' as const, label: t('gotocc.dashboard.usageChart.series.cacheCreation'), color: SERIES_COLORS.cacheCreation },
  { key: 'cache_read_tokens' as const, label: t('gotocc.dashboard.usageChart.series.cacheRead'), color: SERIES_COLORS.cacheRead },
])

const hasUsage = computed(() => totals.value.requests > 0 || totals.value.tokens > 0 || totals.value.cost > 0)

// 横轴按粒度只显示时分或月日，完整时间留给提示标题。
const shortLabel = (date: string): string => (activeRange.value.granularity === 'hour' ? date.slice(11, 16) : date.slice(5, 10))
const axisLabels = computed(() => current.value.map((point) => shortLabel(point.date)))

const mainColor = computed(() => (metric.value === 'cacheHitRate' ? SERIES_COLORS.cacheHitRate : colors.value.brand))

// 窗口包含当前时刻时，最后一个时段还没结束，数据不完整。
const openBucketIndex = computed(() => (
  activeRange.value.endAt.getTime() > Date.now() && current.value.length > 1 ? current.value.length - 1 : null
))

// 未结束的末段画成虚线，避免把还在累计的数据误读成下跌。
const segmentStyle = {
  borderDash: (context: ScriptableLineSegmentContext) => (context.p1DataIndex === openBucketIndex.value ? DASH : undefined),
}

const lineDataset = (label: string, data: Array<number | null>, color: string): UsageDataset => ({
  label,
  data,
  borderColor: color,
  backgroundColor: color,
  borderWidth: 2,
  fill: false,
  cubicInterpolationMode: 'monotone' as const,
  segment: segmentStyle,
  pointRadius: 0,
  pointHoverRadius: 5,
  pointHoverBorderWidth: 2,
  pointHoverBorderColor: colors.value.surface,
  pointHoverBackgroundColor: color,
  spanGaps: true,
})

const chartData = computed(() => {
  const points = current.value
  if (metric.value === 'tokens') {
    const visible = tokenSeries.value.filter((series) => !hiddenTokenSeries.value.includes(series.key))
    return {
      labels: axisLabels.value,
      datasets: visible.map((series) => ({ ...lineDataset(series.label, points.map((point) => point[series.key]), series.color), borderWidth: 1.75 })),
    }
  }
  const datasets: UsageDataset[] = [lineDataset(
    t(metric.value === 'cost' ? 'gotocc.dashboard.usageChart.series.actualCost' : `gotocc.dashboard.usageChart.metrics.${metric.value}`),
    points.map((point) => metricValueOf(point, metric.value)),
    mainColor.value,
  )]
  if (showPrevious.value && previous.value.length > 0) {
    // 上一周期按下标与本期对齐，只取本期已有的时段；order 更大的数据集画在下层。
    datasets.push({
      label: t('gotocc.dashboard.usageChart.previousPeriod'),
      data: previous.value.slice(0, points.length).map((point) => metricValueOf(point, metric.value)),
      borderColor: colors.value.muted,
      backgroundColor: colors.value.muted,
      borderWidth: 1.5,
      borderDash: DASH,
      fill: false,
      cubicInterpolationMode: 'monotone' as const,
      pointRadius: 0,
      pointHoverRadius: 3,
      pointHoverBorderWidth: 0,
      pointHoverBackgroundColor: colors.value.muted,
      spanGaps: true,
      order: 1,
      previous: true,
    })
  }
  return { labels: axisLabels.value, datasets }
})

// 纵轴刻度：消费不带货币符号以免挤占宽度。
const formatAxisValue = (value: number): string => {
  if (metric.value === 'tokens') return formatTokensK(value)
  if (metric.value === 'cacheHitRate') return `${value}%`
  if (metric.value === 'cost') return String(Math.round(value * 10000) / 10000)
  return formatNumber(value)
}

const formatValue = (value: number | null): string => {
  if (value === null) return '—'
  if (metric.value === 'tokens') return formatTokensK(value)
  if (metric.value === 'cacheHitRate') return formatPercent(value)
  if (metric.value === 'cost') return formatUsd(value, 4)
  return formatNumber(value >= 100 ? Math.round(value) : Math.round(value * 10) / 10)
}

// 单指标的峰值与均值；均值只统计已结束的时段。
const annotation = computed(() => {
  if (metric.value === 'tokens' || !hasUsage.value) return null
  const values = current.value.map((point) => metricValueOf(point, metric.value))
  let peakIndex = -1
  values.forEach((value, index) => {
    if (value !== null && (peakIndex < 0 || value > (values[peakIndex] as number))) peakIndex = index
  })
  if (peakIndex < 0 || !values[peakIndex]) return null
  const closed = values.filter((value, index): value is number => value !== null && index !== openBucketIndex.value)
  const average = closed.length > 0 ? closed.reduce((sum, value) => sum + value, 0) / closed.length : null
  return { peakIndex, peakValue: values[peakIndex] as number, average }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false as const,
  interaction: { intersect: false, mode: 'index' as const },
  // 标签胶囊画在绘图区上方，留出空间避免被裁切。
  layout: { padding: { top: LABEL_HEIGHT / 2 } },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: colors.value.tooltip,
      titleColor: '#ffffff',
      bodyColor: '#ffffff',
      footerColor: '#ffffff',
      padding: 10,
      cornerRadius: 8,
      usePointStyle: true,
      boxWidth: 8,
      boxHeight: 8,
      boxPadding: 4,
      // Token 按数值从大到小列出；单指标本期排在上一周期之前。
      itemSort: (a: TooltipItem<'line'>, b: TooltipItem<'line'>) => (
        metric.value === 'tokens' ? ((b.raw as number | null) ?? 0) - ((a.raw as number | null) ?? 0) : a.datasetIndex - b.datasetIndex
      ),
      callbacks: {
        title: (items: TooltipItem<'line'>[]) => current.value[items[0].dataIndex].date,
        label: (context: TooltipItem<'line'>) => {
          const dataset = context.dataset as UsageDataset
          const value = formatValue(context.raw as number | null)
          if (!dataset.previous) return `${dataset.label}: ${value}`
          return `${dataset.label} (${shortLabel(previous.value[context.dataIndex].date)}): ${value}`
        },
        footer: (items: TooltipItem<'line'>[]) => {
          if (metric.value !== 'tokens' || items.length < 2) return ''
          const sum = items.reduce((total, item) => total + ((item.raw as number | null) ?? 0), 0)
          return `${t('gotocc.dashboard.usageChart.series.total')}: ${formatTokensK(sum)}`
        },
      },
    },
  },
  scales: {
    x: {
      grid: { display: false },
      ticks: { color: colors.value.text, autoSkip: true, maxTicksLimit: 8, maxRotation: 0, font: { size: CHART_TICK_FONT_SIZE } },
    },
    y: {
      beginAtZero: true,
      // 顶部留白，峰值标签不贴着上沿；命中率固定 0-100%。
      ...(metric.value === 'cacheHitRate' ? { max: 100 } : { grace: '15%' }),
      grid: { color: colors.value.grid, drawTicks: false },
      border: { display: false, dash: [3, 3] },
      ticks: {
        color: colors.value.text,
        padding: 8,
        maxTicksLimit: 5,
        font: { size: CHART_TICK_FONT_SIZE },
        callback: (value: string | number) => formatAxisValue(Number(value)),
      },
    },
  },
}))

// 悬停时间点画一条贯穿绘图区的竖线，画在折线下层。
const crosshairPlugin: Plugin<'line'> = {
  id: 'gcCrosshair',
  beforeDatasetsDraw(chart) {
    const active = chart.getActiveElements()
    if (!active.length) return
    const { ctx, chartArea } = chart
    const x = active[0].element.x
    ctx.save()
    ctx.strokeStyle = colors.value.muted
    ctx.globalAlpha = 0.5
    ctx.lineWidth = 1
    ctx.setLineDash([3, 3])
    ctx.beginPath()
    ctx.moveTo(x, chartArea.top)
    ctx.lineTo(x, chartArea.bottom)
    ctx.stroke()
    ctx.restore()
  },
}

// 圆角标签胶囊，x 为左边缘，y 为上边缘。
const drawLabel = (ctx: CanvasRenderingContext2D, text: string, x: number, y: number, fill: string, color: string) => {
  const width = ctx.measureText(text).width + LABEL_PADDING_X * 2
  ctx.fillStyle = fill
  ctx.beginPath()
  ctx.roundRect(x, y, width, LABEL_HEIGHT, LABEL_HEIGHT / 2)
  ctx.fill()
  ctx.fillStyle = color
  ctx.fillText(text, x + LABEL_PADDING_X, y + LABEL_HEIGHT / 2)
}

// 均值虚线与峰值环，标签胶囊限制在绘图区内。
const drawAnnotations = (chart: Chart) => {
  const info = annotation.value
  const meta = chart.getDatasetMeta(0)
  if (!info || meta.hidden) return
  const { ctx, chartArea, scales } = chart
  ctx.font = `500 ${CHART_TICK_FONT_SIZE + 1}px ${ChartJS.defaults.font.family}`
  ctx.textBaseline = 'middle'

  if (info.average !== null) {
    const y = scales.y.getPixelForValue(info.average)
    ctx.save()
    ctx.strokeStyle = colors.value.muted
    ctx.globalAlpha = 0.6
    ctx.lineWidth = 1
    ctx.setLineDash([2, 4])
    ctx.beginPath()
    ctx.moveTo(chartArea.left, y)
    ctx.lineTo(chartArea.right, y)
    ctx.stroke()
    ctx.restore()
    const text = `${t('gotocc.dashboard.usageChart.average')} · ${formatValue(info.average)}`
    const width = ctx.measureText(text).width + LABEL_PADDING_X * 2
    drawLabel(ctx, text, chartArea.right - width, y - LABEL_HEIGHT / 2, colors.value.surface, colors.value.muted)
  }

  const point = meta.data[info.peakIndex]
  ctx.save()
  ctx.strokeStyle = mainColor.value
  ctx.fillStyle = colors.value.surface
  ctx.lineWidth = 2
  ctx.beginPath()
  ctx.arc(point.x, point.y, 4.5, 0, Math.PI * 2)
  ctx.fill()
  ctx.stroke()
  ctx.restore()
  const text = `${t('gotocc.dashboard.usageChart.peak')} · ${formatValue(info.peakValue)}`
  const width = ctx.measureText(text).width + LABEL_PADDING_X * 2
  const x = Math.min(Math.max(point.x - width / 2, chartArea.left), chartArea.right - width)
  // 上方放不下时改放在峰值点下方。
  const above = point.y - LABEL_GAP - LABEL_HEIGHT
  const y = above >= chartArea.top - LABEL_HEIGHT / 2 ? above : point.y + LABEL_GAP
  drawLabel(ctx, text, x, y, `${mainColor.value}26`, mainColor.value)
}

// 描线进度 0-1，绘图区按进度从左向右裁剪；减少动态效果时始终为 1。
let revealProgress = 1
let revealFrame = 0

// 记录未结束时段末端点的位置；描线未完成或该时段不存在时隐藏。
const updateLivePoint = (chart: Chart) => {
  const index = openBucketIndex.value
  const meta = chart.getDatasetMeta(0)
  const point = index === null || metric.value === 'tokens' || revealProgress < 1 || meta.hidden ? null : meta.data[index]
  if (!point) {
    livePoint.value = null
    return
  }
  if (livePoint.value?.x === point.x && livePoint.value?.y === point.y) return
  livePoint.value = { x: point.x, y: point.y }
}

const revealPlugin: Plugin<'line'> = {
  id: 'gcReveal',
  beforeDatasetsDraw(chart) {
    const { ctx, chartArea } = chart
    ctx.save()
    if (revealProgress < 1) {
      ctx.beginPath()
      ctx.rect(chartArea.left - LABEL_GAP, 0, (chartArea.width + LABEL_GAP * 2) * revealProgress, chart.height)
      ctx.clip()
    }
  },
  afterDatasetsDraw(chart) {
    drawAnnotations(chart)
    chart.ctx.restore()
  },
  afterDraw(chart) {
    updateLivePoint(chart)
  },
}

const chartPlugins = [crosshairPlugin, revealPlugin]

const cancelReveal = () => {
  cancelAnimationFrame(revealFrame)
  revealFrame = 0
}

// 从左向右重新描线，每帧只重绘不重新布局。
const playReveal = () => {
  cancelReveal()
  if (reducedMotion.value === 'reduce') {
    revealProgress = 1
    return
  }
  revealProgress = 0
  const startedAt = performance.now()
  const tick = (now: number) => {
    const progress = Math.min((now - startedAt) / CHART_REVEAL_MS, 1)
    revealProgress = 1 - Math.pow(1 - progress, 3)
    lineRef.value?.chart?.draw()
    revealFrame = progress < 1 ? requestAnimationFrame(tick) : 0
  }
  revealFrame = requestAnimationFrame(tick)
}

// 切换指标或重新取数时递增，图表重建后从头播放描线。
const chartVersion = ref(0)
const chartKey = computed(() => `${metric.value}-${chartVersion.value}`)

watch(current, () => {
  chartVersion.value += 1
  playReveal()
})
watch(metric, playReveal)

onBeforeUnmount(cancelReveal)

const toggleTokenSeries = (key: TokenSeriesKey) => {
  hiddenTokenSeries.value = hiddenTokenSeries.value.includes(key)
    ? hiddenTokenSeries.value.filter((item) => item !== key)
    : [...hiddenTokenSeries.value, key]
}

// 供页面在热力图选中某天后把图表滚动到可见区域。
defineExpose({ chartCardRef })
</script>

<style scoped>
/* 卡片够宽时才显示走势线和环比说明，窄卡片优先保证数字完整。 */
.metric-card {
  container-type: inline-size;
  transition: border-color var(--motion-fast) var(--motion-ease), box-shadow var(--motion-fast) var(--motion-ease);
}

.metric-sparkline,
.metric-delta-label {
  display: none;
}

@container (min-width: 200px) {
  .metric-sparkline,
  .metric-delta-label {
    display: block;
  }
}

.legend-chip {
  @apply inline-flex items-center gap-2 rounded-full border border-gray-200 px-3 py-1 text-xs dark:border-dark-600;
  transition: color var(--motion-fast) var(--motion-ease);
}

.legend-chip-on {
  @apply text-gray-700 dark:text-dark-100;
}

.legend-chip-off {
  @apply text-gray-400 dark:text-dark-500;
}
</style>

<template>
  <div class="card flex min-w-0 flex-col p-4">
    <div class="mb-4 flex min-h-7 flex-wrap items-center justify-between gap-2">
      <div class="flex items-baseline gap-2">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('gotocc.dashboard.usageChart.trendTitle', { metric: t(METRIC_LABELS[metric]) }) }}</h3>
        <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('gotocc.dashboard.usageChart.byDay') }}</span>
      </div>
      <!-- 图例兼开关：上一周期对比线 -->
      <div class="flex flex-wrap gap-2">
        <span class="legend-chip legend-chip-on">
          <span class="h-0.5 w-3 rounded-full" :style="{ backgroundColor: colors.brand }"></span>
          {{ t('gotocc.dashboard.usageChart.currentPeriod') }}
        </span>
        <button type="button" class="legend-chip" :class="showPrevious ? 'legend-chip-on' : 'legend-chip-off'" :aria-pressed="showPrevious" @click="showPrevious = !showPrevious">
          <span class="w-3 border-t-2 border-dashed" :style="{ borderColor: colors.muted }" :class="{ 'opacity-30': !showPrevious }"></span>
          {{ t('gotocc.dashboard.usageChart.previousPeriod') }}
        </button>
      </div>
    </div>
    <div class="relative h-64 sm:h-72">
      <Skeleton v-if="!loaded" height="100%" />
      <div v-else-if="!hasUsage" class="flex h-full flex-col items-center justify-center gap-2 text-sm text-gray-500 dark:text-dark-400">
        <Icon name="chart" size="lg" class="text-gray-300 dark:text-dark-600" />
        {{ t('team.noUsage') }}
      </div>
      <template v-else>
        <Line ref="lineRef" :key="chartKey" :data="chartData" :options="chartOptions" :plugins="chartPlugins" />
        <!-- 今天还在累计，末端用呼吸点标出 -->
        <span v-if="livePoint" class="pointer-events-none absolute" :style="{ left: `${livePoint.x}px`, top: `${livePoint.y}px`, color: colors.brand }" aria-hidden="true">
          <span class="absolute -left-1.5 -top-1.5 h-3 w-3 animate-ping rounded-full bg-current opacity-60 motion-reduce:animate-none"></span>
          <span class="absolute -left-1 -top-1 h-2 w-2 rounded-full bg-current ring-2 ring-white dark:ring-dark-800"></span>
        </span>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePreferredReducedMotion } from '@vueuse/core'
import { CategoryScale, Chart as ChartJS, LinearScale, LineElement, PointElement, Tooltip, type Chart, type ChartDataset, type Plugin, type ScriptableLineSegmentContext, type TooltipItem } from 'chart.js'
import { Line } from 'vue-chartjs'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import type { TeamUsageSummary } from '@/api/team'
import { formatDateLocalInput } from '@/utils/format'
import { CHART_REVEAL_MS } from '@/components/gotocc/dashboard/motion'
import { CHART_TICK_FONT_SIZE, useChartColors } from '@/components/gotocc/dashboard/tones'
import { formatAxis, formatMetric, seriesOf, type TeamMetric } from './teamMetrics'

// 团队趋势：与仪表盘趋势图同一套表现——从左描线、今天的末段虚线、峰值与均值标注、上一周期对比线。

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip)

const METRIC_LABELS: Record<TeamMetric, string> = {
  cost: 'gotocc.dashboard.usageChart.metrics.cost',
  requests: 'gotocc.dashboard.usageChart.metrics.requests',
  tokens: 'gotocc.dashboard.usageChart.metrics.tokens',
}
const DASH = [4, 4]
const LABEL_HEIGHT = 18
const LABEL_PADDING_X = 6
const LABEL_GAP = 10

const props = defineProps<{
  days: string[]
  previousDays: string[]
  current: TeamUsageSummary | null
  previous: TeamUsageSummary | null
  metric: TeamMetric
  loaded: boolean
}>()

type TrendDataset = ChartDataset<'line', number[]> & { previous?: boolean }

const { t } = useI18n()
const { colors } = useChartColors()
const reducedMotion = usePreferredReducedMotion()
const showPrevious = ref(true)
const lineRef = ref<{ chart?: Chart } | null>(null)
const livePoint = ref<{ x: number; y: number } | null>(null)

const values = computed(() => seriesOf(props.current, props.days, props.metric))
const previousValues = computed(() => seriesOf(props.previous, props.previousDays, props.metric))
const hasUsage = computed(() => values.value.some((value) => value > 0))
const formatValue = (value: number) => formatMetric(value, props.metric)

// 范围包含今天时，最后一天还在累计，末段画虚线并且不计入均值。
const openIndex = computed(() => (props.days[props.days.length - 1] === formatDateLocalInput(new Date()) && props.days.length > 1 ? props.days.length - 1 : null))

const chartData = computed(() => {
  const datasets: TrendDataset[] = [{
    label: t(METRIC_LABELS[props.metric]),
    data: values.value,
    borderColor: colors.value.brand,
    backgroundColor: colors.value.brand,
    borderWidth: 2,
    fill: false,
    cubicInterpolationMode: 'monotone' as const,
    segment: { borderDash: (context: ScriptableLineSegmentContext) => (context.p1DataIndex === openIndex.value ? DASH : undefined) },
    pointRadius: 0,
    pointHoverRadius: 5,
    pointHoverBorderWidth: 2,
    pointHoverBorderColor: colors.value.surface,
    pointHoverBackgroundColor: colors.value.brand,
  }]
  if (showPrevious.value && props.previous) {
    // 上一周期按天序号与本期对齐；order 更大的数据集画在下层。
    datasets.push({
      label: t('gotocc.dashboard.usageChart.previousPeriod'),
      data: previousValues.value,
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
      order: 1,
      previous: true,
    })
  }
  return { labels: props.days.map((date) => date.slice(5)), datasets }
})

// 峰值与均值；均值只统计已结束的日子。
const annotation = computed(() => {
  if (!hasUsage.value) return null
  let peakIndex = 0
  values.value.forEach((value, index) => {
    if (value > values.value[peakIndex]) peakIndex = index
  })
  const closed = values.value.filter((_, index) => index !== openIndex.value)
  const average = closed.length > 0 ? closed.reduce((sum, value) => sum + value, 0) / closed.length : null
  return { peakIndex, peakValue: values.value[peakIndex], average }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false as const,
  interaction: { intersect: false, mode: 'index' as const },
  layout: { padding: { top: LABEL_HEIGHT / 2 } },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: colors.value.tooltip,
      padding: 10,
      cornerRadius: 8,
      usePointStyle: true,
      boxWidth: 8,
      boxHeight: 8,
      boxPadding: 4,
      itemSort: (a: TooltipItem<'line'>, b: TooltipItem<'line'>) => a.datasetIndex - b.datasetIndex,
      callbacks: {
        title: (items: TooltipItem<'line'>[]) => props.days[items[0].dataIndex],
        label: (context: TooltipItem<'line'>) => {
          const dataset = context.dataset as TrendDataset
          const label = dataset.previous ? `${dataset.label} (${props.previousDays[context.dataIndex].slice(5)})` : dataset.label
          return `${label}: ${formatValue(context.raw as number)}`
        },
      },
    },
  },
  scales: {
    x: { grid: { display: false }, ticks: { color: colors.value.text, autoSkip: true, maxTicksLimit: 8, maxRotation: 0, font: { size: CHART_TICK_FONT_SIZE } } },
    y: {
      beginAtZero: true,
      grace: '15%',
      grid: { color: colors.value.grid, drawTicks: false },
      border: { display: false },
      ticks: { color: colors.value.text, padding: 8, maxTicksLimit: 5, font: { size: CHART_TICK_FONT_SIZE }, callback: (value: string | number) => formatAxis(Number(value), props.metric) },
    },
  },
}))

// 悬停时间点的竖线，画在折线下层。
const crosshairPlugin: Plugin<'line'> = {
  id: 'gcTeamCrosshair',
  beforeDatasetsDraw(chart) {
    const active = chart.getActiveElements()
    if (!active.length) return
    const { ctx, chartArea } = chart
    ctx.save()
    ctx.strokeStyle = colors.value.muted
    ctx.globalAlpha = 0.5
    ctx.lineWidth = 1
    ctx.setLineDash([3, 3])
    ctx.beginPath()
    ctx.moveTo(active[0].element.x, chartArea.top)
    ctx.lineTo(active[0].element.x, chartArea.bottom)
    ctx.stroke()
    ctx.restore()
  },
}

const drawLabel = (ctx: CanvasRenderingContext2D, text: string, x: number, y: number, fill: string, color: string) => {
  const width = ctx.measureText(text).width + LABEL_PADDING_X * 2
  ctx.fillStyle = fill
  ctx.beginPath()
  ctx.roundRect(x, y, width, LABEL_HEIGHT, LABEL_HEIGHT / 2)
  ctx.fill()
  ctx.fillStyle = color
  ctx.fillText(text, x + LABEL_PADDING_X, y + LABEL_HEIGHT / 2)
}

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
    drawLabel(ctx, text, chartArea.right - (ctx.measureText(text).width + LABEL_PADDING_X * 2), y - LABEL_HEIGHT / 2, colors.value.surface, colors.value.muted)
  }
  const point = meta.data[info.peakIndex]
  ctx.save()
  ctx.strokeStyle = colors.value.brand
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
  const above = point.y - LABEL_GAP - LABEL_HEIGHT
  drawLabel(ctx, text, x, above >= chartArea.top - LABEL_HEIGHT / 2 ? above : point.y + LABEL_GAP, `${colors.value.brand}26`, colors.value.brand)
}

// 描线进度 0-1，绘图区按进度从左向右裁剪；减少动态效果时始终为 1。
let revealProgress = 1
let revealFrame = 0

const updateLivePoint = (chart: Chart) => {
  const meta = chart.getDatasetMeta(0)
  const point = openIndex.value === null || revealProgress < 1 || meta.hidden ? null : meta.data[openIndex.value]
  if (!point) {
    livePoint.value = null
    return
  }
  if (livePoint.value?.x === point.x && livePoint.value?.y === point.y) return
  livePoint.value = { x: point.x, y: point.y }
}

const revealPlugin: Plugin<'line'> = {
  id: 'gcTeamReveal',
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

// 数据或指标变化时重建图表并从头描线。
const chartVersion = ref(0)
const chartKey = computed(() => `${props.metric}-${chartVersion.value}`)
watch(() => props.current, () => {
  chartVersion.value += 1
  playReveal()
})
watch(() => props.metric, playReveal)

onBeforeUnmount(cancelReveal)
</script>

<style scoped>
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

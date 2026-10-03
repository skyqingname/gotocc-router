<template>
  <div class="card flex min-w-0 flex-col p-4">
    <div class="mb-4 flex min-h-7 items-baseline gap-2">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.memberTrendTitle', { metric: t(`gotocc.dashboard.usageChart.metrics.${metric}`) }) }}</h3>
      <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('gotocc.dashboard.usageChart.byDay') }}</span>
    </div>
    <div class="relative h-64">
      <Skeleton v-if="!loaded" height="100%" />
      <div v-else-if="active.length === 0" class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-dark-400">{{ t('team.noUsage') }}</div>
      <Line v-else :data="chartData" :options="chartOptions" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CategoryScale, Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, Tooltip, type TooltipItem } from 'chart.js'
import { Line } from 'vue-chartjs'
import Skeleton from '@/components/common/Skeleton.vue'
import type { TeamMemberUsageSeries } from '@/api/team'
import { CHART_TICK_FONT_SIZE, useChartColors } from '@/components/gotocc/dashboard/tones'
import { formatMetric, memberColor, seriesOf, totalValue, type TeamMetric } from './teamMetrics'

// 各成员按天的走势，颜色与成员排行一致。
ChartJS.register(CategoryScale, Legend, LinearScale, LineElement, PointElement, Tooltip)

const props = defineProps<{ series: TeamMemberUsageSeries[]; days: string[]; metric: TeamMetric; loaded: boolean }>()
const { t } = useI18n()
const { colors } = useChartColors()

// 只画范围内有用量的成员；颜色按成员在序列中的位置取，切换指标时不变。
const active = computed(() => props.series
  .map((item, index) => ({ item, color: memberColor(index) }))
  .filter(({ item }) => totalValue(item.summary, props.metric) > 0))

const chartData = computed(() => ({
  labels: props.days.map((date) => date.slice(5)),
  datasets: active.value.map(({ item, color }) => ({
    label: item.display_name,
    data: seriesOf(item.summary, props.days, props.metric),
    borderColor: color,
    backgroundColor: color,
    borderWidth: 2,
    pointRadius: 0,
    pointHoverRadius: 4,
    cubicInterpolationMode: 'monotone' as const,
  })),
}))

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: {
    legend: { labels: { color: colors.value.text, usePointStyle: true, pointStyle: 'circle', boxWidth: 6, boxHeight: 6 } },
    tooltip: {
      backgroundColor: colors.value.tooltip,
      padding: 10,
      cornerRadius: 8,
      usePointStyle: true,
      boxWidth: 8,
      boxHeight: 8,
      boxPadding: 4,
      itemSort: (a: TooltipItem<'line'>, b: TooltipItem<'line'>) => (b.raw as number) - (a.raw as number),
      callbacks: {
        title: (items: TooltipItem<'line'>[]) => props.days[items[0].dataIndex],
        label: (context: TooltipItem<'line'>) => `${context.dataset.label}: ${formatMetric(context.raw as number, props.metric)}`,
      },
    },
  },
  scales: {
    x: { grid: { display: false }, ticks: { color: colors.value.text, autoSkip: true, maxTicksLimit: 8, maxRotation: 0, font: { size: CHART_TICK_FONT_SIZE } } },
    y: { beginAtZero: true, grid: { color: colors.value.grid }, border: { display: false }, ticks: { color: colors.value.text, maxTicksLimit: 5, font: { size: CHART_TICK_FONT_SIZE }, callback: (value: string | number) => formatMetric(Number(value), props.metric) } },
  },
}))
</script>

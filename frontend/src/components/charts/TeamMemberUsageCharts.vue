<template>
  <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
    <section class="card p-4">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.memberTrend') }}</h3>
      <div v-if="loading" class="mt-4 h-56"><Skeleton height="100%" /></div>
      <div v-else-if="lineData" class="mt-4 h-56"><Line :data="lineData" :options="lineOptions" /></div>
      <div v-else class="flex h-56 items-center justify-center text-sm text-gray-500 dark:text-dark-400">{{ t('team.noUsage') }}</div>
    </section>
    <section class="card p-4">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.memberComparison') }}</h3>
      <div v-if="loading" class="mt-4 h-56"><Skeleton height="100%" /></div>
      <div v-else-if="barData" class="mt-4 h-56"><Bar :data="barData" :options="barOptions" /></div>
      <div v-else class="flex h-56 items-center justify-center text-sm text-gray-500 dark:text-dark-400">{{ t('team.noUsage') }}</div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { BarElement, CategoryScale, Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, Tooltip, type TooltipItem } from 'chart.js'
import { Bar, Line } from 'vue-chartjs'
import Skeleton from '@/components/common/Skeleton.vue'
import { CHART_TICK_FONT_SIZE, useChartColors } from '@/components/gotocc/dashboard/tones'
import type { TeamUsageSummary } from '@/api/team'

ChartJS.register(BarElement, CategoryScale, Legend, LinearScale, LineElement, PointElement, Tooltip)

const props = defineProps<{
  series: Array<{ userID: number; label: string; summary: TeamUsageSummary }>
  loading?: boolean
}>()

const { t } = useI18n()
const { colors: theme } = useChartColors()
// 成员配色从品牌靛色开始，依次取区分度高的色相，同一成员在两张图中同色。
const PALETTE = ['#6366f1', '#0ea5e9', '#10b981', '#f59e0b', '#8b5cf6', '#f43f5e', '#14b8a6', '#ec4899']
const colorOf = (index: number) => PALETTE[index % PALETTE.length]
const visibleSeries = computed(() => props.series.filter((item) => item.summary.request_count > 0 || item.summary.actual_cost > 0))
const dates = computed(() => Array.from(new Set(visibleSeries.value.flatMap((item) => item.summary.daily.map((point) => point.date)))).sort())
const money = (value: number) => `$${value.toFixed(4)}`

const lineData = computed(() => dates.value.length && visibleSeries.value.length ? {
  labels: dates.value.map((date) => date.slice(5)),
  datasets: visibleSeries.value.map((item, index) => {
    const daily = new Map(item.summary.daily.map((point) => [point.date, point.actual_cost]))
    return {
      label: item.label,
      data: dates.value.map((date) => daily.get(date) ?? 0),
      borderColor: colorOf(index),
      backgroundColor: colorOf(index),
      borderWidth: 2,
      pointRadius: 0,
      pointHoverRadius: 4,
      cubicInterpolationMode: 'monotone' as const,
    }
  }),
} : null)

const barData = computed(() => visibleSeries.value.length ? {
  labels: visibleSeries.value.map((item) => item.label),
  datasets: [{
    data: visibleSeries.value.map((item) => item.summary.actual_cost),
    backgroundColor: visibleSeries.value.map((_, index) => colorOf(index)),
    borderRadius: 6,
    maxBarThickness: 22,
  }],
} : null)

const tooltip = computed(() => ({ backgroundColor: theme.value.tooltip, padding: 10, cornerRadius: 8, usePointStyle: true, boxWidth: 8, boxHeight: 8, boxPadding: 4 }))
const ticks = computed(() => ({ color: theme.value.text, font: { size: CHART_TICK_FONT_SIZE } }))
const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: {
    legend: { labels: { color: theme.value.text, usePointStyle: true, pointStyle: 'circle', boxWidth: 6, boxHeight: 6 } },
    tooltip: { ...tooltip.value, callbacks: { label: (context: TooltipItem<'line'>) => `${context.dataset.label}: ${money(context.raw as number)}` } },
  },
  scales: {
    x: { ticks: { ...ticks.value, maxTicksLimit: 8, maxRotation: 0 }, grid: { display: false } },
    y: { beginAtZero: true, ticks: ticks.value, grid: { color: theme.value.grid }, border: { display: false } },
  },
}))
const barOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  indexAxis: 'y' as const,
  plugins: { legend: { display: false }, tooltip: { ...tooltip.value, callbacks: { label: (context: TooltipItem<'bar'>) => money(context.raw as number) } } },
  scales: {
    x: { beginAtZero: true, ticks: ticks.value, grid: { color: theme.value.grid }, border: { display: false } },
    y: { ticks: ticks.value, grid: { display: false }, border: { display: false } },
  },
}))
</script>

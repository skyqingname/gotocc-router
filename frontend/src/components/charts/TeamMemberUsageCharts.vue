<template>
  <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
    <section class="card p-4">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.memberTrend') }}</h3>
      <div v-if="loading" class="flex h-56 items-center justify-center"><LoadingSpinner /></div>
      <div v-else-if="lineData" class="mt-4 h-56"><Line :data="lineData" :options="lineOptions" /></div>
      <div v-else class="flex h-56 items-center justify-center text-sm text-gray-500">{{ t('team.noUsage') }}</div>
    </section>
    <section class="card p-4">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.memberComparison') }}</h3>
      <div v-if="loading" class="flex h-56 items-center justify-center"><LoadingSpinner /></div>
      <div v-else-if="barData" class="mt-4 h-56"><Bar :data="barData" :options="barOptions" /></div>
      <div v-else class="flex h-56 items-center justify-center text-sm text-gray-500">{{ t('team.noUsage') }}</div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { BarElement, CategoryScale, Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, Tooltip } from 'chart.js'
import { Bar, Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { TeamUsageSummary } from '@/api/team'

ChartJS.register(BarElement, CategoryScale, Legend, LinearScale, LineElement, PointElement, Tooltip)

const props = defineProps<{
  series: Array<{ userID: number; label: string; summary: TeamUsageSummary }>
  loading?: boolean
}>()

const { t } = useI18n()
const colors = ['#2563eb', '#059669', '#d97706', '#dc2626', '#7c3aed', '#0891b2', '#db2777', '#4d7c0f']
const visibleSeries = computed(() => props.series.filter((item) => item.summary.request_count > 0 || item.summary.actual_cost > 0))
const dates = computed(() => Array.from(new Set(visibleSeries.value.flatMap((item) => item.summary.daily.map((point) => point.date)))).sort())
const money = (value: number) => `$${Number(value || 0).toFixed(4)}`

const lineData = computed(() => dates.value.length && visibleSeries.value.length ? {
  labels: dates.value,
  datasets: visibleSeries.value.map((item, index) => {
    const daily = new Map(item.summary.daily.map((point) => [point.date, point.actual_cost]))
    return {
      label: item.label,
      data: dates.value.map((date) => daily.get(date) ?? 0),
      borderColor: colors[index % colors.length],
      backgroundColor: `${colors[index % colors.length]}20`,
      pointRadius: 2,
      tension: 0.25,
    }
  }),
} : null)

const barData = computed(() => visibleSeries.value.length ? {
  labels: visibleSeries.value.map((item) => item.label),
  datasets: [{
    data: visibleSeries.value.map((item) => item.summary.actual_cost),
    backgroundColor: visibleSeries.value.map((_, index) => colors[index % colors.length]),
  }],
} : null)

const axis = { ticks: { color: '#6b7280' }, grid: { color: '#e5e7eb' } }
const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: { tooltip: { callbacks: { label: (context: any) => `${context.dataset.label}: ${money(context.raw)}` } } },
  scales: { x: axis, y: { ...axis, beginAtZero: true } },
}))
const barOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  indexAxis: 'y' as const,
  plugins: { legend: { display: false }, tooltip: { callbacks: { label: (context: any) => money(context.raw) } } },
  scales: { x: { ...axis, beginAtZero: true }, y: { ...axis, grid: { display: false } } },
}))
</script>

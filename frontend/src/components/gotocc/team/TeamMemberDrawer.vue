<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="member" class="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true" :aria-label="t('team.memberDetail')" @keydown.esc="emit('close')">
        <div class="drawer-overlay absolute inset-0 bg-black/40" @click="emit('close')"></div>
        <aside class="drawer-panel relative flex h-full w-full max-w-2xl flex-col bg-white shadow-2xl dark:bg-dark-900" :style="{ '--bar-ms': `${TOP_MODEL_BAR_MS}ms` }">
          <header class="flex items-center gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-primary text-sm font-semibold text-white">{{ initialsOf(displayName) }}</span>
            <div class="min-w-0 flex-1">
              <p class="flex items-center gap-2">
                <span class="truncate font-semibold text-gray-900 dark:text-white">{{ displayName }}</span>
                <span class="badge" :class="member.role === 'owner' ? 'badge-primary' : 'badge-gray'">{{ member.role === 'owner' ? t('team.owner') : t('team.member') }}</span>
              </p>
              <p class="truncate text-xs text-gray-500 dark:text-dark-400">{{ member.email }}</p>
            </div>
            <button ref="closeRef" type="button" class="btn btn-ghost btn-sm px-2" :aria-label="t('common.close')" @click="emit('close')"><Icon name="x" size="md" /></button>
          </header>

          <div class="flex-1 space-y-5 overflow-y-auto px-5 py-5">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <DateRangePicker :start-date="range.from" :end-date="range.to" @change="onRangeChange" />
              <button v-if="manageable" type="button" class="btn btn-secondary btn-sm" @click="emit('edit-limits')"><Icon name="edit" size="sm" />{{ t('team.editLimits') }}</button>
            </div>

            <!-- 本期指标 -->
            <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
              <div v-for="metric in metrics" :key="metric.label" class="rounded-xl border border-gray-100 p-3 dark:border-dark-700">
                <p class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400">
                  <span class="flex h-5 w-5 items-center justify-center rounded" :class="[METRIC_TONES[metric.tone].tile, METRIC_TONES[metric.tone].icon]"><Icon :name="metric.icon" size="xs" /></span>
                  {{ metric.label }}
                </p>
                <Skeleton v-if="!loaded" class="mt-2" :width="72" :height="22" />
                <p v-else class="mt-1.5 text-lg font-semibold tabular-nums text-gray-900 dark:text-white">{{ metric.value }}</p>
              </div>
            </div>

            <!-- 限额进度：Owner 不受成员限额约束 -->
            <section v-if="member.role === 'member'" class="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div v-for="limit in limits" :key="limit.key">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-gray-500 dark:text-dark-400">{{ limit.label }}</span>
                  <span class="tabular-nums text-gray-900 dark:text-white">{{ limit.text }}</span>
                </div>
                <div class="mt-1 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div v-if="limit.limit > 0" class="h-full rounded-full" :class="limit.barClass" :style="{ width: `${limit.percent}%` }"></div>
                </div>
              </div>
            </section>

            <!-- 每日消费 -->
            <section>
              <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.dailyCost') }}</h3>
              <div class="h-44">
                <Skeleton v-if="!loaded" height="100%" />
                <Line v-else :data="trendData" :options="trendOptions" />
              </div>
            </section>

            <!-- 按模型拆分 -->
            <section>
              <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.byModel') }}</h3>
              <p v-if="loaded && models.length === 0" class="py-4 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('team.noModelUsage') }}</p>
              <ol v-else class="space-y-2">
                <li v-for="(row, index) in modelRows" :key="`${row.model}-${version}`" class="rounded-lg px-1">
                  <div class="flex min-w-0 items-center gap-2 text-sm">
                    <ModelIcon :model="row.model" size="16px" class="shrink-0" />
                    <span class="min-w-0 flex-1 truncate text-gray-800 dark:text-dark-100">{{ row.model }}</span>
                    <span class="shrink-0 text-xs text-gray-500 dark:text-dark-400">{{ formatNumber(row.request_count) }} {{ t('team.requestsUnit') }} · {{ formatTokensK(row.input_tokens + row.output_tokens) }} Token</span>
                    <span class="w-20 shrink-0 text-right font-medium tabular-nums text-gray-900 dark:text-white">{{ formatTeamCost(row.actual_cost) }}</span>
                  </div>
                  <div class="mt-1.5 flex items-center gap-2 pl-6">
                    <div class="h-1 flex-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                      <div class="share-bar h-full rounded-full" :style="{ '--bar-w': `${row.share}%`, '--bar-delay': `${index * TOP_MODEL_BAR_STEP_MS}ms` }"></div>
                    </div>
                    <span class="w-12 shrink-0 text-right text-xs tabular-nums text-gray-500 dark:text-dark-400">{{ row.share.toFixed(1) }}%</span>
                  </div>
                </li>
              </ol>
            </section>

            <!-- 明细：按 Key 筛选 -->
            <section>
              <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.usageDetails') }}</h3>
                <div class="flex flex-wrap gap-1.5">
                  <button
                    v-for="option in keyChips"
                    :key="String(option.id)"
                    type="button"
                    class="rounded-full border px-2.5 py-0.5 text-xs transition-colors"
                    :class="keyId === option.id ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-500/15 dark:text-primary-300' : 'border-gray-200 text-gray-600 hover:border-gray-300 dark:border-dark-600 dark:text-dark-300'"
                    @click="selectKey(option.id)"
                  >{{ option.name }}</button>
                </div>
              </div>
              <div class="overflow-hidden rounded-xl border border-gray-100 dark:border-dark-700">
                <table class="min-w-full text-sm">
                  <tbody class="divide-y divide-gray-50 dark:divide-dark-700/60" :class="{ 'opacity-60': logsLoading }">
                    <tr v-if="logs.length === 0"><td class="px-4 py-8 text-center text-gray-500 dark:text-dark-400">{{ t('team.noUsage') }}</td></tr>
                    <tr v-for="item in logs" :key="item.id">
                      <td class="whitespace-nowrap px-4 py-2.5 text-xs text-gray-500 dark:text-dark-400">{{ formatDateTime(item.created_at) }}</td>
                      <td class="px-2 py-2.5">
                        <span class="flex min-w-0 items-center gap-2">
                          <ModelIcon :model="item.model" size="14px" />
                          <span class="max-w-40 truncate text-gray-800 dark:text-dark-100">{{ item.model }}</span>
                        </span>
                        <span class="block text-xs text-gray-400 dark:text-dark-500">{{ item.api_key_name }}</span>
                      </td>
                      <td class="whitespace-nowrap px-2 py-2.5 text-right text-xs tabular-nums text-gray-500 dark:text-dark-400">{{ formatTokensK(item.input_tokens) }} / {{ formatTokensK(item.output_tokens) }}</td>
                      <td class="whitespace-nowrap px-4 py-2.5 text-right font-medium tabular-nums text-gray-900 dark:text-white">{{ formatTeamCost(item.actual_cost) }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div v-if="logsTotal > LOG_PAGE_SIZE" class="mt-3">
                <Pagination v-model:page="logsPage" :total="logsTotal" :page-size="LOG_PAGE_SIZE" :show-page-size-selector="false" />
              </div>
            </section>
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { CategoryScale, Chart as ChartJS, Filler, LinearScale, LineElement, PointElement, Tooltip, type ScriptableContext, type TooltipItem } from 'chart.js'
import { Line } from 'vue-chartjs'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Pagination from '@/components/common/Pagination.vue'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import { TOP_MODEL_BAR_MS, TOP_MODEL_BAR_STEP_MS } from '@/components/gotocc/dashboard/motion'
import { CHART_TICK_FONT_SIZE, METRIC_TONES, type MetricTone, useChartColors } from '@/components/gotocc/dashboard/tones'
import { teamAPI, type TeamAPIKey, type TeamMembership, type TeamUsageLog, type TeamUsageModel, type TeamUsageSummary } from '@/api/team'
import { formatDateTime, formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import { formatLimitAmount, formatTeamCost, initialsOf } from './teamFormat'

// 成员用量详情抽屉：本期指标、限额、每日消费、按模型拆分和按 Key 筛选的分页明细。
ChartJS.register(CategoryScale, Filler, LinearScale, LineElement, PointElement, Tooltip)

const LOG_PAGE_SIZE = 10

const props = defineProps<{
  member: TeamMembership | null
  keys: TeamAPIKey[]
  initialRange: { from: string; to: string }
  manageable: boolean
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'edit-limits'): void
}>()

const { t } = useI18n()
const { colors } = useChartColors()

const range = ref({ ...props.initialRange })
const loaded = ref(false)
const summary = ref<TeamUsageSummary | null>(null)
const models = ref<TeamUsageModel[]>([])
const keyId = ref<number | null>(null)
const logs = ref<TeamUsageLog[]>([])
const logsTotal = ref(0)
const logsPage = ref(1)
const logsLoading = ref(false)
const closeRef = ref<HTMLElement | null>(null)
// 每次取到新的模型数据时递增，让占比条重新伸展。
const version = ref(0)

const displayName = computed(() => props.member!.username || props.member!.email.split('@')[0])

type MetricIcon = 'dollar' | 'swap' | 'cube'
const metrics = computed(() => [
  { label: t('team.totalCost'), value: formatTeamCost(summary.value?.actual_cost ?? 0), tone: 'cost' as MetricTone, icon: 'dollar' as MetricIcon },
  { label: t('team.requests'), value: formatNumber(summary.value?.request_count ?? 0), tone: 'requests' as MetricTone, icon: 'swap' as MetricIcon },
  { label: t('team.inputTokens'), value: formatTokensK(summary.value?.input_tokens ?? 0), tone: 'tokens' as MetricTone, icon: 'cube' as MetricIcon },
  { label: t('team.outputTokens'), value: formatTokensK(summary.value?.output_tokens ?? 0), tone: 'cacheHitRate' as MetricTone, icon: 'cube' as MetricIcon },
])

const limits = computed(() => {
  const member = props.member!
  return [
    { key: 'daily', label: t('team.daily'), used: member.daily_usage_usd, limit: member.daily_limit_usd },
    { key: 'weekly', label: t('team.weekly'), used: member.weekly_usage_usd, limit: member.weekly_limit_usd },
    { key: 'monthly', label: t('team.monthly'), used: member.monthly_usage_usd, limit: member.monthly_limit_usd },
  ].map((item) => {
    const percent = item.limit > 0 ? Math.min(100, (item.used / item.limit) * 100) : 0
    return {
      ...item,
      percent,
      text: item.limit > 0 ? `${formatLimitAmount(item.used)} / ${formatLimitAmount(item.limit)}` : `${formatLimitAmount(item.used)} · ${t('team.unlimited')}`,
      barClass: percent >= 100 ? 'bg-red-500' : percent >= 80 ? 'bg-amber-500' : 'bg-emerald-500',
    }
  })
})

// 范围内每一天，接口只返回有消费的日期，补零后折线间隔均匀。
const days = computed(() => {
  const result: string[] = []
  const cursor = new Date(`${range.value.from}T00:00:00`)
  const end = new Date(`${range.value.to}T00:00:00`)
  while (cursor <= end) {
    result.push(`${cursor.getFullYear()}-${String(cursor.getMonth() + 1).padStart(2, '0')}-${String(cursor.getDate()).padStart(2, '0')}`)
    cursor.setDate(cursor.getDate() + 1)
  }
  return result
})

const trendData = computed(() => {
  const byDate = new Map((summary.value?.daily ?? []).map((point) => [point.date, point.actual_cost]))
  return {
    labels: days.value.map((date) => date.slice(5)),
    datasets: [{
      data: days.value.map((date) => byDate.get(date) ?? 0),
      borderColor: colors.value.brand,
      borderWidth: 2,
      fill: true,
      // 线下渐变填充，上浓下淡。
      backgroundColor: (context: ScriptableContext<'line'>) => {
        const { ctx, chartArea } = context.chart
        if (!chartArea) return 'transparent'
        const gradient = ctx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom)
        gradient.addColorStop(0, `${colors.value.brand}40`)
        gradient.addColorStop(1, `${colors.value.brand}00`)
        return gradient
      },
      cubicInterpolationMode: 'monotone' as const,
      pointRadius: 0,
      pointHoverRadius: 4,
      pointHoverBackgroundColor: colors.value.brand,
    }],
  }
})

const trendOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: colors.value.tooltip,
      padding: 10,
      cornerRadius: 8,
      displayColors: false,
      callbacks: {
        title: (items: TooltipItem<'line'>[]) => days.value[items[0].dataIndex],
        label: (context: TooltipItem<'line'>) => formatTeamCost(context.raw as number),
      },
    },
  },
  scales: {
    x: { grid: { display: false }, ticks: { color: colors.value.text, font: { size: CHART_TICK_FONT_SIZE }, maxTicksLimit: 7, maxRotation: 0 } },
    y: { beginAtZero: true, grid: { color: colors.value.grid }, border: { display: false }, ticks: { color: colors.value.text, font: { size: CHART_TICK_FONT_SIZE }, maxTicksLimit: 4 } },
  },
}))

const modelRows = computed(() => {
  const total = models.value.reduce((sum, model) => sum + model.actual_cost, 0)
  return models.value.map((model) => ({ ...model, share: total > 0 ? (model.actual_cost / total) * 100 : 0 }))
})

// 只列该成员的团队 Key。
const keyChips = computed(() => [
  { id: null as number | null, name: t('team.allKeys') },
  ...props.keys.filter((key) => key.user_id === props.member!.user_id).map((key) => ({ id: key.id as number | null, name: key.name })),
])

const scope = () => ({ from: range.value.from, to: range.value.to, member_id: props.member!.user_id })

const loadLogs = async () => {
  logsLoading.value = true
  const query = { ...scope(), limit: LOG_PAGE_SIZE, offset: (logsPage.value - 1) * LOG_PAGE_SIZE, ...(keyId.value === null ? {} : { api_key_id: keyId.value }) }
  const result = await teamAPI.usageLogs(query).finally(() => {
    logsLoading.value = false
  })
  logs.value = result.items
  logsTotal.value = result.total
}

const load = async () => {
  loaded.value = false
  logsPage.value = 1
  const [usage, modelList] = await Promise.all([teamAPI.usage(scope()), teamAPI.usageModels(scope()), loadLogs()])
  summary.value = usage
  models.value = modelList
  version.value += 1
  loaded.value = true
}

const selectKey = (id: number | null) => {
  keyId.value = id
  logsPage.value = 1
  void loadLogs()
}

const onRangeChange = (value: { startDate: string; endDate: string }) => {
  range.value = { from: value.startDate, to: value.endDate }
  void load()
}

watch(logsPage, (page, previous) => {
  if (page !== previous && loaded.value) void loadLogs()
})

// 打开另一位成员时沿用页面的时间范围并重新取数，焦点移到关闭按钮，Esc 可关闭；同一成员的限额更新只刷新显示。
watch(() => props.member?.user_id, (userId) => {
  if (userId === undefined) return
  range.value = { ...props.initialRange }
  keyId.value = null
  void load()
  void nextTick(() => closeRef.value?.focus())
})
</script>

<style scoped>
.drawer-enter-active,
.drawer-leave-active {
  transition: opacity var(--motion-layout) var(--motion-ease);
}

.drawer-enter-active .drawer-panel,
.drawer-leave-active .drawer-panel {
  transition: transform var(--motion-layout) var(--motion-ease);
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}

.drawer-enter-from .drawer-panel,
.drawer-leave-to .drawer-panel {
  transform: translateX(calc(var(--motion-shift) * 10));
}

.share-bar {
  width: var(--bar-w);
  background-image: linear-gradient(90deg, theme('colors.primary.500'), theme('colors.accent.500'));
  animation: share-bar var(--bar-ms) var(--motion-ease) var(--bar-delay) both;
}

@keyframes share-bar {
  from {
    width: 0;
  }
  to {
    width: var(--bar-w);
  }
}

@media (prefers-reduced-motion: reduce) {
  .share-bar {
    animation: none;
  }
}
</style>

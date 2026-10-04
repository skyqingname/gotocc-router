<template>
  <div class="space-y-4" :class="{ 'dash-rise': entering }" :style="dashboardMotionVars" @animationend="onRiseEnd">
    <!-- 筛选在最上方，作用于下方全部区块：指标、趋势、常用模型、成员和明细 -->
    <div class="dash-rise-item flex flex-wrap items-center gap-2" :style="{ '--rise-i': 0 }">
      <div v-if="isOwner" class="w-44"><Select v-model="memberId" :options="memberOptions" searchable /></div>
      <div class="w-44"><Select v-model="keyId" :options="keyOptions" searchable /></div>
      <button v-if="memberId !== null || keyId !== null" type="button" class="btn btn-ghost btn-sm" @click="resetFilters">{{ t('common.reset') }}</button>
    </div>

    <div class="space-y-4 transition-opacity" :class="{ 'opacity-60': busy }">
      <!-- 指标卡：本期合计、迷你走势和较上一周期；点选消费、请求或 Token 切换下方图表和排行 -->
      <div class="dash-rise-item grid grid-cols-2 gap-4 lg:grid-cols-4" :style="{ '--rise-i': 1 }" role="tablist">
        <component
          :is="card.selectable ? 'button' : 'div'"
          v-for="card in cards"
          :key="card.key"
          :type="card.selectable ? 'button' : undefined"
          :role="card.selectable ? 'tab' : undefined"
          :aria-selected="card.selectable ? metric === card.key : undefined"
          class="metric-card card flex min-w-0 flex-col gap-3 overflow-hidden p-4 text-left"
          :class="card.selectable && metric === card.key ? '!border-primary-500 ring-1 ring-primary-500/30 dark:!border-primary-400' : card.selectable ? 'hover:!border-gray-300 dark:hover:!border-dark-500' : ''"
          @click="card.selectable && (metric = card.key as TeamMetric)"
        >
          <div class="flex min-w-0 items-center gap-2">
            <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md" :class="[METRIC_TONES[card.tone].tile, METRIC_TONES[card.tone].icon]">
              <Icon :name="card.icon" size="sm" />
            </span>
            <span class="truncate text-xs font-medium text-gray-500 dark:text-dark-400">{{ card.label }}</span>
          </div>
          <div class="flex min-w-0 items-end justify-between gap-3">
            <Skeleton v-if="!loaded" :width="96" :height="32" />
            <span v-else class="min-w-0 truncate text-2xl font-semibold tracking-tight tabular-nums text-gray-900 dark:text-white">{{ card.value }}</span>
            <UsageSparkline
              v-if="loaded"
              class="metric-sparkline h-8 w-20 shrink-0"
              :class="metric === card.key ? 'text-primary-600 dark:text-primary-400' : 'text-gray-300 dark:text-dark-500'"
              :values="card.series"
            />
          </div>
          <Skeleton v-if="!loaded" :width="120" :height="20" />
          <span v-else class="flex min-w-0 items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
            <template v-if="card.delta">
              <span class="inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-0.5 font-medium tabular-nums" :class="DELTA_CLASSES[card.delta.tone]">
                <Icon v-if="card.delta.tone !== 'flat'" :name="card.delta.tone === 'up' ? 'arrowUp' : 'arrowDown'" size="xs" />
                {{ card.delta.text }}
              </span>
              <span class="metric-delta-label truncate">{{ t('gotocc.dashboard.usageChart.vsPrevious') }}</span>
            </template>
            <span v-else class="truncate">{{ t('gotocc.dashboard.usageChart.noPrevious') }}</span>
          </span>
        </component>
      </div>

      <!-- 团队趋势与常用模型 -->
      <div class="dash-rise-item grid grid-cols-1 gap-4 xl:grid-cols-3" :style="{ '--rise-i': 2 }">
        <TeamTrendChart class="xl:col-span-2" :days="days" :previous-days="previousDays" :current="current" :previous="previous" :metric="metric" :loaded="loaded" />
        <TeamTopModels :models="models" :metric="metric" :loaded="loaded" />
      </div>

      <!-- 成员维度：各成员走势与排行，点成员打开详情抽屉 -->
      <div v-if="isOwner" class="dash-rise-item grid grid-cols-1 gap-4 xl:grid-cols-3" :style="{ '--rise-i': 3 }">
        <TeamMemberTrend class="xl:col-span-2" :series="series" :days="days" :metric="metric" :loaded="loaded" />
        <TeamMemberRanking :series="series" :metric="metric" :loaded="loaded" @open="emit('open-member', $event)" />
      </div>
    </div>

    <TeamUsageScope class="dash-rise-item" :style="{ '--rise-i': 4 }" data-rise-last="true" :range="range" :member-id="memberId" :key-id="keyId" :is-owner="isOwner" data-tour="team-usage-records" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import UsageSparkline from '@/components/gotocc/dashboard/UsageSparkline.vue'
import { COUNT_UP_MS, dashboardMotionVars } from '@/components/gotocc/dashboard/motion'
import { METRIC_TONES, type MetricTone } from '@/components/gotocc/dashboard/tones'
import { useCountUp } from '@/components/gotocc/dashboard/useCountUp'
import { teamAPI, type TeamAPIKey, type TeamMembership, type TeamMemberUsageSeries, type TeamUsageModel, type TeamUsageQuery, type TeamUsageSummary } from '@/api/team'
import TeamMemberRanking from './TeamMemberRanking.vue'
import TeamMemberTrend from './TeamMemberTrend.vue'
import TeamTopModels from './TeamTopModels.vue'
import TeamTrendChart from './TeamTrendChart.vue'
import TeamUsageScope from './TeamUsageScope.vue'
import { daysBetween, formatMetric, previousRangeOf, seriesOf, totalValue, type TeamCardMetric, type TeamMetric } from './teamMetrics'

// 团队概览：仪表盘的多人团队版。筛选、指标卡、趋势、常用模型、成员走势与排行、明细共用同一组条件。

const props = defineProps<{
  range: { from: string; to: string }
  members: TeamMembership[]
  keys: TeamAPIKey[]
  isOwner: boolean
}>()

const emit = defineEmits<{ (event: 'open-member', userId: number): void }>()

const { t } = useI18n()
const memberId = ref<number | null>(null)
const keyId = ref<number | null>(null)
const metric = ref<TeamMetric>('cost')
const current = ref<TeamUsageSummary | null>(null)
const previous = ref<TeamUsageSummary | null>(null)
const models = ref<TeamUsageModel[]>([])
const series = ref<TeamMemberUsageSeries[]>([])
const loaded = ref(false)
// 已有数据后再次取数时为 true，内容变淡而不是回到骨架。
const busy = ref(false)
// 入场动画只在首次挂载时播放。
const entering = ref(true)

const previousRange = computed(() => previousRangeOf(props.range.from, props.range.to))
const days = computed(() => daysBetween(props.range.from, props.range.to))
const previousDays = computed(() => daysBetween(previousRange.value.from, previousRange.value.to))

const memberOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('team.allMembers') },
  ...props.members.map((member) => ({ value: member.user_id, label: member.username || member.email })),
])
// 选了成员时只列该成员的 Key。
const keyOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('team.allKeys') },
  ...props.keys.filter((key) => memberId.value === null || key.user_id === memberId.value).map((key) => ({ value: key.id, label: key.name })),
])

const scope = (range: { from: string; to: string }): TeamUsageQuery => ({
  from: range.from,
  to: range.to,
  ...(memberId.value === null ? {} : { member_id: memberId.value }),
  ...(keyId.value === null ? {} : { api_key_id: keyId.value }),
})

// 用递增序号丢弃过期响应，快速切换范围或筛选时只保留最后一次结果。
let requestSeq = 0

const load = async () => {
  const seq = ++requestSeq
  busy.value = true
  const [now, before, modelList, memberSeries] = await Promise.all([
    teamAPI.usage(scope(props.range)),
    teamAPI.usage(scope(previousRange.value)),
    teamAPI.usageModels(scope(props.range)),
    teamAPI.memberUsage(scope(props.range)),
  ]).finally(() => {
    if (seq === requestSeq) busy.value = false
  })
  if (seq !== requestSeq) return
  current.value = now
  previous.value = before
  models.value = modelList
  series.value = memberSeries
  loaded.value = true
}

watch(memberId, () => {
  keyId.value = null
})
watch([() => props.range.from, () => props.range.to, memberId, keyId], () => void load(), { immediate: true })

const resetFilters = () => {
  memberId.value = null
  keyId.value = null
}

const DELTA_CLASSES = {
  up: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  down: 'bg-red-500/10 text-red-600 dark:text-red-400',
  flat: 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300',
}
type Delta = { text: string; tone: keyof typeof DELTA_CLASSES }

// 金额、请求和 Token 按百分比比较；活跃成员按人数差比较。上一周期为 0 时无法计算百分比。
const deltaOf = (key: TeamCardMetric, now: number, before: number): Delta | null => {
  if (key === 'activeMembers') {
    const diff = now - before
    return { text: `${diff > 0 ? '+' : ''}${diff}`, tone: diff > 0 ? 'up' : diff < 0 ? 'down' : 'flat' }
  }
  if (before <= 0) return null
  const percent = Math.round(((now - before) / before) * 100)
  return { text: `${percent > 0 ? '+' : ''}${percent}%`, tone: percent > 0 ? 'up' : percent < 0 ? 'down' : 'flat' }
}

const totalOf = (key: TeamCardMetric) => (current.value ? totalValue(current.value, key) : 0)
const animated: Record<TeamCardMetric, ReturnType<typeof useCountUp>> = {
  cost: useCountUp(() => totalOf('cost'), COUNT_UP_MS),
  requests: useCountUp(() => totalOf('requests'), COUNT_UP_MS),
  tokens: useCountUp(() => totalOf('tokens'), COUNT_UP_MS),
  activeMembers: useCountUp(() => totalOf('activeMembers'), COUNT_UP_MS),
}

type CardIcon = 'dollar' | 'swap' | 'cube' | 'users'
const CARD_DEFS: Array<{ key: TeamCardMetric; label: string; tone: MetricTone; icon: CardIcon }> = [
  { key: 'cost', label: 'team.totalCost', tone: 'cost', icon: 'dollar' },
  { key: 'requests', label: 'team.requests', tone: 'requests', icon: 'swap' },
  { key: 'tokens', label: 'team.tokens', tone: 'tokens', icon: 'cube' },
  { key: 'activeMembers', label: 'team.activeMembers', tone: 'cacheHitRate', icon: 'users' },
]

const cards = computed(() => CARD_DEFS.map((def) => ({
  ...def,
  label: t(def.label),
  selectable: def.key !== 'activeMembers',
  value: formatMetric(animated[def.key].value, def.key),
  series: seriesOf(current.value, days.value, def.key),
  delta: previous.value ? deltaOf(def.key, totalOf(def.key), totalValue(previous.value, def.key)) : null,
})))

const onRiseEnd = (event: AnimationEvent) => {
  if ((event.target as HTMLElement).dataset.riseLast) entering.value = false
}
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

/* 区块入场：自下方 8px 上移并淡入，相邻区块错开一个步长。 */
.dash-rise .dash-rise-item,
.dash-rise :deep(.dash-rise-item) {
  animation: dash-rise var(--dash-rise-ms) var(--motion-ease) backwards;
  animation-delay: calc(var(--rise-i, 0) * var(--dash-rise-step-ms));
}

@keyframes dash-rise {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .dash-rise .dash-rise-item,
  .dash-rise :deep(.dash-rise-item) {
    animation: none;
  }
}
</style>

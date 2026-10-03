<template>
  <div class="member-row" :class="{ 'member-row-open': expanded }">
    <!-- 收起时一行摘要：身份、日/周/月限额进度、本期消费和最近活跃 -->
    <div class="flex items-center gap-3 px-4 py-3 sm:px-5">
      <button type="button" class="flex min-w-0 flex-1 items-center gap-3 text-left" :aria-expanded="expanded" @click="expanded = !expanded">
        <Icon name="chevronRight" size="sm" class="chevron shrink-0 text-gray-400" />
        <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-gradient-primary text-xs font-semibold text-white">{{ initialsOf(displayName) }}</span>
        <span class="min-w-0 sm:w-48 sm:flex-none">
          <span class="flex items-center gap-2">
            <span class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ displayName }}</span>
            <span class="badge shrink-0" :class="member.role === 'owner' ? 'badge-primary' : 'badge-gray'">{{ member.role === 'owner' ? t('team.owner') : t('team.member') }}</span>
          </span>
          <span class="block truncate text-xs text-gray-500 dark:text-dark-400">{{ member.email }}</span>
        </span>
        <span v-if="member.role === 'owner'" class="hidden min-w-0 flex-1 text-xs text-gray-500 dark:text-dark-400 lg:block">{{ t('team.ownerNoLimit') }}</span>
        <span v-else class="hidden min-w-0 flex-1 grid-cols-3 gap-4 lg:grid">
          <span v-for="limit in limits" :key="limit.key" class="min-w-0">
            <span class="flex items-baseline justify-between gap-2 text-[11px]">
              <span class="text-gray-500 dark:text-dark-400">{{ limit.label }}</span>
              <span class="truncate tabular-nums text-gray-700 dark:text-dark-200">{{ limit.text }}</span>
            </span>
            <span class="mt-1 block h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
              <span v-if="limit.limit > 0" class="block h-full rounded-full transition-[width] duration-500" :class="limit.barClass" :style="{ width: `${limit.percent}%` }"></span>
            </span>
          </span>
        </span>
      </button>
      <span class="hidden w-28 shrink-0 text-right sm:block">
        <span class="block text-sm font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatTeamCost(summary?.actual_cost ?? 0) }}</span>
        <span class="block text-[11px] text-gray-500 dark:text-dark-400">{{ t('team.periodCost') }}</span>
      </span>
      <span class="hidden w-28 shrink-0 text-right text-xs text-gray-500 dark:text-dark-400 md:block" :title="member.last_active_at ? formatDateTime(member.last_active_at) : undefined">
        {{ member.last_active_at ? formatRelativeTime(member.last_active_at, locale) : t('team.neverActive') }}
      </span>
      <!-- 操作统一收进 ⋯ 菜单 -->
      <div ref="menuRef" class="relative shrink-0">
        <button type="button" class="btn btn-ghost btn-sm px-2" :aria-label="t('team.memberActions')" :aria-expanded="menuOpen" @click="menuOpen = !menuOpen">
          <Icon name="more" size="sm" />
        </button>
        <Transition name="dropdown-fade">
          <div v-if="menuOpen" class="dropdown right-0 z-20 mt-1 w-44 py-0">
            <div class="menu-section">
              <button type="button" class="menu-item" @click="act('details')"><Icon name="chartBar" size="sm" />{{ t('team.viewUsage') }}</button>
              <template v-if="manageable">
                <button type="button" class="menu-item" @click="act('edit-limits')"><Icon name="edit" size="sm" />{{ t('team.editLimits') }}</button>
                <button type="button" class="menu-item" @click="act('transfer')"><Icon name="swap" size="sm" />{{ t('team.transfer') }}</button>
              </template>
            </div>
            <div v-if="manageable" class="menu-section">
              <button type="button" class="menu-item menu-item-danger" @click="act('remove')"><Icon name="trash" size="sm" />{{ t('team.remove') }}</button>
            </div>
          </div>
        </Transition>
      </div>
    </div>

    <!-- 展开：限额明细、本期用量和每日消费走势 -->
    <div class="collapse-body" :inert="!expanded">
      <div class="collapse-inner">
        <div class="grid gap-4 border-t border-gray-100 bg-gray-50/60 px-4 py-4 dark:border-dark-700 dark:bg-dark-900/40 sm:px-5 md:grid-cols-3">
          <div class="space-y-3">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('team.limitProgress') }}</p>
            <p v-if="member.role === 'owner'" class="text-xs text-gray-500 dark:text-dark-400">{{ t('team.ownerNoLimit') }}</p>
            <div v-for="limit in limits" v-else :key="limit.key">
              <div class="flex items-center justify-between text-xs">
                <span class="text-gray-600 dark:text-dark-300">{{ limit.label }}</span>
                <span class="tabular-nums text-gray-900 dark:text-white">{{ limit.text }}</span>
              </div>
              <div class="mt-1 h-1.5 overflow-hidden rounded-full bg-gray-200/70 dark:bg-dark-700">
                <div v-if="limit.limit > 0" class="h-full rounded-full" :class="limit.barClass" :style="{ width: `${limit.percent}%` }"></div>
              </div>
            </div>
          </div>
          <div>
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('team.periodUsage') }}</p>
            <dl class="mt-3 grid grid-cols-2 gap-3 text-xs">
              <div v-for="metric in metrics" :key="metric.label">
                <dt class="text-gray-500 dark:text-dark-400">{{ metric.label }}</dt>
                <dd class="mt-0.5 text-sm font-semibold tabular-nums text-gray-900 dark:text-white">{{ metric.value }}</dd>
              </div>
            </dl>
          </div>
          <div class="flex flex-col">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('team.dailyCost') }}</p>
            <UsageSparkline v-if="expanded" class="mt-3 h-12 w-full text-primary-600 dark:text-primary-400" :values="dailyCost" />
            <button type="button" class="btn btn-secondary btn-sm mt-auto self-start" @click="emit('details')">
              <Icon name="chartBar" size="sm" />{{ t('team.viewUsage') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import UsageSparkline from '@/components/gotocc/dashboard/UsageSparkline.vue'
import type { TeamMembership, TeamUsageSummary } from '@/api/team'
import { formatDateTime, formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import { formatLimitAmount, formatRelativeTime, formatTeamCost, initialsOf } from './teamFormat'

// 团队成员行：收起只占一行，展开看限额明细和本期用量；操作收进菜单。
type MemberAction = 'details' | 'edit-limits' | 'transfer' | 'remove'

const props = defineProps<{
  member: TeamMembership
  // 本期（页面所选范围）该成员的用量汇总；范围内没有记录时为空。
  summary: TeamUsageSummary | undefined
  // 本期每天的日期，用于把每日消费补齐成连续走势。
  days: string[]
  // 当前用户是 Owner 且该行是普通成员时可管理。
  manageable: boolean
}>()

const emit = defineEmits<{
  (event: 'details'): void
  (event: 'edit-limits'): void
  (event: 'transfer'): void
  (event: 'remove'): void
}>()

const { t, locale } = useI18n()
const expanded = ref(false)
const menuOpen = ref(false)
const menuRef = ref<HTMLElement | null>(null)

const displayName = computed(() => props.member.username || props.member.email.split('@')[0])

const limits = computed(() => [
  { key: 'daily', label: t('team.daily'), used: props.member.daily_usage_usd, limit: props.member.daily_limit_usd },
  { key: 'weekly', label: t('team.weekly'), used: props.member.weekly_usage_usd, limit: props.member.weekly_limit_usd },
  { key: 'monthly', label: t('team.monthly'), used: props.member.monthly_usage_usd, limit: props.member.monthly_limit_usd },
].map((item) => {
  // 限额为 0 表示不限额。
  const percent = item.limit > 0 ? Math.min(100, (item.used / item.limit) * 100) : 0
  return {
    ...item,
    percent,
    text: item.limit > 0 ? `${formatLimitAmount(item.used)} / ${formatLimitAmount(item.limit)}` : `${formatLimitAmount(item.used)} · ${t('team.unlimited')}`,
    barClass: percent >= 100 ? 'bg-red-500' : percent >= 80 ? 'bg-amber-500' : 'bg-emerald-500',
  }
}))

const metrics = computed(() => [
  { label: t('team.totalCost'), value: formatTeamCost(props.summary?.actual_cost ?? 0) },
  { label: t('team.requests'), value: formatNumber(props.summary?.request_count ?? 0) },
  { label: t('team.inputTokens'), value: formatTokensK(props.summary?.input_tokens ?? 0) },
  { label: t('team.outputTokens'), value: formatTokensK(props.summary?.output_tokens ?? 0) },
])

const dailyCost = computed(() => {
  const byDate = new Map((props.summary?.daily ?? []).map((point) => [point.date, point.actual_cost]))
  return props.days.map((date) => byDate.get(date) ?? 0)
})

const act = (action: MemberAction) => {
  menuOpen.value = false
  if (action === 'details') emit('details')
  else if (action === 'edit-limits') emit('edit-limits')
  else if (action === 'transfer') emit('transfer')
  else emit('remove')
}

const handleClickOutside = (event: MouseEvent) => {
  if (menuRef.value && !menuRef.value.contains(event.target as Node)) menuOpen.value = false
}

onMounted(() => document.addEventListener('click', handleClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', handleClickOutside))
</script>

<style scoped>
.chevron {
  transition: transform var(--motion-normal) var(--motion-ease);
}

.member-row-open .chevron {
  transform: rotate(90deg);
}

/* 展开收起用 grid 行高过渡，内容高度不固定也能平滑动画。 */
.collapse-body {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows var(--motion-layout) var(--motion-ease);
}

.member-row-open .collapse-body {
  grid-template-rows: 1fr;
}

.collapse-inner {
  min-height: 0;
  overflow: hidden;
}
</style>

<template>
  <div v-if="cards.length > 0" class="card p-4">
    <div class="mb-4 flex items-center justify-between gap-2">
      <h3 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
        <Icon name="server" size="sm" class="text-primary-600 dark:text-primary-400" />
        {{ t('gotocc.dashboard.platforms.title') }}
      </h3>
      <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('gotocc.dashboard.platforms.count', { count: platformCount }) }}</span>
    </div>
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <div
        v-for="card in cards"
        :key="card.platform"
        class="rounded-xl border p-3"
        :class="card.isOther ? 'border-dashed border-gray-300 dark:border-dark-600' : 'border-gray-200 dark:border-dark-700'"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="flex min-w-0 items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
            <span class="h-2.5 w-2.5 shrink-0 rounded-full" :style="{ backgroundColor: card.isOther ? undefined : platformAccentColor(card.platform) }" :class="{ 'bg-gray-300 dark:bg-dark-500': card.isOther }"></span>
            <span class="truncate">{{ card.isOther ? t('gotocc.dashboard.platforms.other') : PLATFORM_LABELS[card.platform] ?? card.platform }}</span>
          </span>
          <span class="text-base font-semibold tabular-nums text-gray-900 dark:text-white" :title="t('gotocc.dashboard.platforms.total')">{{ formatCostAuto(card.total_actual_cost) }}</span>
        </div>
        <dl class="mt-3 grid grid-cols-3 gap-2 text-xs">
          <div>
            <dt class="text-gray-500 dark:text-dark-400">{{ t('gotocc.dashboard.platforms.today') }}</dt>
            <dd class="mt-0.5 font-medium tabular-nums text-gray-800 dark:text-dark-100">{{ formatCostAuto(card.today_actual_cost) }}</dd>
          </div>
          <div>
            <dt class="text-gray-500 dark:text-dark-400">{{ t('gotocc.dashboard.platforms.requests') }}</dt>
            <dd class="mt-0.5 font-medium tabular-nums text-gray-800 dark:text-dark-100">{{ card.total_requests > 0 ? formatNumber(card.total_requests) : '—' }}</dd>
          </div>
          <div>
            <dt class="text-gray-500 dark:text-dark-400">{{ t('gotocc.dashboard.platforms.tokens') }}</dt>
            <dd class="mt-0.5 font-medium tabular-nums text-gray-800 dark:text-dark-100">
              {{ card.total_tokens > 0 ? formatTokensK(card.total_tokens) : '—' }}
              <span v-if="card.total_tokens > 0 && formatTokenShare(card.total_tokens, stats?.total_tokens)" class="font-normal text-gray-400">({{ formatTokenShare(card.total_tokens, stats?.total_tokens) }})</span>
            </dd>
          </div>
        </dl>

        <!-- 额度：至少一个窗口配置了上限时显示；上限为 0 表示该平台已禁用 -->
        <div v-if="card.windows.length > 0" class="mt-3 space-y-2 border-t border-gray-100 pt-3 dark:border-dark-700">
          <div v-for="window in card.windows" :key="window.key">
            <div class="flex items-center justify-between text-xs">
              <span class="text-gray-500 dark:text-dark-400">{{ t(`gotocc.dashboard.platforms.${window.key}`) }}</span>
              <span v-if="window.limit === 0" class="font-medium text-red-500">{{ t('gotocc.dashboard.platforms.disabled') }}</span>
              <span v-else class="tabular-nums text-gray-700 dark:text-dark-200">{{ formatUsd(window.usage, 2) }} / {{ formatUsd(window.limit, 2) }}</span>
            </div>
            <div v-if="window.limit > 0" class="mt-1 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
              <div class="quota-bar h-full rounded-full" :class="quotaBarClass(window.percent)" :style="{ '--bar-w': `${window.percent}%` }"></div>
            </div>
            <p v-if="window.resetsAt" class="mt-0.5 text-[10px] text-gray-400 dark:text-dark-500">{{ t('gotocc.dashboard.platforms.resetsAt', { time: formatResetTime(window.resetsAt) }) }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { usageAPI, type PlatformDashboardStats, type UserDashboardStats } from '@/api/usage'
import { getMyPlatformQuotas } from '@/api/user'
import type { PlatformQuotaItem } from '@/types'
import { formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import { platformAccentColor } from '@/utils/platformColors'
import { formatTokenShare } from '@/utils/tokenShare'
import { formatCostAuto, formatUsd } from './format'

// 各平台累计用量与日/周/月额度，数据与原仪表盘的平台拆分一致；各平台之和小于总值的差额显示为「其他」。

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  kimi: 'Kimi',
  zhipu: 'Zhipu GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
}
const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok']
// 差额小于该值视为舍入误差，不显示「其他」。
const OTHER_THRESHOLD = 0.0001
const WINDOWS = ['daily', 'weekly', 'monthly'] as const

interface QuotaWindow {
  key: (typeof WINDOWS)[number]
  limit: number
  usage: number
  percent: number
  resetsAt: string | null | undefined
}

const { t } = useI18n()
const stats = ref<UserDashboardStats | null>(null)
const quotas = ref<PlatformQuotaItem[]>([])

const quotaWindows = (quota: PlatformQuotaItem | undefined): QuotaWindow[] => {
  if (!quota) return []
  return WINDOWS.flatMap((key) => {
    const limit = quota[`${key}_limit_usd`]
    if (limit === null) return []
    const usage = quota[`${key}_usage_usd`]
    return [{ key, limit, usage, percent: limit > 0 ? Math.min(100, Math.round((usage / limit) * 100)) : 0, resetsAt: quota[`${key}_window_resets_at`] }]
  })
}

const rank = (platform: string) => {
  const index = PLATFORM_ORDER.indexOf(platform)
  return index === -1 ? PLATFORM_ORDER.length : index
}

const cards = computed(() => {
  const current = stats.value
  if (!current) return []
  const byPlatform = new Map<string, PlatformDashboardStats>((current.by_platform ?? []).map((item) => [item.platform, item]))
  const byQuota = new Map(quotas.value.map((quota) => [quota.platform as string, quota]))
  // 卡片 = 有用量的平台 ∪ 至少配置了一档上限的平台。
  const platforms = new Set<string>(byPlatform.keys())
  for (const [platform, quota] of byQuota) {
    if (quotaWindows(quota).length > 0) platforms.add(platform)
  }
  const list = [...platforms]
    .sort((a, b) => rank(a) - rank(b) || a.localeCompare(b))
    .map((platform) => {
      const stat = byPlatform.get(platform)
      return {
        platform,
        isOther: false,
        total_actual_cost: stat?.total_actual_cost ?? 0,
        today_actual_cost: stat?.today_actual_cost ?? 0,
        total_requests: stat?.total_requests ?? 0,
        total_tokens: stat?.total_tokens ?? 0,
        windows: quotaWindows(byQuota.get(platform)),
      }
    })
  // 后端按平台聚合时会略去无法归属平台的用量，差额单独列出，保证与总值对得上。
  const diffTotal = current.total_actual_cost - list.reduce((sum, card) => sum + card.total_actual_cost, 0)
  const diffToday = current.today_actual_cost - list.reduce((sum, card) => sum + card.today_actual_cost, 0)
  const diffTokens = current.total_tokens - list.reduce((sum, card) => sum + card.total_tokens, 0)
  if (diffTotal > OTHER_THRESHOLD || diffToday > OTHER_THRESHOLD || diffTokens > 0) {
    list.push({ platform: '__other__', isOther: true, total_actual_cost: Math.max(0, diffTotal), today_actual_cost: Math.max(0, diffToday), total_requests: 0, total_tokens: Math.max(0, diffTokens), windows: [] })
  }
  return list
})

const platformCount = computed(() => cards.value.filter((card) => !card.isOther).length)

const quotaBarClass = (percent: number) => (percent >= 95 ? 'bg-red-500' : percent >= 75 ? 'bg-amber-500' : 'bg-emerald-500')

const formatResetTime = (iso: string) => new Date(iso).toLocaleString(undefined, { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false })

const load = async () => {
  const [statsResult, quotaResult] = await Promise.all([usageAPI.getDashboardStats(), getMyPlatformQuotas()])
  stats.value = statsResult
  quotas.value = quotaResult.platform_quotas ?? []
}

onMounted(() => void load())

defineExpose({ reload: load })
</script>

<style scoped>
.quota-bar {
  width: var(--bar-w);
  animation: quota-bar var(--dash-top-model-bar-ms) var(--motion-ease) both;
}

@keyframes quota-bar {
  from {
    width: 0;
  }
  to {
    width: var(--bar-w);
  }
}

@media (prefers-reduced-motion: reduce) {
  .quota-bar {
    animation: none;
  }
}
</style>

<template>
  <!-- 最近请求：近 7 天最新几条，状态、模型、首字/耗时、费用。管理员允许查看错误请求时把失败的也排进来。 -->
  <section class="card recent-card" :aria-busy="loading">
    <header class="recent-head">
      <h2 class="recent-title"><Icon name="clock" size="sm" class="text-primary-600 dark:text-primary-400" />{{ t('dashboardFocus.recent.title') }}</h2>
      <RouterLink to="/usage" class="text-link inline-flex items-center gap-1 text-xs font-medium">{{ t('gotocc.dashboard.topModels.viewAll') }}<Icon name="chevronRight" size="xs" class="row-arrow" /></RouterLink>
    </header>
    <div v-if="loading && !rows.length" class="space-y-2 p-4"><div v-for="index in 4" :key="index" class="gc-skeleton h-11 w-full"></div></div>
    <div v-else-if="!rows.length" class="recent-empty">
      <Icon name="chart" size="lg" class="text-gray-300 dark:text-dark-600" />
      <p class="text-sm font-medium">{{ t('dashboard.noUsageRecords') }}</p>
      <p class="text-xs surface-muted">{{ t('dashboard.startUsingApi') }}</p>
    </div>
    <div v-else class="record-table-scroll">
      <table class="record-table w-full min-w-[560px] text-left text-sm">
        <thead><tr class="surface-muted">
          <th class="px-3 py-2 font-medium" :data-tip="t('dashboardFocus.recent.statusTip')">{{ t('dashboardFocus.recent.status') }}</th>
          <th class="px-3 py-2 font-medium" :data-tip="t('consoleTips.usage.columns.model')">{{ t('dashboardFocus.recent.model') }}</th>
          <th class="px-3 py-2 font-medium" :data-tip="t('dashboardFocus.recent.latencyTip')">{{ t('dashboardFocus.recent.latency') }}</th>
          <th class="px-3 py-2 text-right font-medium" :data-tip="t('dashboardFocus.recent.costTip')">{{ t('dashboardFocus.recent.cost') }}</th>
        </tr></thead>
        <tbody>
          <tr v-for="row in rows" :key="row.key">
            <td class="px-3 py-2.5">
              <span class="recent-status" :class="row.ok ? 'recent-ok' : 'recent-fail'" :data-tip="row.statusTip"><i aria-hidden="true"></i>{{ row.status }}</span>
            </td>
            <td class="max-w-[16rem] px-3 py-2.5">
              <p class="truncate font-medium">{{ row.model || '—' }}</p>
              <p class="text-xs surface-muted" :data-tip="timeTip(row.createdAt)">{{ formatRelativeTime(row.createdAt) }}</p>
            </td>
            <td class="px-3 py-2.5 gc-num text-xs">{{ row.latency }}</td>
            <td class="px-3 py-2.5 text-right gc-num" :data-tip="row.costTip">{{ row.cost }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import { listMyErrorRequests, usageAPI } from '@/api/usage'
import { useAppStore } from '@/stores/app'
import { exactCredit, timeTip } from '@/utils/consoleTips'
import { formatDateLocalInput, formatRelativeTime } from '@/utils/format'
import { usageMoney } from '@/utils/usageInspection'
import type { UsageLog, UserErrorRequest } from '@/types'

// 最近请求固定看近 7 天，列出最新 RECENT_COUNT 条。
const RECENT_DAYS = 7
const RECENT_COUNT = 6

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const usage = ref<UsageLog[]>([])
const errors = ref<UserErrorRequest[]>([])
const errorViewEnabled = computed(() => appStore.cachedPublicSettings?.allow_user_view_error_requests ?? false)

const seconds = (ms: number) => (ms < 1000 ? `${Math.round(ms)}ms` : `${(ms / 1000).toFixed(2)}s`)

const rows = computed(() => [
  ...usage.value.map((log) => ({
    key: `u${log.id}`,
    ok: true,
    status: t('dashboardFocus.recent.success'),
    statusTip: t('dashboardFocus.recent.successTip'),
    model: log.model,
    createdAt: log.created_at,
    latency: `${log.first_token_ms == null ? '—' : seconds(log.first_token_ms)} / ${log.duration_ms == null ? '—' : seconds(log.duration_ms)}`,
    cost: usageMoney(log.actual_cost),
    costTip: exactCredit(log.actual_cost),
  })),
  ...errors.value.map((item) => ({
    key: `e${item.id}`,
    ok: false,
    status: String(item.status_code),
    statusTip: item.message,
    model: item.model,
    createdAt: item.created_at,
    latency: '—',
    cost: '—',
    costTip: t('dashboardFocus.recent.failedCost'),
  })),
].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()).slice(0, RECENT_COUNT))

const load = async () => {
  loading.value = true
  const start = new Date()
  start.setDate(start.getDate() - (RECENT_DAYS - 1))
  const startDate = formatDateLocalInput(start)
  const endDate = formatDateLocalInput(new Date())
  const [usageResult, errorResult] = await Promise.all([
    usageAPI.getByDateRange(startDate, endDate),
    errorViewEnabled.value ? listMyErrorRequests({ page: 1, page_size: RECENT_COUNT, start_date: startDate, end_date: endDate }) : null,
  ]).finally(() => {
    loading.value = false
  })
  usage.value = usageResult.items.slice(0, RECENT_COUNT)
  errors.value = errorResult?.items ?? []
}

onMounted(() => { void load() })

defineExpose({ reload: load })
</script>

<style scoped>
.recent-card { overflow: hidden; border-radius: var(--surface-radius); }
.recent-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 14px 18px 6px; }
.recent-title { display: flex; align-items: center; gap: 8px; font-size: 14px; font-weight: 600; color: var(--surface-text); }
.recent-empty { display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 32px 16px 40px; text-align: center; }
.recent-status { display: inline-flex; align-items: center; gap: 6px; border-radius: var(--surface-radius-tag); padding: 1px 8px; font-size: 11px; font-weight: 600; }
.recent-status > i { width: 6px; height: 6px; border-radius: 9999px; background: currentColor; }
.recent-ok { color: theme('colors.emerald.700'); background: rgb(16 185 129 / 12%); }
.recent-fail { color: theme('colors.red.700'); background: rgb(239 68 68 / 12%); }
.dark .recent-ok { color: theme('colors.emerald.300'); }
.dark .recent-fail { color: theme('colors.red.300'); }
</style>

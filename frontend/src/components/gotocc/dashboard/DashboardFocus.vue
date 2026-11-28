<template>
  <!-- 仪表盘第一行「重点」：余额（最大）、今天、服务质量。数据每分钟刷新，页面隐藏时暂停。 -->
  <div class="focus-grid" :aria-busy="!loaded">
    <!-- 余额 -->
    <section class="card gc-stat focus-card focus-balance">
      <header class="focus-head">
        <span class="focus-icon" :class="[METRIC_TONES.cost.tile, METRIC_TONES.cost.icon]"><Icon name="creditCard" size="sm" /></span>
        <span class="focus-title">{{ isCustomer ? t('dashboardFocus.balance.customerTitle') : t('dashboardFocus.balance.title') }}</span>
      </header>
      <div class="focus-hero" :data-tip="balanceTip">
        <span aria-hidden="true">{{ money(animatedBalance) }}</span>
        <span class="sr-only">{{ money(balance) }}</span>
      </div>
      <Skeleton v-if="!loaded" :width="220" :height="16" />
      <p v-else class="focus-line" :data-tip="t('dashboardFocus.balance.runwayTip')">
        <template v-if="weekCost > 0">
          {{ t('dashboardFocus.balance.average', { amount: formatCostAuto(dailyAverage) }) }}<span class="focus-sep">·</span><strong :class="runwayTone">{{ runwayText }}</strong>
        </template>
        <template v-else>{{ t('dashboardFocus.balance.noSpend') }}</template>
      </p>
      <ul v-if="subscriptionRows.length" class="focus-subs">
        <li v-for="row in subscriptionRows" :key="row.id" class="focus-sub" :data-tip="row.tip">
          <Icon name="calendar" size="xs" class="shrink-0 text-primary-500" />
          <span class="min-w-0 truncate font-medium">{{ row.name }}</span>
          <span class="focus-sep">·</span><span class="shrink-0">{{ row.expires }}</span>
          <template v-if="row.remaining"><span class="focus-sep">·</span><span class="shrink-0">{{ row.remaining }}</span></template>
        </li>
      </ul>
      <div v-if="!supportMode" class="focus-actions">
        <template v-if="isCustomer">
          <RouterLink to="/credits" class="btn btn-primary btn-sm">{{ t('dashboardFocus.balance.credits') }}</RouterLink>
        </template>
        <template v-else>
          <RouterLink v-if="paymentEnabled" :to="{ path: '/wallet', query: { tab: 'recharge' } }" class="btn btn-primary btn-sm" :data-tip="t('dashboardFocus.balance.rechargeTip')">{{ t('dashboardFocus.balance.recharge') }}</RouterLink>
          <RouterLink :to="{ path: '/wallet', query: { tab: 'redeem' } }" class="btn btn-sm" :class="paymentEnabled ? 'btn-secondary' : 'btn-primary'" :data-tip="t('dashboardFocus.balance.redeemTip')">{{ t('dashboardFocus.balance.redeem') }}</RouterLink>
        </template>
      </div>
    </section>

    <!-- 今天 -->
    <section class="card gc-stat focus-card">
      <header class="focus-head">
        <span class="focus-icon" :class="[METRIC_TONES.requests.tile, METRIC_TONES.requests.icon]"><Icon name="clock" size="sm" /></span>
        <span class="focus-title">{{ t('dashboardFocus.today.title') }}</span>
        <span class="focus-live" :data-tip="t('dashboardFocus.today.liveTip')">
          <i class="focus-live-dot" aria-hidden="true"></i>RPM {{ formatRpm(stats?.rpm ?? 0) }} · TPM {{ formatTokensK(stats?.tpm ?? 0) }}
        </span>
      </header>
      <div class="focus-value" :data-tip="`${t('consoleTips.dashboard.todayCost')}\n${exactCredit(stats?.today_actual_cost ?? 0)}`">
        <Skeleton v-if="!loaded" :width="120" :height="32" />
        <template v-else><span aria-hidden="true">{{ formatCostAuto(animatedTodayCost) }}</span><span class="sr-only">{{ formatCostAuto(stats?.today_actual_cost ?? 0) }}</span></template>
      </div>
      <span class="focus-caption">{{ t('dashboardFocus.today.cost') }}</span>
      <dl class="focus-pairs">
        <div :data-tip="t('dashboardFocus.today.requestsTip')">
          <dt>{{ t('dashboardFocus.today.requests') }}</dt>
          <dd class="gc-num">{{ formatNumber(stats?.today_requests ?? 0) }}</dd>
        </div>
        <div :data-tip="`${t('dashboardFocus.today.tokensTip')}\n${formatNumber(stats?.today_tokens ?? 0)}`">
          <dt>{{ t('dashboardFocus.today.tokens') }}</dt>
          <dd class="gc-num">{{ formatTokensK(stats?.today_tokens ?? 0) }}</dd>
        </div>
      </dl>
    </section>

    <!-- 服务质量（近 24 小时，本人） -->
    <section class="card gc-stat focus-card">
      <header class="focus-head">
        <span class="focus-icon" :class="[METRIC_TONES.completed.tile, METRIC_TONES.completed.icon]"><Icon name="shield" size="sm" /></span>
        <span class="focus-title">{{ t('dashboardFocus.quality.title') }}</span>
        <span class="focus-badge">{{ t('dashboardFocus.quality.window') }}</span>
      </header>
      <div class="focus-value" :class="successTone" :data-tip="successTip">
        <Skeleton v-if="!loaded" :width="96" :height="32" />
        <template v-else>{{ successRate === null ? '—' : `${successRate.toFixed(successRate >= 99.95 ? 0 : 1)}%` }}</template>
      </div>
      <span class="focus-caption">{{ successRate === null ? t('dashboardFocus.quality.noRequests') : t('dashboardFocus.quality.successRate') }}</span>
      <dl class="focus-pairs">
        <div :data-tip="t('dashboardFocus.quality.ttftTip')">
          <dt>{{ t('dashboardFocus.quality.ttft') }}</dt>
          <dd class="gc-num">{{ ttftText }}</dd>
        </div>
        <div :data-tip="failedTip">
          <dt>{{ t('dashboardFocus.quality.failed') }}</dt>
          <dd class="gc-num">
            <RouterLink v-if="errorViewEnabled && failed > 0 && !supportMode" :to="{ path: '/usage', query: { view: 'errors' } }" class="focus-link">{{ formatNumber(failed) }}<Icon name="chevronRight" size="xs" class="row-arrow" /></RouterLink>
            <template v-else>{{ formatNumber(failed) }}</template>
          </dd>
        </div>
      </dl>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import { useUserView } from '@/composables/useUserView'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { useAppStore } from '@/stores/app'
import { adminSupportContext } from '@/utils/adminSupportContext'
import { exactCredit } from '@/utils/consoleTips'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import { formatCostAuto, formatUsd } from './format'
import { COUNT_UP_MS } from './motion'
import { METRIC_TONES } from './tones'
import { useCountUp } from './useCountUp'
import type { UserSubscription } from '@/types'

// 统计接口每分钟刷新一次（RPM/TPM 是近 5 分钟均值），页面隐藏时暂停，回来立即刷新。
const POLL_INTERVAL_MS = 60 * 1000
// 余额卡按近 7 天的实际扣费算日均消费。
const RUNWAY_DAYS = 7

const { t } = useI18n()
const view = useUserView()
const appStore = useAppStore()
const subscriptionStore = useSubscriptionStore()

const stats = ref<UserDashboardStats | null>(null)
const loaded = ref(false)
let timer: ReturnType<typeof setInterval> | undefined

const supportMode = computed(() => adminSupportContext.value !== null)
const isCustomer = computed(() => !!view.user?.reseller_customer)
const paymentEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.payment))
const errorViewEnabled = computed(() => appStore.cachedPublicSettings?.allow_user_view_error_requests ?? false)

const balance = computed(() => Number(view.user?.balance ?? 0))
const frozen = computed(() => Number(view.user?.frozen_balance ?? 0))
const animatedBalance = useCountUp(balance, COUNT_UP_MS)
const animatedTodayCost = useCountUp(() => stats.value?.today_actual_cost ?? 0, COUNT_UP_MS)
const money = (value: number) => formatUsd(value, 2)
const balanceTip = computed(() => t('consoleTips.header.balance', {
  available: exactCredit(balance.value),
  frozen: exactCredit(frozen.value),
  total: exactCredit(balance.value + frozen.value),
}))

const weekCost = computed(() => stats.value?.last7d_actual_cost ?? 0)
const dailyAverage = computed(() => weekCost.value / RUNWAY_DAYS)
const runwayDays = computed(() => Math.floor(balance.value / dailyAverage.value))
const runwayText = computed(() => (runwayDays.value < 1 ? t('dashboardFocus.balance.runwayShort') : t('dashboardFocus.balance.runway', { days: formatNumber(runwayDays.value) })))
const runwayTone = computed(() => (runwayDays.value < 3 ? 'focus-bad' : runwayDays.value < 7 ? 'focus-warn' : 'focus-good'))

// 生效中的订阅：名称、到期和最近一个有上限的窗口还剩多少。
function remainingOf(subscription: UserSubscription): string {
  const group = subscription.group
  if (group?.daily_limit_usd != null) return t('dashboardFocus.balance.remainingDaily', { amount: money(Math.max(0, group.daily_limit_usd - subscription.daily_usage_usd)) })
  if (group?.weekly_limit_usd != null) return t('dashboardFocus.balance.remainingWeekly', { amount: money(Math.max(0, group.weekly_limit_usd - subscription.weekly_usage_usd)) })
  if (group?.monthly_limit_usd != null) return t('dashboardFocus.balance.remainingMonthly', { amount: money(Math.max(0, group.monthly_limit_usd - subscription.monthly_usage_usd)) })
  return ''
}
const subscriptionRows = computed(() => subscriptionStore.activeSubscriptions.slice(0, 2).map((subscription) => {
  const name = subscription.group?.name ?? `#${subscription.group_id}`
  const days = subscription.expires_at ? Math.max(0, Math.ceil((new Date(subscription.expires_at).getTime() - Date.now()) / 86400000)) : null
  const expires = days === null ? t('dashboardFocus.balance.noExpiry') : t('dashboardFocus.balance.daysLeft', { days })
  return {
    id: subscription.id,
    name,
    expires,
    remaining: remainingOf(subscription),
    tip: subscription.expires_at ? t('dashboardFocus.balance.subscriptionTip', { name, time: new Date(subscription.expires_at).toLocaleString() }) : name,
  }
}))

const success = computed(() => stats.value?.last24h_success ?? 0)
const platformFailed = computed(() => stats.value?.last24h_platform_failed ?? 0)
const failed = computed(() => stats.value?.last24h_failed ?? 0)
const successRate = computed(() => (success.value + platformFailed.value > 0 ? (success.value / (success.value + platformFailed.value)) * 100 : null))
const successTone = computed(() => (successRate.value === null ? '' : successRate.value >= 99 ? 'focus-good' : successRate.value >= 95 ? 'focus-warn' : 'focus-bad'))
const successTip = computed(() => t('dashboardFocus.quality.successTip', { success: formatNumber(success.value), failed: formatNumber(platformFailed.value) }))
const failedTip = computed(() => t(errorViewEnabled.value ? 'dashboardFocus.quality.failedTipLink' : 'dashboardFocus.quality.failedTip', { platform: formatNumber(platformFailed.value) }))
const ttftText = computed(() => {
  const ms = stats.value?.last24h_ttft_median_ms
  if (ms == null) return '—'
  return ms < 1000 ? `${Math.round(ms)}ms` : `${(ms / 1000).toFixed(2)}s`
})
const formatRpm = (value: number) => formatNumber(Math.round(value * 10) / 10)

const load = async () => {
  stats.value = await usageAPI.getDashboardStats().finally(() => {
    loaded.value = true
  })
}

const startPolling = () => {
  clearInterval(timer)
  timer = setInterval(() => void load(), POLL_INTERVAL_MS)
}

const onVisibilityChange = () => {
  if (document.visibilityState === 'hidden') {
    clearInterval(timer)
    return
  }
  void load()
  startPolling()
}

onMounted(() => {
  void load()
  startPolling()
  document.addEventListener('visibilitychange', onVisibilityChange)
  if (isFeatureFlagEnabled(FeatureFlags.subscription)) void subscriptionStore.fetchActiveSubscriptions()
})

onBeforeUnmount(() => {
  clearInterval(timer)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})

defineExpose({ reload: () => Promise.all([load(), isFeatureFlagEnabled(FeatureFlags.subscription) ? subscriptionStore.fetchActiveSubscriptions(true) : undefined]) })
</script>

<style scoped>
.focus-grid { display: grid; grid-template-columns: minmax(0, 1.4fr) repeat(2, minmax(0, 1fr)); gap: 16px; }
.focus-card { position: relative; display: flex; min-width: 0; flex-direction: column; gap: 6px; padding: 18px 20px; overflow: hidden; border-radius: var(--surface-radius); }
.focus-balance::before { content: ''; position: absolute; inset: 0; pointer-events: none; border-radius: inherit; background: radial-gradient(24rem 11rem at 0% 0%, rgb(99 102 241 / 14%), transparent 70%), radial-gradient(16rem 9rem at 100% 100%, rgb(245 158 11 / 10%), transparent 70%); }
.focus-card > * { position: relative; }
.focus-head { display: flex; min-width: 0; align-items: center; gap: 8px; margin-bottom: 4px; }
.focus-icon { display: inline-flex; flex: none; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: var(--surface-radius-control); }
.focus-title { min-width: 0; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 13px; font-weight: 600; color: var(--surface-text); }
.focus-badge { flex: none; border-radius: var(--surface-radius-tag); padding: 1px 7px; font-size: 11px; color: var(--surface-muted); background: var(--surface-soft); }
.focus-live { display: inline-flex; flex: none; align-items: center; gap: 6px; font-size: 11px; font-variant-numeric: tabular-nums; color: var(--surface-muted); }
.focus-live-dot { width: 6px; height: 6px; border-radius: 9999px; background: theme('colors.emerald.500'); box-shadow: 0 0 0 3px rgb(16 185 129 / 18%); }
.focus-hero { font-size: 38px; line-height: 1.1; font-weight: 700; letter-spacing: -.025em; font-variant-numeric: tabular-nums; color: var(--surface-text); }
.focus-value { min-height: 34px; font-size: 28px; line-height: 1.2; font-weight: 700; letter-spacing: -.02em; font-variant-numeric: tabular-nums; color: var(--surface-text); }
.focus-caption { font-size: 12px; color: var(--surface-muted); }
.focus-line { font-size: 13px; color: var(--surface-muted); }
.focus-line strong { font-weight: 600; }
.focus-sep { margin-inline: 6px; opacity: .5; }
.focus-subs { display: flex; flex-direction: column; gap: 4px; margin-top: 2px; }
.focus-sub { display: flex; min-width: 0; align-items: center; gap: 6px; font-size: 12px; color: var(--surface-muted); }
.focus-sub .font-medium { color: var(--surface-text); }
.focus-actions { display: flex; flex-wrap: wrap; gap: 8px; margin-top: auto; padding-top: 10px; }
.focus-pairs { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin-top: auto; padding-top: 12px; border-top: 1px solid var(--surface-edge); }
.focus-pairs dt { font-size: 11px; color: var(--surface-muted); }
.focus-pairs dd { margin-top: 2px; font-size: 15px; font-weight: 600; color: var(--surface-text); }
.focus-link { display: inline-flex; align-items: center; gap: 2px; color: theme('colors.red.600'); }
.dark .focus-link { color: theme('colors.red.400'); }
.focus-link:hover { text-decoration: underline; }
.focus-good { color: theme('colors.emerald.600'); }
.focus-warn { color: theme('colors.amber.600'); }
.focus-bad { color: theme('colors.red.600'); }
.dark .focus-good { color: theme('colors.emerald.400'); }
.dark .focus-warn { color: theme('colors.amber.400'); }
.dark .focus-bad { color: theme('colors.red.400'); }
@media (max-width: 1279px) {
  .focus-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .focus-balance { grid-column: span 2; }
}
@media (max-width: 640px) {
  .focus-grid { grid-template-columns: minmax(0, 1fr); }
  .focus-balance { grid-column: auto; }
  .focus-hero { font-size: 32px; }
}
</style>

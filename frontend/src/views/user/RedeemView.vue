<template>
  <AppLayout>
    <!-- 宽屏左栏放余额、兑换与说明，右栏放兑换记录；窄屏按兑换、说明、记录顺序堆叠。 -->
    <div class="grid w-full gap-6 lg:grid-cols-[22.5rem_minmax(0,1fr)] lg:items-start">
      <div class="flex min-w-0 flex-col gap-6">
        <section class="card p-4 sm:p-6">
          <div class="flex flex-col gap-4">
            <RedeemCelebration
              :sequence="celebration?.sequence ?? 0"
              :title="celebration?.title ?? ''"
              :detail="celebration?.detail ?? ''"
            >
              <dl class="grid grid-cols-2 gap-4">
                <div>
                  <dt class="text-xs font-medium text-gray-500 dark:text-dark-400">
                    {{ t('redeem.currentBalance') }}
                  </dt>
                  <dd class="mt-1 text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">
                    ${{ user?.balance?.toFixed(2) || '0.00' }}
                  </dd>
                </div>
                <div>
                  <dt class="text-xs font-medium text-gray-500 dark:text-dark-400">
                    {{ t('redeem.concurrency') }}
                  </dt>
                  <dd class="mt-1 text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">
                    {{ user?.concurrency || 0 }}
                    <span class="text-sm font-normal text-gray-500 dark:text-dark-400">{{ t('redeem.requests') }}</span>
                  </dd>
                </div>
              </dl>
            </RedeemCelebration>

            <!-- 中等宽度下输入框与按钮同排，宽屏左栏较窄时恢复纵向排列。 -->
            <form
              v-support-readonly
              class="flex flex-col gap-3 border-t border-gray-100 pt-4 dark:border-dark-700 sm:flex-row lg:flex-col"
              @submit.prevent="handleRedeem"
            >
              <label for="code" class="sr-only">{{ t('redeem.redeemCodeLabel') }}</label>
              <div class="relative min-w-0 flex-1">
                <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                  <Icon name="gift" size="md" class="text-gray-400 dark:text-dark-500" />
                </div>
                <input
                  id="code"
                  v-model="redeemCode"
                  type="text"
                  required
                  autocomplete="off"
                  spellcheck="false"
                  :placeholder="t('redeem.redeemCodePlaceholder')"
                  :disabled="submitting"
                  class="input pl-10"
                />
              </div>
              <button
                type="submit"
                :disabled="!redeemCode || submitting"
                class="btn btn-primary w-full sm:w-auto sm:px-6 lg:w-full"
              >
                <svg v-if="submitting" class="-ml-1 mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path
                    class="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  ></path>
                </svg>
                {{ submitting ? t('redeem.redeeming') : t('redeem.redeemButton') }}
              </button>
            </form>
            <p class="-mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('redeem.redeemCodeHint') }}</p>
            <p
              v-if="errorMessage"
              role="alert"
              class="flex items-start gap-2 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"
            >
              <Icon name="exclamationCircle" size="sm" class="mt-0.5 shrink-0" />
              <span class="min-w-0">
                <span class="font-medium">{{ t('redeem.redeemFailed') }}</span>
                {{ errorMessage }}
              </span>
            </p>
          </div>
        </section>

        <section class="card p-4 sm:p-6">
          <h2 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
            <Icon name="infoCircle" size="sm" class="text-primary-600 dark:text-primary-400" />
            {{ t('redeem.aboutCodes') }}
          </h2>
          <ul class="mt-3 list-inside list-disc space-y-1.5 text-sm text-gray-600 dark:text-dark-300">
            <li>{{ t('redeem.codeRule1') }}</li>
            <li>{{ t('redeem.codeRule2') }}</li>
            <li>
              {{ t('redeem.codeRule3') }}
              <span
                v-if="contactInfo"
                class="ml-1.5 inline-flex items-center rounded-md bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
              >
                {{ contactInfo }}
              </span>
            </li>
            <li>{{ t('redeem.codeRule4') }}</li>
          </ul>
        </section>
      </div>

      <!-- 兑换记录用分隔线划分列表项，窄屏共用一层内边距。 -->
      <section class="card flex min-w-0 flex-col overflow-hidden">
        <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:px-6 sm:py-4">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">
            {{ t('redeem.recentActivity') }}
          </h2>
        </div>

        <!-- 首次加载显示转圈，翻页时当前列表降低透明度并保持占位。 -->
        <div v-if="loadingHistory && history.length === 0" class="flex items-center justify-center py-12">
          <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
        </div>

        <ul
          v-else-if="history.length > 0"
          :class="['divide-y divide-gray-100 transition-opacity dark:divide-dark-700', loadingHistory ? 'opacity-60' : '']"
        >
          <li v-for="item in history" :key="item.id" class="flex items-center gap-3 px-4 py-3 sm:gap-4 sm:px-6">
            <div :class="['flex h-9 w-9 shrink-0 items-center justify-center rounded-lg', getHistoryTone(item).bg]">
              <Icon v-if="isBalanceType(item.type)" name="dollar" size="sm" :class="getHistoryTone(item).text" />
              <Icon v-else-if="isSubscriptionType(item.type)" name="badge" size="sm" :class="getHistoryTone(item).text" />
              <Icon v-else name="bolt" size="sm" :class="getHistoryTone(item).text" />
            </div>

            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-900 dark:text-white">
                {{ getHistoryItemTitle(item) }}
              </p>
              <p class="mt-0.5 flex min-w-0 items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400">
                <span class="shrink-0">{{ formatDateTime(item.used_at) }}</span>
                <span aria-hidden="true">·</span>
                <template v-if="isAdminAdjustment(item.type)">
                  <span class="shrink-0">{{ t('redeem.adminAdjustment') }}</span>
                  <template v-if="item.notes">
                    <span aria-hidden="true">·</span>
                    <span class="truncate italic" :title="item.notes">{{ item.notes }}</span>
                  </template>
                </template>
                <span v-else class="truncate font-mono">{{ item.code.slice(0, 8) }}...</span>
              </p>
            </div>

            <p
              :class="['max-w-40 shrink-0 truncate text-right text-sm font-semibold tabular-nums', getHistoryTone(item).text]"
            >
              {{ formatHistoryValue(item) }}
            </p>
          </li>
        </ul>

        <div v-else class="empty-state py-12">
          <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-gray-100 dark:bg-dark-800">
            <Icon name="clock" size="lg" class="text-gray-400 dark:text-dark-500" />
          </div>
          <p class="text-sm text-gray-500 dark:text-dark-400">
            {{ t('redeem.historyWillAppear') }}
          </p>
        </div>

        <div
          class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:text-dark-300 sm:px-6"
        >
          <span>{{ t('common.total') }}: {{ historyTotal }} {{ t('pagination.results') }}</span>
          <div class="flex flex-wrap items-center gap-3">
            <label class="flex items-center gap-2">
              {{ t('pagination.perPage') }}
              <select
                v-model="historyPageSize"
                class="input w-20"
                :disabled="loadingHistory || submitting"
                @change="fetchHistory(1)"
              >
                <option v-for="size in [20, 50, 100]" :key="size" :value="size">{{ size }}</option>
              </select>
            </label>
            <button
              class="btn btn-secondary"
              :disabled="loadingHistory || submitting || historyPage <= 1"
              @click="fetchHistory(historyPage - 1)"
            >{{ t('pagination.previous') }}</button>
            <button
              class="btn btn-secondary"
              :disabled="loadingHistory || submitting || historyPage * historyPageSize >= historyTotal"
              @click="fetchHistory(historyPage + 1)"
            >{{ t('pagination.next') }}</button>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { supportReadonly as vSupportReadonly } from '@/directives/supportReadonly'
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserView as useAuthStore } from '@/composables/useUserView'
import { useAppStore } from '@/stores/app'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { redeemAPI, authAPI, type RedeemHistoryItem } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import RedeemCelebration from '@/components/user/RedeemCelebration.vue'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const subscriptionStore = useSubscriptionStore()

const user = computed(() => authStore.user)

const redeemCode = ref('')
const submitting = ref(false)
const celebration = ref<{ sequence: number; title: string; detail: string } | null>(null)
let celebrationSequence = 0
const errorMessage = ref('')

// History data
const history = ref<RedeemHistoryItem[]>([])
const loadingHistory = ref(false)
const historyPage = ref(1)
const historyPageSize = ref(20)
const historyTotal = ref(0)
let historyRequest = 0
let loadedHistoryPageSize = 20
const contactInfo = ref('')

// Helper functions for history display
const isBalanceType = (type: string) => {
  return type === 'balance' || type === 'admin_balance' || type === 'team_transfer' || type === 'canvas_transfer' || type === 'canvas_reversal'
}

const isSubscriptionType = (type: string) => {
  return type === 'subscription'
}

const isAdminAdjustment = (type: string) => {
  return type === 'admin_balance' || type === 'admin_concurrency'
}

const getHistoryTone = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    return item.value >= 0
      ? { bg: 'bg-emerald-100 dark:bg-emerald-900/30', text: 'text-emerald-600 dark:text-emerald-400' }
      : { bg: 'bg-red-100 dark:bg-red-900/30', text: 'text-red-600 dark:text-red-400' }
  }
  if (isSubscriptionType(item.type)) {
    return { bg: 'bg-purple-100 dark:bg-purple-900/30', text: 'text-purple-600 dark:text-purple-400' }
  }
  return item.value >= 0
    ? { bg: 'bg-blue-100 dark:bg-blue-900/30', text: 'text-blue-600 dark:text-blue-400' }
    : { bg: 'bg-orange-100 dark:bg-orange-900/30', text: 'text-orange-600 dark:text-orange-400' }
}

// 庆祝内容取自兑换响应，不依赖随后的余额刷新。
const getCelebrationDetail = (result: RedeemHistoryItem) => {
  if (result.type === 'balance') {
    return `${t('redeem.added')} $${result.value.toFixed(2)}`
  }
  if (result.type === 'concurrency') {
    return `${t('redeem.added')} ${result.value} ${t('redeem.concurrentRequests')}`
  }
  const days = t('redeem.subscriptionDays', { days: result.validity_days || Math.round(result.value) })
  return [t('redeem.subscriptionAssigned'), result.group?.name, days].filter(Boolean).join(' · ')
}

const getHistoryItemTitle = (item: RedeemHistoryItem) => {
  if (item.type === 'balance') {
    return t('redeem.balanceAddedRedeem')
  } else if (item.type === 'admin_balance') {
    return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
  } else if (item.type === 'team_transfer') {
    return t('redeem.balanceTeamTransfer')
  } else if (item.type === 'canvas_transfer') {
    return t('redeem.balanceCanvasTransfer')
  } else if (item.type === 'canvas_reversal') {
    return t('redeem.balanceCanvasTransferReversal')
  } else if (item.type === 'concurrency') {
    return t('redeem.concurrencyAddedRedeem')
  } else if (item.type === 'admin_concurrency') {
    return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
  } else if (item.type === 'subscription') {
    return t('redeem.subscriptionAssigned')
  }
  return t('common.unknown')
}

const formatHistoryValue = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}$${item.value.toFixed(2)}`
  } else if (isSubscriptionType(item.type)) {
    // 订阅类型显示有效天数和分组名称
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}${t('redeem.days')} - ${groupName}` : `${days}${t('redeem.days')}`
  } else {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}${item.value} ${t('redeem.requests')}`
  }
}

const fetchHistory = async (page = 1) => {
  const request = ++historyRequest
  const pageSize = historyPageSize.value
  loadingHistory.value = true
  try {
    const result = await redeemAPI.getHistory(page, pageSize)
    if (request !== historyRequest) return
    history.value = result.items
    historyTotal.value = result.total
    historyPage.value = page
    historyPageSize.value = pageSize
    loadedHistoryPageSize = pageSize
  } catch (error) {
    if (request !== historyRequest) return
    historyPageSize.value = loadedHistoryPageSize
    appStore.showError(t('redeem.historyLoadFailed'))
    console.error('Failed to fetch history:', error)
  } finally {
    if (request === historyRequest) loadingHistory.value = false
  }
}

const handleRedeem = async () => {
  if (!redeemCode.value.trim()) {
    appStore.showError(t('redeem.pleaseEnterCode'))
    return
  }

  submitting.value = true
  errorMessage.value = ''
  celebration.value = null

  try {
    const result = await redeemAPI.redeem(redeemCode.value.trim())

    celebration.value = {
      sequence: ++celebrationSequence,
      title: t('redeem.redeemSuccess'),
      detail: getCelebrationDetail(result)
    }
    redeemCode.value = ''

    // Refresh user data to get updated balance/concurrency
    try {
      await authStore.refreshUser()
    } catch (error) {
      console.error('Failed to refresh user after redeem:', error)
      appStore.showWarning(t('redeem.userRefreshFailed'))
    }

    // If subscription type, immediately refresh subscription status
    if (result.type === 'subscription') {
      try {
        await subscriptionStore.fetchActiveSubscriptions(true) // force refresh
      } catch (error) {
        console.error('Failed to refresh subscriptions after redeem:', error)
        appStore.showWarning(t('redeem.subscriptionRefreshFailed'))
      }
    }

    // Refresh history
    await fetchHistory()

    // Show success toast
    appStore.showSuccess(t('redeem.codeRedeemSuccess'))
  } catch (error: any) {
    errorMessage.value = error.response?.data?.detail || t('redeem.failedToRedeem')

    appStore.showError(t('redeem.redeemFailed'))
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  fetchHistory()
  try {
    const settings = await authAPI.getPublicSettings()
    contactInfo.value = settings.contact_info || ''
  } catch (error) {
    console.error('Failed to load contact info:', error)
  }
})
</script>

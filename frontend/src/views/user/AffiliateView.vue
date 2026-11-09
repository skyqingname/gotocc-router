<template>
  <AppLayout appearance="user">
    <div class="affiliate-workspace">
      <div v-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else-if="loadError" class="card p-6">
        <p role="alert" class="text-sm text-red-600">{{ loadError }}</p>
        <button class="btn btn-secondary mt-4" @click="loadAffiliateDetail()">{{ t('common.retry') }}</button>
      </div>
      <template v-else-if="detail">
        <!-- LC-024：未成为代理时只展示申请入口，返利界面整体不出现 -->
        <div v-if="!isAgent" class="card p-6">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div class="min-w-0">
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ agentTitle }}
              </h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ agentDescription }}</p>
              <p v-if="agentAppliedAt" class="mt-2 text-xs text-gray-400 dark:text-dark-500">
                {{ t('affiliate.agent.appliedAt', { time: agentAppliedAt }) }}
              </p>
            </div>
            <button v-support-readonly
              v-if="canApplyForAgent"
              class="btn btn-primary w-full sm:w-auto sm:shrink-0"
              :disabled="applyingAgent"
              @click="applyForAgent"
            >
              <Icon v-if="applyingAgent" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="userPlus" size="sm" />
              <span>{{ applyingAgent ? t('affiliate.agent.applying') : agentActionLabel }}</span>
            </button>

          </div>
        </div>


        <div v-if="isAgent" class="affiliate-stats grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <p class="flex items-center gap-1.5 text-sm surface-muted">
              <Icon name="dollar" size="sm" class="text-primary-500" />
              {{ t('affiliate.stats.rebateRate') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">
              {{ formattedRebateRates }}
            </p>
            <p class="mt-1 text-xs surface-muted">{{ t('affiliate.stats.rebateRateHint') }}</p>
          </div>
          <div class="card p-5">
            <p class="text-sm surface-muted">{{ t('affiliate.stats.invitedUsers') }}</p>
            <p class="mt-2 text-2xl font-semibold">{{ formatCount(detail.aff_count) }}</p>
          </div>
          <div class="card p-5">
            <p class="text-sm surface-muted">{{ t('affiliate.stats.availableQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">{{ formatCurrency(detail.aff_quota) }}</p>
          </div>
          <div class="card p-5">
            <p class="text-sm surface-muted">{{ t('affiliate.stats.totalQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold">{{ formatCurrency(detail.aff_history_quota) }}</p>
            <p v-if="detail.aff_frozen_quota > 0" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {{ t('affiliate.stats.frozenQuota') }}: {{ formatCurrency(detail.aff_frozen_quota) }}
            </p>
          </div>
        </div>

        <div class="affiliate-top-row">
          <section class="card affiliate-invitation">
            <h3 class="text-base font-semibold">{{ t('affiliate.title') }}</h3>
            <p class="mt-1 text-sm surface-muted">{{ t('affiliate.description') }}</p>
            <div class="mt-4 grid gap-4 md:grid-cols-2">
              <div class="min-w-0 space-y-2">
                <p class="text-sm font-medium surface-muted">{{ t('affiliate.yourCode') }}</p>
                <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 surface-inset px-3 py-2 dark:border-dark-700 sm:flex-row sm:items-center">
                  <code class="min-w-0 break-all text-xs font-semibold sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                  <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyCode">
                    <Icon name="copy" size="sm" /><span>{{ t('affiliate.copyCode') }}</span>
                  </button>
                </div>
              </div>
              <div class="min-w-0 space-y-2">
                <p class="text-sm font-medium surface-muted">{{ t('affiliate.inviteLink') }}</p>
                <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 surface-inset px-3 py-2 dark:border-dark-700 sm:flex-row sm:items-center">
                  <code class="min-w-0 break-all text-xs sm:flex-1 sm:truncate surface-muted">{{ inviteLink }}</code>
                  <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                    <Icon name="copy" size="sm" /><span>{{ t('affiliate.copyLink') }}</span>
                  </button>
                </div>
              </div>
            </div>
          </section>

          <section v-if="isAgent" class="card affiliate-transfer">
            <div class="flex flex-wrap items-center justify-between gap-5">
              <div class="min-w-0 flex-1">
                <h3 class="text-base font-semibold">{{ t('affiliate.transfer.title') }}</h3>
                <p class="mt-1 text-sm surface-muted">{{ t('affiliate.transfer.description') }}</p>
                <div class="affiliate-transfer-value">
                  <strong>{{ formatCurrency(detail.aff_quota) }}</strong>
                  <span>{{ t('affiliate.stats.availableQuota') }}</span>
                </div>
              </div>
              <button v-support-readonly class="btn btn-primary" :disabled="transferring || detail.aff_quota <= 0" @click="transferQuota">
                <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
                <Icon v-else name="dollar" size="sm" />
                <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
              </button>
            </div>
            <p v-if="detail.aff_quota <= 0" class="mt-3 text-sm text-amber-600 dark:text-amber-400">{{ t('affiliate.transfer.empty') }}</p>
          </section>
        </div>

        <section class="affiliate-guide" :aria-label="t('affiliate.tips.title')">
          <h3 class="mb-3 text-sm font-semibold">{{ t('affiliate.tips.title') }}</h3>
          <ul class="affiliate-guide-grid">
            <li v-for="(tip, index) in visibleAffiliateTips" :key="tip.body" class="card affiliate-guide-card">
              <div class="affiliate-guide-top">
                <span class="affiliate-guide-icon"><Icon :name="tip.icon" size="md" /></span>
                <span class="affiliate-guide-step">{{ String(index + 1).padStart(2, '0') }}</span>
              </div>
              <h4 class="text-sm font-semibold">{{ t(tip.title) }}</h4>
              <p class="text-[13px] leading-relaxed surface-muted">{{ t(tip.body, { rates: formattedRebateRates }) }}</p>
            </li>
          </ul>
        </section>

        <section v-if="isAgent" ref="inviteeSection" class="card affiliate-invitees">
          <header class="affiliate-invitees-heading">
            <h3 class="text-[15px] font-semibold">{{ t('affiliate.invitees.title') }}</h3>
            <span class="text-xs surface-muted">{{ t('common.total') }}: {{ detail.invitees.length }}</span>
          </header>
          <div v-if="detail.invitees.length === 0" class="p-6 text-center text-sm surface-muted">{{ t('affiliate.invitees.empty') }}</div>
          <div v-else class="record-table-scroll affiliate-invitees-viewport">
            <table class="record-table w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 dark:border-dark-700 surface-muted">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.level') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-3 py-2 font-medium text-right">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in pagedInvitees" :key="item.user_id" class="border-b border-gray-100 last:border-b-0 dark:border-dark-800">
                  <td class="px-3 py-3">{{ t('affiliate.invitees.generation', { level: item.level }) }}</td>
                  <td class="px-3 py-3">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3 surface-muted">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3 surface-muted">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <nav v-if="isAgent" class="card affiliate-pagination" :aria-label="t('affiliate.invitees.title')">
          <div class="affiliate-pagination-info">
            <span>{{ t('common.total') }}: {{ detail.invitees.length }} {{ t('pagination.results') }}</span>
            <span>{{ t('pagination.perPage') }}: {{ inviteePageSize }}</span>
          </div>
          <div class="affiliate-pagination-actions">
            <button class="btn btn-secondary" :disabled="inviteePage <= 1" @click="changeInviteePage(inviteePage - 1)">{{ t('pagination.previous') }}</button>
            <template v-for="(page, index) in visibleInviteePages" :key="`${page}-${index}`">
              <span v-if="page === 'ellipsis'" class="px-1 text-sm surface-muted" aria-hidden="true">…</span>
              <button v-else type="button" class="affiliate-page-number surface-control" :aria-label="t('pagination.goToPage', { page })" :aria-current="page === inviteePage ? 'page' : undefined" @click="changeInviteePage(page)">{{ page }}</button>
            </template>
            <button class="btn btn-secondary" :disabled="inviteePage >= inviteePageCount" @click="changeInviteePage(inviteePage + 1)">{{ t('pagination.next') }}</button>
          </div>
        </nav>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { supportReadonly as vSupportReadonly } from '@/directives/supportReadonly'
import { computed, onMounted, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { AgentProfile, UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useUserView as useAuthStore } from '@/composables/useUserView'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)
const loadError = ref('')

// LC-024 enrollment state. The rebate surface is gated on an approved identity;
// invite code and share link stay available so recruiting never depends on it.
const agentProfile = ref<AgentProfile | null>(null)
const applyingAgent = ref(false)
const isAgent = computed(() => agentProfile.value?.status === 'approved')
const canApplyForAgent = computed(() => !isAgent.value)
const agentAppliedAt = computed(() => agentProfile.value?.applied_at ? formatDateTime(agentProfile.value.applied_at) : '')
const agentTitle = computed(() => t('affiliate.agent.applyTitle'))
const agentDescription = computed(() => t('affiliate.agent.applyDescription'))
const agentActionLabel = computed(() => t('affiliate.agent.applyButton'))

async function submitAgentApplication(): Promise<void> {
  if (applyingAgent.value || !canApplyForAgent.value) return
  applyingAgent.value = true
  try {
    agentProfile.value = await userAPI.applyForAgent()
    await loadAffiliateDetail(true)
    appStore.showSuccess(t('affiliate.agent.applied'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.agent.applyFailed')))
  } finally {
    applyingAgent.value = false
  }
}

function applyForAgent(): void {
  void submitAgentApplication()
}
const inviteePageSize = 10
const inviteePage = ref(1)
const inviteeSection = ref<HTMLElement | null>(null)
const inviteePageCount = computed(() => Math.max(1, Math.ceil((detail.value?.invitees.length || 0) / inviteePageSize)))
const pagedInvitees = computed(() => (detail.value?.invitees || []).slice((inviteePage.value - 1) * inviteePageSize, inviteePage.value * inviteePageSize))
const visibleInviteePages = computed(() => {
  const total = inviteePageCount.value
  const candidates = new Set([1, total, inviteePage.value - 1, inviteePage.value, inviteePage.value + 1])
  if (total <= 7) for (let page = 1; page <= total; page++) candidates.add(page)
  const pages = [...candidates].filter(page => page >= 1 && page <= total).sort((a, b) => a - b)
  const result: Array<number | 'ellipsis'> = []
  for (const page of pages) {
    const previous = result[result.length - 1]
    if (typeof previous === 'number' && page - previous > 1) result.push('ellipsis')
    result.push(page)
  }
  return result
})
const affiliateTips = [
  { title: 'affiliate.tips.shareTitle', body: 'affiliate.tips.line1', icon: 'link' },
  { title: 'affiliate.tips.earnTitle', body: 'affiliate.tips.line2', icon: 'dollar' },
  { title: 'affiliate.tips.transferTitle', body: 'affiliate.tips.line3', icon: 'arrowRight' },
  { title: 'affiliate.tips.frozenTitle', body: 'affiliate.tips.line4', icon: 'lock' },
] as const

const visibleAffiliateTips = computed(() => affiliateTips.filter(tip =>
  (tip.body !== 'affiliate.tips.line2' || isAgent.value) &&
  (tip.body !== 'affiliate.tips.line4' || (detail.value?.aff_frozen_quota || 0) > 0)
))

async function changeInviteePage(page: number): Promise<void> {
  inviteePage.value = Math.max(1, Math.min(inviteePageCount.value, page))
  await nextTick()
  inviteeSection.value?.scrollIntoView({ block: 'start' })
}
watch(inviteePageCount, count => {
  if (inviteePage.value > count) void changeInviteePage(count)
})


const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?invitation_code=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?invitation_code=${encodeURIComponent(detail.value.aff_code)}`
})

const formattedRebateRates = computed(() =>
  detail.value?.rebate_rates_percent.map(rate => `${Number(rate.toFixed(2))}%`).join(' / ') ?? ''
)

function formatCount(value: number): string {
  return value.toLocaleString()
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }
  loadError.value = ''
  try {
    const [affiliate, agent] = await Promise.all([
      userAPI.getAffiliateDetail(),
      userAPI.getAgentProfile(),
    ])
    detail.value = affiliate
    agentProfile.value = agent
  } catch (error) {
    loadError.value = extractApiErrorMessage(error, t('affiliate.loadFailed'))
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>

<style scoped>
.affiliate-workspace { display: flex; flex-direction: column; gap: var(--shell-zone-gap); min-width: 0; }
.affiliate-top-row { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; align-items: stretch; }
.affiliate-top-row > .card { min-width: 0; min-height: 190px; padding: 20px 24px; border-radius: 16px; }
.affiliate-transfer { display: flex; flex-direction: column; justify-content: center; }
.affiliate-transfer-value { display: flex; flex-wrap: wrap; align-items: baseline; gap: 12px; margin-top: 14px; }
.affiliate-transfer-value strong { font-size: 30px; line-height: 1.3; font-weight: 600; letter-spacing: -.03em; font-variant-numeric: tabular-nums; }
.affiliate-transfer-value span { color: var(--surface-muted); font-size: 12px; }
.affiliate-guide-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 16px; }
.affiliate-guide-card { display: flex; flex-direction: column; gap: 8px; min-width: 0; padding: 16px 20px; border-radius: 16px; }
.affiliate-guide-top { display: flex; align-items: center; justify-content: space-between; }
.affiliate-guide-icon { display: inline-flex; align-items: center; justify-content: center; width: 38px; height: 38px; border-radius: 12px; background: var(--surface-soft); color: var(--console-accent-readable); }
.affiliate-guide-step { color: var(--surface-muted); opacity: .65; font: 500 12px/1.5 ui-monospace, monospace; }
.affiliate-invitees { border-radius: var(--surface-radius); overflow: hidden; }
.affiliate-invitees-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 14px 20px; }
.affiliate-invitees-viewport { padding: 0 16px 12px; }
.affiliate-pagination { display: flex; align-items: center; justify-content: space-between; gap: 16px; min-width: 0; padding: 14px 20px; }
.affiliate-pagination-info, .affiliate-pagination-actions { display: flex; align-items: center; gap: 16px; min-width: 0; }
.affiliate-pagination-info { color: var(--surface-muted); font-size: 13px; }
.affiliate-pagination-actions { gap: 8px; margin-left: auto; }
.affiliate-pagination-actions > .btn { min-width: 80px; white-space: nowrap; }
.affiliate-page-number { min-width: 34px; height: 34px; padding: 0 9px; border: 1px solid transparent; border-radius: 9px; color: var(--surface-muted); background: transparent; font-size: 13px; }
.affiliate-page-number[aria-current="page"] { border-color: var(--surface-edge); background: var(--surface-soft); color: var(--surface-text); font-weight: 600; }
@media (max-width: 1023px) {
  .affiliate-top-row { grid-template-columns: 1fr; }
  .affiliate-guide-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 639px) {
  .affiliate-guide-grid { grid-template-columns: 1fr; }
  .affiliate-pagination { flex-wrap: wrap; }
  .affiliate-pagination-info { width: 100%; justify-content: space-between; }
  .affiliate-pagination-actions { width: 100%; justify-content: flex-end; flex-wrap: wrap; }
}
</style>

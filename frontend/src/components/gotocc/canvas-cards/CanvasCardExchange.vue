<template>
  <section class="card mb-6 overflow-hidden" :aria-label="t('canvasCard.title')">
    <div class="flex flex-wrap items-start justify-between gap-4 border-b border-gray-100 p-5 dark:border-dark-700 sm:p-6">
      <div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('canvasCard.title') }}</h2>
        <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">{{ t('canvasCard.subtitle') }} {{ t('canvasCard.hint') }}</p>
      </div>
      <a :href="navigationLinks.canvas_url" target="_blank" rel="noopener noreferrer" class="btn btn-secondary btn-sm">{{ t('canvasCard.openCanvas') }} ↗</a>
    </div>

    <div class="p-5 sm:p-6">
      <p v-if="loading" class="text-sm text-gray-500">{{ t('common.loading') }}</p>
      <div v-else-if="loadError" role="alert" class="flex items-center gap-3 text-sm text-red-600 dark:text-red-400">
        {{ loadError }} <button class="btn btn-secondary btn-sm" @click="load">{{ t('common.retry') }}</button>
      </div>
      <template v-else-if="overview">
        <fieldset :disabled="submitting || pending !== null" class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_19rem]">
          <legend class="sr-only">{{ t('canvasCard.denomination') }}</legend>
          <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <button v-for="item in overview.stock" :key="item.denomination" type="button"
              class="relative flex min-h-28 flex-col items-start justify-center rounded-xl border px-4 py-4 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-50"
              :class="amount === item.denomination ? 'border-primary-500 bg-primary-50 ring-1 ring-primary-500 dark:bg-primary-500/10' : 'border-gray-200 bg-gray-50/60 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800/40'"
              :disabled="item.available === 0" :aria-pressed="amount === item.denomination" @click="amount = item.denomination">
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('canvasCard.denomination') }}</span>
              <span class="mt-1 text-3xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ item.denomination }}</span>
              <span class="mt-2 text-xs" :class="item.available ? 'text-primary-700 dark:text-primary-300' : 'text-gray-400'">{{ item.available ? t('canvasCard.available', { count: item.available }) : t('canvasCard.soldOut') }}</span>
              <span v-if="amount === item.denomination" aria-hidden="true" class="absolute right-3 top-3 text-primary-600">✓</span>
            </button>
          </div>
          <div>
            <p class="mb-2 text-sm font-medium text-gray-900 dark:text-white">{{ t('canvasCard.funding') }}</p>
            <div class="space-y-2">
              <label class="flex cursor-pointer items-center gap-3 rounded-lg border border-gray-200 px-3 py-2.5 dark:border-dark-600">
                <input v-model="funding" type="radio" value="personal" name="canvas-card-funding" class="text-primary-600" />
                <span class="text-sm text-gray-700 dark:text-dark-200">{{ t('canvasCard.personal') }}<small class="mt-0.5 block text-xs text-gray-500">{{ t('canvasCard.balance', { amount: overview.personal_balance.toFixed(2) }) }}</small></span>
              </label>
              <label v-if="overview.team" class="flex cursor-pointer items-center gap-3 rounded-lg border border-gray-200 px-3 py-2.5 dark:border-dark-600" :class="!teamAvailable && 'opacity-50'">
                <input v-model="funding" type="radio" value="team" name="canvas-card-funding" :disabled="!teamAvailable" class="text-primary-600" />
                <span class="min-w-0 text-sm text-gray-700 dark:text-dark-200">{{ t('canvasCard.team') }} · {{ overview.team.name }}<small class="mt-0.5 block text-xs text-gray-500">{{ teamAvailable ? t('canvasCard.balance', { amount: overview.team.balance.toFixed(2) }) : t('canvasCard.teamUnavailable') }}</small></span>
              </label>
            </div>
          </div>
        </fieldset>
        <p v-if="funding === 'team'" class="mt-3 text-xs text-gray-500 dark:text-dark-400">{{ t('canvasCard.teamHint') }}</p>
        <div class="mt-5 flex flex-wrap items-center justify-end gap-3 border-t border-gray-100 pt-4 dark:border-dark-700">
          <p v-if="pending && !submitting" class="max-w-xl text-sm text-amber-700 dark:text-amber-300">{{ t('canvasCard.pending') }}</p>
          <p v-else-if="amount !== null && balance < amount" class="text-sm text-red-600 dark:text-red-400">{{ t('canvasCard.insufficient') }}</p>
          <button class="btn btn-primary w-full sm:w-auto" :disabled="submitting || (!pending && !canExchange)" @click="exchange">
            {{ submitting ? t('canvasCard.exchanging') : pending ? t('canvasCard.retry') : amount === null ? t('canvasCard.choose') : t('canvasCard.exchange', { amount }) }}
          </button>
        </div>
        <p v-if="exchangeError" role="alert" class="mt-3 text-sm text-red-600 dark:text-red-400">{{ exchangeError }}</p>
      </template>
    </div>

    <details class="border-t border-gray-100 dark:border-dark-700" @toggle="onHistoryToggle">
      <summary class="cursor-pointer px-5 py-4 text-sm font-medium text-gray-800 dark:text-dark-100 sm:px-6">{{ t('canvasCard.history') }}</summary>
      <div class="px-5 pb-5 sm:px-6">
        <p class="mb-3 text-xs text-gray-500">{{ t('canvasCard.historyHint') }}</p>
        <p v-if="historyError" role="alert" class="text-sm text-red-600">{{ historyError }} <button class="btn btn-secondary btn-sm" @click="loadHistory(historyPage)">{{ t('common.retry') }}</button></p>
        <p v-if="historyLoading" class="text-sm text-gray-500">{{ t('common.loading') }}</p>
        <ul v-else-if="history?.items.length" class="divide-y divide-gray-100 dark:divide-dark-700">
          <li v-for="item in history.items" :key="item.id" class="flex flex-wrap items-center gap-3 py-3">
            <div class="flex h-11 w-14 items-center justify-center rounded-lg bg-primary-50 text-xl font-semibold tabular-nums text-primary-700 dark:bg-primary-500/10 dark:text-primary-300">{{ item.denomination }}</div>
            <div class="min-w-0 flex-1">
              <p class="text-sm text-gray-900 dark:text-white">{{ t('canvasCard.order', { id: item.id }) }} · {{ sourceLabel(item.funding_source) }}<span v-if="item.team_id"> · {{ t('canvasCard.teamNumber', { id: item.team_id }) }}</span></p>
              <p class="mt-0.5 text-xs text-gray-500">{{ formatDateTime(item.created_at) }}</p>
              <code class="mt-1 block break-all text-sm text-gray-700 dark:text-dark-200">{{ item.code }}</code>
            </div>
            <button class="btn btn-secondary btn-sm" @click="copy(item.code)">{{ t('canvasCard.copy') }}</button>
          </li>
        </ul>
        <p v-else-if="!historyError" class="py-3 text-sm text-gray-500">{{ t('canvasCard.noHistory') }}</p>
        <div v-if="history && history.pages > 1" class="mt-3 flex items-center justify-end gap-3">
          <button class="btn btn-secondary btn-sm" :disabled="historyLoading || history.page <= 1" @click="loadHistory(history.page - 1)">{{ t('pagination.previous') }}</button>
          <span class="text-xs text-gray-500">{{ history.page }} / {{ history.pages }}</span>
          <button class="btn btn-secondary btn-sm" :disabled="historyLoading || history.page >= history.pages" @click="loadHistory(history.page + 1)">{{ t('pagination.next') }}</button>
        </div>
      </div>
    </details>
  </section>

  <BaseDialog :show="result !== null" :title="t('canvasCard.success')" width="narrow" @close="result = null">
    <div v-if="result" class="space-y-4">
      <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('canvasCard.result', { source: sourceLabel(result.funding_source), amount: result.denomination }) }}</p>
      <div class="rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-500/30 dark:bg-primary-500/10">
        <p class="mb-2 text-xs text-primary-700 dark:text-primary-300">{{ t('canvasCard.code') }} · {{ result.denomination }}</p>
        <code class="block select-all break-all text-lg text-gray-900 dark:text-white">{{ result.code }}</code>
      </div>
      <p class="text-xs text-gray-500">{{ t('canvasCard.historyHint') }}</p>
    </div>
    <template #footer>
      <div v-if="result" class="flex flex-wrap justify-end gap-3">
        <button class="btn btn-secondary" @click="copy(result.code)">{{ t('canvasCard.copy') }}</button>
        <a :href="navigationLinks.canvas_url" target="_blank" rel="noopener noreferrer" class="btn btn-primary">{{ t('canvasCard.openCanvas') }} ↗</a>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { canvasCardAPI, type CanvasCardFunding, type CanvasCardOverview, type CanvasCardOrder, type CanvasCardPage, type CanvasCardRequest } from '@/api/canvasCard'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/format'
import navigationLinks from '../../../../../navigation-links.json'

const emit = defineEmits<{ exchanged: [] }>()
const { t } = useI18n()
const app = useAppStore()
const auth = useAuthStore()
const overview = ref<CanvasCardOverview | null>(null)
const loading = ref(false)
const loadError = ref('')
const amount = ref<number | null>(null)
const funding = ref<CanvasCardFunding>('personal')
const submitting = ref(false)
const exchangeError = ref('')
const result = ref<CanvasCardOrder | null>(null)
const pending = ref<CanvasCardRequest | null>(null)
const storageKey = `canvas-card-exchange:${auth.user!.id}`
const history = ref<CanvasCardPage<CanvasCardOrder> | null>(null)
const historyPage = ref(1)
const historyLoading = ref(false)
const historyError = ref('')
const teamAvailable = computed(() => overview.value?.team_enabled && overview.value.team?.status === 'active')
const balance = computed(() => funding.value === 'team' ? overview.value?.team?.balance ?? 0 : overview.value?.personal_balance ?? 0)
const canExchange = computed(() => amount.value !== null && balance.value >= amount.value && (funding.value === 'personal' || teamAvailable.value) && overview.value?.stock.some(item => item.denomination === amount.value && item.available > 0))
const sourceLabel = (source: CanvasCardFunding) => t(source === 'team' ? 'canvasCard.team' : 'canvasCard.personal')

async function load() {
  loading.value = true
  loadError.value = ''
  try { overview.value = await canvasCardAPI.overview() }
  catch (error: any) { loadError.value = error.message || t('canvasCard.loadError') }
  finally { loading.value = false }
}
async function loadHistory(page: number) {
  historyPage.value = page
  historyLoading.value = true
  historyError.value = ''
  try { history.value = await canvasCardAPI.history(page) }
  catch (error: any) { historyError.value = error.message || t('common.error') }
  finally { historyLoading.value = false }
}
function onHistoryToggle(event: Event) {
  if ((event.target as HTMLDetailsElement).open) void loadHistory(1)
}
function clearPending() {
  pending.value = null
  sessionStorage.removeItem(storageKey)
}
async function exchange() {
  if (submitting.value || (!pending.value && !canExchange.value)) return
  if (!pending.value) {
    pending.value = { operation_id: crypto.randomUUID(), denomination: amount.value!, funding_source: funding.value, team_id: funding.value === 'team' ? overview.value!.team!.id : null }
    sessionStorage.setItem(storageKey, JSON.stringify(pending.value))
  }
  submitting.value = true
  exchangeError.value = ''
  try {
    result.value = await canvasCardAPI.exchange(pending.value)
    clearPending()
    amount.value = null
    emit('exchanged')
    await Promise.all([load(), loadHistory(1)])
  } catch (error: any) {
    exchangeError.value = error.message || t('common.error')
    if (error.status >= 400 && error.status < 500 && error.status !== 408) {
      clearPending()
      await load()
    }
  } finally { submitting.value = false }
}
async function copy(code: string) {
  try { await navigator.clipboard.writeText(code); app.showSuccess(t('canvasCard.copied')) }
  catch { app.showError(t('common.error')) }
}
onMounted(() => {
  const saved = sessionStorage.getItem(storageKey)
  if (saved) { pending.value = JSON.parse(saved); amount.value = pending.value!.denomination; funding.value = pending.value!.funding_source }
  void load()
})
</script>

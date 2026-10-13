<template>
  <section v-if="!readonly" class="card flex items-center gap-3 p-4 sm:p-5">
    <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300"><Icon name="gift" size="md" /></div>
    <div class="min-w-0 flex-1">
      <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('canvasCard.title') }}</h2>
      <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('canvasCard.subtitle') }}</p>
    </div>
    <button type="button" class="btn btn-ghost btn-sm shrink-0 text-primary-600 dark:text-primary-300" @click="openExchange">{{ t('redeem.redeemButton') }} <span aria-hidden="true">→</span></button>
  </section>

  <Teleport to="body">
    <dialog ref="dialog" class="canvas-card-drawer fixed inset-y-0 left-auto right-0 m-0 h-dvh max-h-none w-full max-w-md border-0 bg-white p-0 text-gray-900 shadow-2xl dark:bg-dark-900 dark:text-white" :aria-label="t('canvasCard.drawerTitle')" @cancel.prevent="close" @click="onBackdropClick">
      <div class="flex h-full flex-col">
        <header class="flex items-center justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="font-semibold">{{ result ? t('canvasCard.code') : t('canvasCard.drawerTitle') }}</h2>
          <button type="button" class="btn btn-ghost btn-sm px-2" :aria-label="t('common.close')" :disabled="submitting" @click="close"><Icon name="x" size="md" /></button>
        </header>
        <div class="min-h-0 flex-1 overflow-y-auto p-5">
          <div v-if="result" class="space-y-5">
            <RedeemCelebration :sequence="celebrationSequence" :title="t('canvasCard.success')" :detail="t('canvasCard.activityTitle', {amount: result.denomination})">
              <div class="flex min-h-16 items-center gap-3">
                <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10"><Icon name="check" size="lg" /></div>
                <div><p class="text-sm font-medium">{{ t('canvasCard.activityTitle', {amount: result.denomination}) }}</p><p class="mt-1 text-xs text-gray-500">{{ formatDateTime(result.created_at) }}</p></div>
              </div>
            </RedeemCelebration>
            <div class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
              <p class="mb-2 text-xs text-gray-500">{{ t('canvasCard.code') }}</p>
              <code class="block select-all break-all text-base">{{ result.code }}</code>
              <button class="btn btn-secondary btn-sm mt-4 w-full" @click="copy(result.code)">{{ t('canvasCard.copy') }}</button>
            </div>
            <dl class="space-y-3 text-sm"><div class="flex justify-between gap-3"><dt class="text-gray-500">{{ t('canvasCard.funding') }}</dt><dd>{{ sourceLabel(result.funding_source) }}<span v-if="result.team_id"> · #{{ result.team_id }}</span></dd></div><div class="flex justify-between gap-3"><dt class="text-gray-500">{{ t('canvasCard.paid') }}</dt><dd class="tabular-nums">{{ result.denomination }}</dd></div></dl>
            <p class="text-xs leading-5 text-gray-500">{{ t('canvasCard.historyHint') }}</p>
          </div>
          <p v-else-if="loading" class="py-6 text-center text-sm text-gray-500">{{ t('common.loading') }}</p>
          <div v-else-if="loadError" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ loadError }}<button class="btn btn-secondary btn-sm mt-3 block" @click="load">{{ t('common.retry') }}</button></div>
          <div v-else-if="overview" class="space-y-6">
            <p class="text-sm leading-6 text-gray-500 dark:text-dark-400">{{ t('canvasCard.hint') }}</p>
            <fieldset :disabled="submitting || pending !== null">
              <legend class="mb-3 text-sm font-medium">{{ t('canvasCard.denomination') }}</legend>
              <div class="space-y-2">
                <label v-for="item in overview.stock" :key="item.denomination" class="flex cursor-pointer items-center gap-3 rounded-xl border px-3 py-3 transition-colors" :class="[amount === item.denomination ? 'border-primary-400 bg-primary-50/60 dark:bg-primary-500/10' : 'border-gray-200 dark:border-dark-700', item.available === 0 && 'cursor-not-allowed opacity-50']">
                  <input v-model="amount" type="radio" name="canvas-card-amount" :value="item.denomination" :disabled="item.available === 0" class="text-primary-600" />
                  <span class="flex-1 text-sm font-medium">{{ t('canvasCard.amountLabel', {amount:item.denomination}) }}</span>
                  <span class="text-xs text-gray-500">{{ item.available ? t('canvasCard.available', {count:item.available}) : t('canvasCard.soldOut') }}</span>
                </label>
              </div>
            </fieldset>
            <div>
              <label for="canvas-card-funding" class="input-label">{{ t('canvasCard.funding') }}</label>
              <Select id="canvas-card-funding" v-model="funding" :options="fundingOptions" :disabled="submitting || pending !== null" />
              <p v-if="funding === 'team'" class="mt-2 text-xs leading-5 text-gray-500">{{ t('canvasCard.teamHint') }}</p>
            </div>
            <p v-if="pending && !submitting" class="text-sm leading-6 text-amber-700 dark:text-amber-300">{{ t('canvasCard.pending') }}</p>
            <p v-else-if="amount !== null && balance < amount" class="text-sm text-red-600 dark:text-red-400">{{ t('canvasCard.insufficient') }}</p>
            <p v-if="exchangeError" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ exchangeError }}</p>
          </div>
        </div>
        <footer class="border-t border-gray-100 p-5 dark:border-dark-700">
          <a v-if="result" :href="navigationLinks.canvas_url" target="_blank" rel="noopener noreferrer" class="btn btn-primary w-full">{{ t('canvasCard.openCanvas') }} ↗</a>
          <button v-else class="btn btn-primary w-full" :disabled="submitting || loading || !!loadError || (!pending && !canExchange)" @click="exchange">{{ submitting ? t('canvasCard.exchanging') : pending ? t('canvasCard.retry') : amount === null ? t('canvasCard.choose') : t('canvasCard.exchange', {amount}) }}</button>
        </footer>
      </div>
    </dialog>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import RedeemCelebration from '@/components/user/RedeemCelebration.vue'
import { canvasCardAPI, type CanvasCardFunding, type CanvasCardOverview, type CanvasCardOrder, type CanvasCardRequest } from '@/api/canvasCard'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/format'
import navigationLinks from '../../../../../navigation-links.json'

const props = withDefaults(defineProps<{readonly?: boolean}>(), {readonly:false})
const emit = defineEmits<{exchanged: []}>()
const { t } = useI18n()
const app = useAppStore()
const auth = useAuthStore()
const dialog = ref<HTMLDialogElement | null>(null)
const overview = ref<CanvasCardOverview | null>(null)
const loading = ref(false)
const loadError = ref('')
const amount = ref<number | null>(null)
const funding = ref<CanvasCardFunding>('personal')
const submitting = ref(false)
const exchangeError = ref('')
const result = ref<CanvasCardOrder | null>(null)
const pending = ref<CanvasCardRequest | null>(null)
const celebrationSequence = ref(0)
let nextCelebration = 0
const storageKey = computed(() => `canvas-card-exchange:${auth.user!.id}`)
const teamAvailable = computed(() => overview.value?.team_enabled && overview.value.team?.status === 'active')
const balance = computed(() => funding.value === 'team' ? overview.value?.team?.balance ?? 0 : overview.value?.personal_balance ?? 0)
const canExchange = computed(() => amount.value !== null && balance.value >= amount.value && (funding.value === 'personal' || teamAvailable.value) && overview.value?.stock.some(item => item.denomination === amount.value && item.available > 0))
const sourceLabel = (source: CanvasCardFunding) => t(source === 'team' ? 'canvasCard.team' : 'canvasCard.personal')
const fundingOptions = computed(() => {
  if (!overview.value) return []
  const options = [{value:'personal',label:`${t('canvasCard.personal')} · ${overview.value.personal_balance.toFixed(2)}`,disabled:false}]
  if (overview.value.team) options.push({value:'team',label:`${t('canvasCard.team')} · ${overview.value.team.name} · ${overview.value.team.balance.toFixed(2)}${teamAvailable.value ? '' : ` (${t('canvasCard.teamUnavailable')})`}`,disabled:!teamAvailable.value})
  return options
})
function close() { if (!submitting.value) dialog.value?.close() }
function onBackdropClick(event: MouseEvent) { if (event.target === dialog.value) close() }
async function openExchange() {
  if (props.readonly) return
  result.value = null
  celebrationSequence.value = 0
  exchangeError.value = ''
  const saved = sessionStorage.getItem(storageKey.value)
  pending.value = saved ? JSON.parse(saved) : null
  amount.value = pending.value?.denomination ?? null
  funding.value = pending.value?.funding_source ?? 'personal'
  await nextTick()
  dialog.value!.showModal()
  await load()
}
async function showOrder(order: CanvasCardOrder) {
  result.value = order
  celebrationSequence.value = 0
  await nextTick()
  dialog.value!.showModal()
}
async function load() {
  loading.value = true; loadError.value = ''
  try { overview.value = await canvasCardAPI.overview() }
  catch (error: any) { loadError.value = error.message || t('canvasCard.loadError') }
  finally { loading.value = false }
}
function clearPending() { pending.value = null; sessionStorage.removeItem(storageKey.value) }
async function exchange() {
  if (props.readonly || submitting.value || (!pending.value && !canExchange.value)) return
  if (!pending.value) {
    pending.value = {operation_id:crypto.randomUUID(),denomination:amount.value!,funding_source:funding.value,team_id:funding.value === 'team' ? overview.value!.team!.id : null}
    sessionStorage.setItem(storageKey.value,JSON.stringify(pending.value))
  }
  submitting.value = true; exchangeError.value = ''
  try {
    const issued = await canvasCardAPI.exchange(pending.value)
    clearPending()
    celebrationSequence.value = ++nextCelebration
    result.value = issued
    amount.value = null
    emit('exchanged')
  } catch (error: any) {
    exchangeError.value = error.message || t('common.error')
    if (error.status >= 400 && error.status < 500 && error.status !== 408) {clearPending(); await load()}
  } finally { submitting.value = false }
}
async function copy(code: string) {
  try { await navigator.clipboard.writeText(code); app.showSuccess(t('canvasCard.copied')) }
  catch { app.showError(t('common.error')) }
}
defineExpose({showOrder})
</script>

<style scoped>
.canvas-card-drawer::backdrop { background: rgb(0 0 0 / 35%); }
.canvas-card-drawer[open] { animation: canvas-card-drawer-in var(--motion-layout) var(--motion-ease); }
@keyframes canvas-card-drawer-in { from { transform: translateX(100%); } to { transform: translateX(0); } }
@media (prefers-reduced-motion: reduce) { .canvas-card-drawer[open] { animation: none; } }
</style>

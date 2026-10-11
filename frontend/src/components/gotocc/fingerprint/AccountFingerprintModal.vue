<template>
  <BaseDialog :show="show" :title="t('fingerprint.title')" width="extra-wide" @close="close">
    <div v-if="account" class="space-y-5">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('fingerprint.account') }}</p>
          <div class="mt-1 flex items-center gap-2">
            <h4 class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ account.name }}</h4>
            <span class="rounded-md bg-gray-100 px-2 py-0.5 font-mono text-xs text-gray-500 dark:bg-dark-700 dark:text-dark-300">#{{ account.id }}</span>
          </div>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('fingerprint.description') }}</p>
        </div>
        <span class="rounded-full border border-sky-200 bg-sky-50 px-3 py-1 text-xs font-medium text-sky-700 dark:border-sky-900 dark:bg-sky-950/40 dark:text-sky-300">
          {{ t('fingerprint.fixedAccount') }}
        </span>
      </div>

      <div class="grid gap-3 rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-800/60 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)] sm:items-center">
        <div class="space-y-1.5">
          <label for="fingerprint-model" class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('fingerprint.model') }}</label>
          <Select
            id="fingerprint-model"
            v-model="selectedModel"
            :options="modelOptions"
            :placeholder="loadingModels ? t('common.loading') : t('fingerprint.selectModel')"
            :disabled="busy || loadingModels"
          />
        </div>
        <div class="space-y-1 text-xs leading-relaxed text-gray-500 dark:text-dark-400">
          <p class="text-gray-700 dark:text-dark-200">{{ t('fingerprint.runHint', { count: parameters.challenge_count }) }}</p>
          <p>{{ t('fingerprint.privacy') }}</p>
        </div>
      </div>
      <p v-if="!loadingModels && !modelOptions.length" class="text-sm text-amber-700 dark:text-amber-300">{{ t('fingerprint.noModels') }}</p>
      <p v-if="errorMessage" role="alert" class="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300">{{ errorMessage }}</p>

      <div class="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.15fr)]">
        <div class="space-y-3">
          <div v-for="(sample, index) in samples" :key="sample.id" class="overflow-hidden rounded-xl border border-gray-200 dark:border-dark-600">
            <div class="flex items-center gap-3 px-4 py-3">
              <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg font-mono text-sm font-semibold"
                :class="sample.status === 'streaming' ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300'">
                {{ String(index + 1).padStart(2, '0') }}
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex items-center justify-between gap-2">
                  <span class="text-sm font-medium text-gray-800 dark:text-dark-100">{{ t('fingerprint.sample', { index: index + 1 }) }}</span>
                  <span class="text-xs" :class="sample.status === 'failed' ? 'text-red-600 dark:text-red-400' : 'text-gray-500 dark:text-dark-400'">{{ t(`fingerprint.${sample.status}`) }}</span>
                </div>
                <div class="mt-1 flex flex-wrap justify-between gap-1 text-xs text-gray-500 dark:text-dark-400">
                  <span>{{ t('fingerprint.target', { count: sample.expected_count }) }}</span>
                  <span class="font-mono tabular-nums">{{ t('fingerprint.numbers', { count: parseNumbers(sample.text).length }) }}</span>
                </div>
              </div>
            </div>
            <p v-if="sample.error" class="border-t border-red-100 bg-red-50 px-4 py-2 text-xs leading-relaxed text-red-700 dark:border-red-950 dark:bg-red-950/20 dark:text-red-300">{{ sample.error }}</p>
            <details class="border-t border-gray-100 dark:border-dark-700">
              <summary class="cursor-pointer px-4 py-2 text-xs text-gray-500 hover:text-sky-700 dark:text-dark-400 dark:hover:text-sky-300">{{ t('fingerprint.detail') }}</summary>
              <div class="space-y-3 px-4 pb-4">
                <div>
                  <p class="mb-1 text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('fingerprint.prompt') }}</p>
                  <p class="text-xs leading-relaxed text-gray-500 dark:text-dark-400">{{ sample.prompt }}</p>
                </div>
                <div>
                  <p class="mb-1 text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('fingerprint.reply') }}</p>
                  <pre class="max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-gray-50 p-3 text-xs leading-relaxed text-gray-700 dark:bg-dark-900 dark:text-dark-200">{{ sample.text || t('fingerprint.emptyReply') }}</pre>
                </div>
              </div>
            </details>
          </div>
        </div>

        <section class="overflow-hidden rounded-xl border border-gray-200 dark:border-dark-600" aria-live="polite">
          <div class="border-b border-gray-100 px-5 py-3 dark:border-dark-700">
            <h4 class="text-sm font-semibold text-gray-800 dark:text-dark-100">{{ t('fingerprint.result') }}</h4>
          </div>
          <div v-if="!analysis" class="flex min-h-64 flex-col items-center justify-center px-6 py-8 text-center">
            <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-2xl bg-sky-50 text-sky-600 dark:bg-sky-950/40 dark:text-sky-300">
              <Icon :name="busy ? 'refresh' : 'chart'" size="lg" :class="{ 'animate-spin': busy }" />
            </div>
            <p class="text-sm font-medium text-gray-800 dark:text-dark-100">{{ progressLabel || t('fingerprint.beforeResult') }}</p>
            <p class="mt-2 max-w-sm text-xs leading-relaxed text-gray-500 dark:text-dark-400">{{ t('fingerprint.beforeResultHint') }}</p>
          </div>
          <div v-else class="space-y-4 p-5">
            <div class="flex items-center justify-between gap-3 text-xs text-gray-500 dark:text-dark-400">
              <span>{{ t('fingerprint.usedSamples', { count: analysis.used_outputs, total: parameters.challenge_count }) }}</span>
              <span>{{ t('fingerprint.reference', { count: bank.models.length }) }}</span>
            </div>
            <div v-if="candidates.length">
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('fingerprint.closest') }}</p>
              <p class="mt-1 break-words text-xl font-semibold tracking-tight text-gray-900 dark:text-gray-100">{{ candidates[0].display_name }}</p>
              <div class="mt-4 space-y-3">
                <div v-for="(candidate, index) in candidates" :key="candidate.model">
                  <div class="flex items-start justify-between gap-3 text-sm">
                    <div class="flex min-w-0 items-center gap-2">
                      <span class="font-mono text-xs text-gray-400 dark:text-dark-500">{{ index + 1 }}</span>
                      <span class="break-words text-gray-700 dark:text-dark-200">{{ candidate.display_name }}</span>
                    </div>
                    <span class="shrink-0 font-mono tabular-nums text-gray-800 dark:text-dark-100">{{ candidate.probability === null ? '—' : `${(candidate.probability * 100).toFixed(1)}%` }}</span>
                  </div>
                  <div v-if="candidate.probability !== null" class="mt-1.5 h-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                    <div class="h-full rounded-full bg-sky-500" :style="{ width: `${candidate.probability * 100}%` }"></div>
                  </div>
                  <p class="mt-1 text-right text-[11px] text-gray-400 dark:text-dark-500">{{ t('fingerprint.score') }} {{ candidate.score.toFixed(3) }}</p>
                </div>
              </div>
              <p class="mt-3 text-right text-xs text-gray-500 dark:text-dark-400">{{ t('fingerprint.confidence') }}</p>
            </div>
            <p v-else class="text-sm leading-relaxed text-amber-700 dark:text-amber-300">{{ t('fingerprint.unscorable') }}</p>
            <p v-if="analysis.decision === 'partial'" class="text-xs leading-relaxed text-amber-700 dark:text-amber-300">{{ t('fingerprint.partial') }}</p>
            <p v-if="upstreamModel" class="break-words text-xs text-gray-500 dark:text-dark-400">{{ t('fingerprint.mappedModel', { model: upstreamModel }) }}</p>
          </div>
          <div class="space-y-2 border-t border-gray-100 bg-gray-50 px-5 py-3 text-xs leading-relaxed dark:border-dark-700 dark:bg-dark-800/60">
            <p v-if="selectedModel && !modelInBank" class="text-amber-700 dark:text-amber-300">{{ t('fingerprint.notInBank', { model: upstreamModel || selectedModel }) }}</p>
            <p class="text-gray-500 dark:text-dark-400">{{ t('fingerprint.limitation') }}</p>
          </div>
        </section>
      </div>
    </div>
    <template #footer>
      <div class="flex w-full flex-wrap items-center justify-between gap-3">
        <a :href="parameters.reference_repository" target="_blank" rel="noopener noreferrer" class="text-xs text-gray-500 hover:text-sky-600 dark:text-dark-400 dark:hover:text-sky-300">
          {{ t('fingerprint.source') }} · LM Fingerpoint Detector
        </a>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary" type="button" @click="busy ? stop() : close()">{{ busy ? t('fingerprint.stop') : t('common.close') }}</button>
          <button class="btn btn-primary" type="button" :disabled="busy || loadingModels || !selectedModel || !account || !supportsFingerprintPlatform(account.platform)" @click="start">
            {{ busy ? progressLabel : hasRun ? t('fingerprint.rerun') : t('fingerprint.start') }}
          </button>
        </div>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { Account } from '@/types'
import parameters from './parameters.gen.json'
import bank from './vendor/bank.json'
import { generateChallenges } from './vendor/challenge-browser.js'
import { parseNumbers } from './vendor/fingerprint-core.js'
import type { Analysis, Challenge, Output } from './vendor/types'
import { isFingerprintTextModel, sampleAccountFingerprint, supportsFingerprintPlatform } from './accountFingerprint'

const props = defineProps<{ show: boolean; account: Account | null }>()
const emit = defineEmits<{ (event: 'close'): void }>()
const { t } = useI18n()
type Sample = Challenge & { text: string; status: 'waiting' | 'streaming' | 'completed' | 'failed' | 'stopped'; error: string }
const samples = ref<Sample[]>([])
const modelOptions = ref<{ value: string; label: string }[]>([])
const selectedModel = ref('')
const upstreamModel = ref('')
const loadingModels = ref(false)
const state = ref<'idle' | 'loading' | 'running' | 'analyzing' | 'done' | 'cancelled'>('idle')
const hasRun = ref(false)
const currentSample = ref(0)
const errorMessage = ref('')
const analysis = ref<Analysis | null>(null)
let controller: AbortController | null = null
let worker: Worker | null = null
let rejectWorker: ((error: Error) => void) | null = null

const busy = computed(() => ['loading', 'running', 'analyzing'].includes(state.value))
const candidates = computed(() => analysis.value?.results.slice(0, parameters.top_candidates) || [])
const normalizeModel = (model: string) => model.toLowerCase().replace(/\./g, '-')
const modelInBank = computed(() => bank.models.some(model => normalizeModel(model.id) === normalizeModel(upstreamModel.value || selectedModel.value)))
const progressLabel = computed(() => {
  if (state.value === 'running') return t('fingerprint.running', { current: currentSample.value, total: parameters.challenge_count })
  if (state.value === 'loading') return t('fingerprint.loading')
  if (state.value === 'analyzing') return t('fingerprint.analyzing')
  if (state.value === 'cancelled') return t('fingerprint.cancelled')
  return ''
})
const newSamples = () => generateChallenges(parameters.challenge_count).map(challenge => ({ ...challenge, text: '', status: 'waiting' as const, error: '' }))

function workerRequest(message: { type: 'load' | 'analyze'; outputs?: Output[] }): Promise<Analysis | null> {
  return new Promise((resolve, reject) => {
    rejectWorker = reject
    worker!.onmessage = (event: MessageEvent<{ type: string; analysis?: Analysis; message?: string }>) => {
      rejectWorker = null
      if (event.data.type === 'error') reject(new Error(event.data.message))
      else resolve(event.data.analysis ?? null)
    }
    worker!.onerror = event => { rejectWorker = null; reject(new Error(event.message)) }
    worker!.postMessage(message)
  })
}

function stop() {
  controller?.abort()
  controller = null
  rejectWorker?.(new DOMException('Detection stopped', 'AbortError'))
  rejectWorker = null
  worker?.terminate()
  worker = null
  for (const sample of samples.value) if (sample.status === 'streaming' || sample.status === 'waiting') sample.status = 'stopped'
  if (busy.value) state.value = 'cancelled'
}

function close() { stop(); emit('close') }

async function start() {
  if (!props.account || !selectedModel.value || busy.value) return
  const accountID = props.account.id, modelID = selectedModel.value
  if (hasRun.value) samples.value = newSamples()
  hasRun.value = true
  analysis.value = null
  errorMessage.value = ''
  upstreamModel.value = ''
  const run = new AbortController()
  controller = run
  try {
    state.value = 'loading'
    if (!worker) {
      worker = new Worker(new URL('./fingerprint.worker.ts', import.meta.url), { type: 'module' })
      await workerRequest({ type: 'load' })
    }
    if (run.signal.aborted) return
    state.value = 'running'
    for (const [index, sample] of samples.value.entries()) {
      currentSample.value = index + 1
      sample.status = 'streaming'
      try {
        sample.text = await sampleAccountFingerprint(accountID, modelID, sample.prompt, run.signal,
          text => { if (!run.signal.aborted) sample.text = text },
          model => { if (!run.signal.aborted) upstreamModel.value = model })
        sample.status = 'completed'
      } catch (error) {
        if (run.signal.aborted) return
        sample.status = 'failed'
        sample.error = error instanceof Error ? error.message : String(error)
      }
    }
    if (run.signal.aborted) return
    state.value = 'analyzing'
    analysis.value = await workerRequest({ type: 'analyze', outputs: samples.value.map(sample => ({ text: sample.status === 'completed' ? sample.text : '', expected_count: sample.expected_count })) })
    state.value = 'done'
  } catch (error) {
    if (run.signal.aborted) return
    const message = error instanceof Error ? error.message : String(error)
    errorMessage.value = t(state.value === 'loading' ? 'fingerprint.loadFailed' : 'fingerprint.workerFailed', { message })
    worker?.terminate()
    worker = null
    state.value = 'done'
  } finally {
    if (controller === run) controller = null
  }
}

watch(() => props.show, async visible => {
  stop()
  if (!visible || !props.account) return
  const accountID = props.account.id
  state.value = 'idle'
  hasRun.value = false
  analysis.value = null
  errorMessage.value = ''
  upstreamModel.value = ''
  selectedModel.value = ''
  modelOptions.value = []
  samples.value = newSamples()
  loadingModels.value = true
  try {
    const models = await adminAPI.accounts.getAvailableModels(accountID)
    if (!props.show || props.account?.id !== accountID) return
    modelOptions.value = models.filter(model => isFingerprintTextModel(model.id)).map(model => ({ value: model.id, label: model.display_name || model.id }))
  } catch (error) {
    if (props.show && props.account?.id === accountID) errorMessage.value = error instanceof Error ? error.message : String(error)
  } finally {
    if (props.account?.id === accountID) loadingModels.value = false
  }
}, { immediate: true })

watch(selectedModel, () => { analysis.value = null; upstreamModel.value = '' })
onBeforeUnmount(stop)
</script>

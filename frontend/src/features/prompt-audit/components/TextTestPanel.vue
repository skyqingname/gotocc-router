<template>
  <section class="py-6 sm:py-8" aria-labelledby="prompt-text-test-title">
    <div class="mb-6 flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 id="prompt-text-test-title" class="text-lg font-semibold text-gray-950 dark:text-white">{{ t('admin.promptAudit.textTest.title') }}</h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">{{ t('admin.promptAudit.textTest.description') }}</p>
      </div>
      <span class="rounded-full bg-gray-100 px-3 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">{{ t('admin.promptAudit.textTest.savedPolicy', { version: config.config_version }) }}</span>
    </div>
    <p v-if="dirty" class="mb-5 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:bg-amber-950/30 dark:text-amber-200">{{ t('admin.promptAudit.textTest.dirty') }}</p>
    <div class="grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_minmax(300px,0.85fr)]">
      <form class="min-w-0" @submit.prevent="run">
        <div class="mb-2 flex items-center justify-between gap-3">
          <label for="prompt-test-input" class="text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.promptAudit.textTest.input') }}</label>
          <span class="text-xs tabular-nums text-gray-400">{{ t('admin.promptAudit.textTest.characters', { count: characters, limit: config.text_test_max_runes }) }}</span>
        </div>
        <textarea id="prompt-test-input" v-model="text" rows="12" class="input w-full resize-y leading-7" :placeholder="t('admin.promptAudit.textTest.placeholder')" :disabled="loading" />
        <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('admin.promptAudit.textTest.longInput') }}</p>
        <div class="mt-5 flex items-center gap-3">
          <button type="submit" class="btn btn-primary" :disabled="loading || !text.trim() || tooLong">{{ t(loading ? 'admin.promptAudit.textTest.running' : 'admin.promptAudit.textTest.run') }}</button>
          <button v-if="loading" type="button" class="btn btn-secondary" @click="controller?.abort()">{{ t('admin.promptAudit.textTest.cancel') }}</button>
        </div>
      </form>

      <div class="min-w-0 rounded-2xl border border-gray-200 bg-gray-50/60 p-5 sm:p-6 dark:border-dark-700 dark:bg-dark-900/40" aria-live="polite" :aria-busy="loading">
        <template v-if="result || error">
          <div class="mb-5 flex items-center gap-3">
            <span class="h-3 w-3 rounded-full" :class="toneDot" />
            <h3 class="text-xl font-semibold" :class="toneText">{{ outcome }}</h3>
          </div>
          <template v-if="result?.ok && result.result">
            <div v-if="score !== undefined" class="mb-5 rounded-xl bg-white px-4 py-4 dark:bg-dark-800">
              <div class="flex items-end justify-between gap-3">
                <span class="text-sm text-gray-500 dark:text-dark-300">{{ t(result.response_format === 'jev' ? 'admin.promptAudit.jev.score' : 'admin.promptAudit.textTest.score') }}</span>
                <strong class="text-3xl font-semibold tabular-nums text-gray-950 dark:text-white">{{ score.toFixed(2) }}</strong>
              </div>
              <div class="my-3 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700"><div class="h-full rounded-full" :class="toneDot" :style="{ width: `${score * 100}%` }" /></div>
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.promptAudit.textTest.threshold') }} ≥ {{ result.confidence_threshold }}</p>
            </div>
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('admin.promptAudit.textTest.reason') }}</p>
            <p class="mt-2 whitespace-pre-wrap break-words text-sm leading-6 text-gray-900 dark:text-dark-100">{{ reason || t('admin.promptAudit.textTest.noReason') }}</p>
          </template>
          <template v-else>
            <p class="text-sm leading-6 text-gray-700 dark:text-dark-200">{{ error || failureMessage }}</p>
            <p v-if="result" class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('admin.promptAudit.textTest.failureBody') }}</p>
            <p v-if="result?.error_code" class="mt-3 break-all font-mono text-xs text-gray-500">{{ result.error_code }}</p>
          </template>
          <dl v-if="result" class="mt-6 grid grid-cols-2 gap-x-4 gap-y-4 border-t border-gray-200 pt-5 text-sm dark:border-dark-700">
            <div><dt class="text-xs text-gray-500">{{ t('admin.promptAudit.textTest.latency') }}</dt><dd class="mt-1 tabular-nums text-gray-900 dark:text-white">{{ (result.latency_ms / 1000).toFixed(2) }} s</dd></div>
            <div v-if="result.result"><dt class="text-xs text-gray-500">{{ t('admin.promptAudit.textTest.chunks') }}</dt><dd class="mt-1 text-gray-900 dark:text-white">{{ result.result.chunk_total }}</dd></div>
            <div v-if="result.http_status"><dt class="text-xs text-gray-500">{{ t('admin.promptAudit.textTest.httpStatus') }}</dt><dd class="mt-1 text-gray-900 dark:text-white">{{ result.http_status }}</dd></div>
            <div class="col-span-2"><dt class="text-xs text-gray-500">{{ t('admin.promptAudit.textTest.mode') }}</dt><dd class="mt-1 text-gray-900 dark:text-white">{{ t(`admin.promptAudit.mode.${result.effective_mode}`) }}</dd></div>
            <div v-if="endpointName" class="col-span-2"><dt class="text-xs text-gray-500">{{ t('admin.promptAudit.textTest.node') }}</dt><dd class="mt-1 break-words text-gray-900 dark:text-white">{{ endpointName }}</dd></div>
            <div v-if="result.result?.scanner_version" class="col-span-2"><dt class="text-xs text-gray-500">{{ t('admin.promptAudit.textTest.model') }}</dt><dd class="mt-1 break-words text-gray-900 dark:text-white">{{ result.result.scanner_version }}</dd></div>
          </dl>
        </template>
        <div v-else class="flex min-h-64 flex-col items-center justify-center text-center">
          <span class="mb-4 block h-10 w-px bg-gray-300 dark:bg-dark-600" />
          <h3 class="text-base font-medium text-gray-900 dark:text-white">{{ t(loading ? 'admin.promptAudit.textTest.running' : 'admin.promptAudit.textTest.waitingTitle') }}</h3>
          <p class="mt-2 max-w-xs text-sm leading-6 text-gray-500 dark:text-dark-400">{{ t('admin.promptAudit.textTest.waitingBody') }}</p>
        </div>
      </div>
    </div>
    <p class="mt-6 border-t border-gray-100 pt-5 text-xs leading-6 text-gray-500 dark:border-dark-700 dark:text-dark-400">{{ t('admin.promptAudit.textTest.scope') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { extractApiErrorMessage } from '@/utils/apiError'
import { testText } from '../api'
import type { PromptAuditConfig, PromptTextPreviewResult } from '../types'

const props = defineProps<{ config: PromptAuditConfig; dirty: boolean }>()
const { t } = useI18n()
const text = ref('')
const loading = ref(false)
const result = ref<PromptTextPreviewResult | null>(null)
const error = ref('')
let controller: AbortController | null = null
const characters = computed(() => Array.from(text.value).length)
const tooLong = computed(() => props.config.text_test_max_runes !== undefined && characters.value > props.config.text_test_max_runes)
const score = computed(() => result.value?.result?.scanner_scores.confidence)
const reason = computed(() => {
  if (result.value?.response_format === 'jev') {
    const categories = result.value.result?.categories ?? []
    return categories.length
      ? categories.map(category => t(`admin.promptAudit.scanners.${category}`)).join('\n')
      : t('admin.promptAudit.jev.noMatch')
  }
  return Object.values(result.value?.result?.scanner_evidence ?? {}).filter(Boolean).join('\n')
})
const endpointName = computed(() => props.config.endpoints.find(endpoint => endpoint.id === result.value?.guard_endpoint_id)?.name)
const failed = computed(() => !!error.value || (result.value !== null && !result.value.ok))
const outcome = computed(() => t(`admin.promptAudit.textTest.${failed.value ? 'failed' : result.value?.decision}`))
const toneDot = computed(() => failed.value ? 'bg-amber-500' : result.value?.would_block ? 'bg-red-500' : result.value?.decision === 'flag' ? 'bg-amber-500' : 'bg-emerald-500')
const toneText = computed(() => failed.value ? 'text-amber-700 dark:text-amber-300' : result.value?.would_block ? 'text-red-700 dark:text-red-300' : 'text-gray-950 dark:text-white')
const failureMessage = computed(() => t(`admin.promptAudit.textTest.${result.value?.error_kind || 'unavailable'}`))

async function run() {
  result.value = null
  error.value = ''
  loading.value = true
  controller = new AbortController()
  try {
    result.value = await testText(text.value, controller.signal)
  } catch (failure) {
    error.value = controller.signal.aborted ? t('admin.promptAudit.textTest.cancelled') : extractApiErrorMessage(failure, t('admin.promptAudit.textTest.unavailable'))
  } finally {
    loading.value = false
    controller = null
  }
}
onBeforeUnmount(() => controller?.abort())
</script>

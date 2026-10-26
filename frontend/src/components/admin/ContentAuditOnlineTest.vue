<template>
  <section class="min-w-0 space-y-4" aria-labelledby="content-audit-online-title">
    <div class="flex items-start gap-3">
      <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-300">
        <Icon name="beaker" size="md" />
      </span>
      <div>
        <h3 id="content-audit-online-title" class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.riskControl.onlineTest.title') }}</h3>
        <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('admin.riskControl.onlineTest.hint') }}</p>
      </div>
    </div>

    <form class="space-y-3" @submit.prevent="run">
      <div>
        <label class="input-label" for="content-audit-online-endpoint">{{ t('admin.riskControl.onlineTest.endpoint') }}</label>
        <Select id="content-audit-online-endpoint" v-model="endpointID" :options="endpointOptions" :searchable="false" :disabled="running" :placeholder="t('admin.riskControl.onlineTest.selectEndpoint')" />
      </div>
      <div>
        <label class="sr-only" for="content-audit-online-prompt">{{ t('admin.riskControl.onlineTest.input') }}</label>
        <textarea id="content-audit-online-prompt" v-model="prompt" class="input min-h-28 resize-y text-sm" :disabled="running" :placeholder="t('admin.riskControl.onlineTest.placeholder')"></textarea>
      </div>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.riskControl.onlineTest.savedConfig') }}</p>
        <button type="submit" class="btn btn-primary inline-flex items-center gap-2" :disabled="running || !endpointID || !prompt.trim()">
          <Icon name="beaker" size="sm" :class="{ 'animate-pulse': running }" />
          {{ t(running ? 'admin.riskControl.onlineTest.running' : 'admin.riskControl.onlineTest.submit') }}
        </button>
      </div>
    </form>

    <div v-if="error || result" class="space-y-3 rounded-lg border border-gray-200 p-4 dark:border-dark-600" aria-live="polite" :aria-busy="running">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span class="inline-flex rounded-full px-2.5 py-1 text-xs font-semibold" :class="error ? 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300' : result?.audit_result?.flagged ? 'bg-amber-50 text-amber-800 dark:bg-amber-900/20 dark:text-amber-300' : 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'">
          {{ t(error ? 'admin.riskControl.onlineTest.failed' : result?.audit_result?.flagged ? 'admin.riskControl.onlineTest.hit' : 'admin.riskControl.onlineTest.pass') }}
        </span>
        <span v-if="result" class="text-xs tabular-nums text-gray-500 dark:text-gray-400">
          {{ result.http_status ? `HTTP ${result.http_status}` : t('admin.riskControl.onlineTest.noResponse') }} · {{ result.latency_ms }} ms
        </span>
      </div>
      <p v-if="result" class="break-words text-xs text-gray-500 dark:text-gray-400">{{ result.endpoint_name }} · {{ result.model }} · {{ result.api_key_masked }}</p>
      <div v-if="error" role="alert">
        <p class="mb-2 text-xs font-medium text-red-700 dark:text-red-300">{{ t('admin.riskControl.onlineTest.errorDetail') }}</p>
        <pre class="max-h-72 overflow-auto whitespace-pre-wrap break-words rounded-md bg-red-50 p-3 font-mono text-xs leading-5 text-red-800 dark:bg-red-900/10 dark:text-red-200">{{ error }}</pre>
      </div>
      <template v-else-if="result?.audit_result">
        <p class="text-sm font-medium text-gray-900 dark:text-white">{{ verdict }}</p>
        <p class="text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('admin.riskControl.onlineTest.scope') }}</p>
        <div class="max-h-72 overflow-auto">
          <table class="w-full text-left text-xs">
            <thead class="text-gray-500 dark:text-gray-400">
              <tr>
                <th class="pb-2 font-medium">{{ t('admin.riskControl.onlineTest.category') }}</th>
                <th class="pb-2 text-right font-medium">{{ t('admin.riskControl.onlineTest.score') }}</th>
                <th class="pb-2 text-right font-medium">{{ t('admin.riskControl.onlineTest.threshold') }}</th>
                <th class="pb-2 text-right font-medium">{{ t('admin.riskControl.onlineTest.result') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="row in scores" :key="row.category" :class="row.hit ? 'text-red-700 dark:text-red-300' : 'text-gray-600 dark:text-gray-300'">
                <td class="break-all py-2 pr-3">{{ row.category }}</td>
                <td class="py-2 text-right font-mono">{{ percent(row.score) }}</td>
                <td class="py-2 text-right font-mono">{{ row.threshold === undefined ? '—' : percent(row.threshold) }}</td>
                <td class="py-2 pl-3 text-right">{{ t(row.hit ? 'admin.riskControl.onlineTest.matched' : 'admin.riskControl.onlineTest.unmatched') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { testText, type ContentModerationEndpoint, type ContentModerationOnlineTestResult } from '@/api/admin/riskControl'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{ endpoints: ContentModerationEndpoint[] }>()
const { t } = useI18n()
const endpointID = ref('')
const prompt = ref('')
const running = ref(false)
const result = ref<ContentModerationOnlineTestResult | null>(null)
const error = ref('')
let request: AbortController | undefined

const endpointOptions = computed(() => props.endpoints.map(endpoint => ({
  value: endpoint.id,
  label: `${endpoint.name} · ${endpoint.model}`,
})))

watch(() => props.endpoints.map(endpoint => endpoint.id), ids => {
  if (!ids.includes(endpointID.value)) endpointID.value = ids[0] ?? ''
}, { immediate: true })
watch([endpointID, prompt], () => {
  result.value = null
  error.value = ''
})

const scores = computed(() => {
  const audit = result.value?.audit_result
  if (!audit) return []
  return Object.entries(audit.category_scores).map(([category, score]) => ({
    category, score, threshold: audit.thresholds[category],
    hit: audit.thresholds[category] !== undefined && score >= audit.thresholds[category],
  })).sort((a, b) => Number(b.hit) - Number(a.hit) || b.score - a.score)
})
const verdict = computed(() => {
  if (!result.value?.audit_result?.flagged) return t('admin.riskControl.onlineTest.noModelBlock')
  return t(result.value.blocking_enabled ? 'admin.riskControl.onlineTest.wouldBlock' : 'admin.riskControl.onlineTest.blockingOff')
})
const percent = (value: number) => `${(value * 100).toFixed(2)}%`

async function run() {
  running.value = true
  result.value = null
  error.value = ''
  request = new AbortController()
  try {
    result.value = await testText({ endpoint_id: endpointID.value, prompt: prompt.value }, request.signal)
    error.value = result.value.error
  } catch (err) {
    if (!request.signal.aborted) error.value = extractApiErrorMessage(err, t('admin.riskControl.onlineTest.failed'))
  } finally {
    running.value = false
  }
}
onUnmounted(() => request?.abort())
</script>

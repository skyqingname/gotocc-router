<template>
  <section aria-labelledby="audit-prompt-title" class="border-t border-gray-100 py-6 dark:border-dark-800">
    <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700/60 dark:bg-dark-900/20 sm:p-5">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 id="audit-prompt-title" class="text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.promptAudit.auditPrompt.title') }}
          </h2>
          <p class="mt-1 max-w-4xl text-sm text-gray-500 dark:text-dark-300">
            {{ t('admin.promptAudit.auditPrompt.description') }}
          </p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" data-test="restore-default-audit-prompt" @click="restoreDefault">
          {{ t('admin.promptAudit.auditPrompt.restoreDefault') }}
        </button>
      </div>

      <div class="mt-4 grid gap-4 sm:grid-cols-2">
        <label class="text-sm text-gray-700 dark:text-dark-200">
          {{ t('admin.promptAudit.auditPrompt.responseFormat') }}
          <select :value="draft.response_format" class="input mt-2 w-full" @change="updateFormat(($event.target as HTMLSelectElement).value as AuditResponseFormat)">
            <option value="qwen3guard">{{ t('admin.promptAudit.auditPrompt.qwenFormat') }}</option>
            <option value="confidence_json">{{ t('admin.promptAudit.auditPrompt.confidenceFormat') }}</option>
          </select>
        </label>
        <label v-if="draft.response_format === 'confidence_json'" class="text-sm text-gray-700 dark:text-dark-200">
          {{ t('admin.promptAudit.auditPrompt.threshold') }}
          <input :value="draft.confidence_threshold" type="number" min="0" max="1" step="0.01" class="input mt-2 w-full" @input="updateThreshold(Number(($event.target as HTMLInputElement).value))" />
        </label>
      </div>
      <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">
        {{ t(draft.response_format === 'confidence_json' ? 'admin.promptAudit.auditPrompt.confidenceHint' : 'admin.promptAudit.auditPrompt.qwenHint') }}
      </p>

      <div class="mt-4 flex items-center justify-between gap-3">
        <label for="prompt-audit-template" class="text-sm text-gray-700 dark:text-dark-200">
          {{ t('admin.promptAudit.auditPrompt.label') }}
        </label>
        <span class="text-xs tabular-nums" :class="valid ? 'text-gray-500 dark:text-dark-400' : 'text-red-600 dark:text-red-300'">
          {{ characterCount }} / {{ MAX_AUDIT_PROMPT_RUNES }}
        </span>
      </div>
      <textarea
        id="prompt-audit-template"
        :value="draft.audit_prompt"
        rows="18"
        class="input mt-2 min-h-[22rem] w-full resize-y font-mono text-sm leading-6"
        data-test="audit-prompt-editor"
        :aria-invalid="!valid"
        :aria-describedby="valid ? 'audit-prompt-hint' : 'audit-prompt-error'"
        @input="updatePrompt(($event.target as HTMLTextAreaElement).value)"
      />
      <p v-if="valid" id="audit-prompt-hint" class="mt-2 text-xs text-gray-500 dark:text-dark-400">
        {{ t('admin.promptAudit.auditPrompt.hint') }}
      </p>
      <p v-else id="audit-prompt-error" role="alert" class="mt-2 text-xs text-red-600 dark:text-red-300">
        {{ t('admin.promptAudit.auditPrompt.required') }}
      </p>

      <div class="mt-4 rounded-lg bg-gray-50 px-4 py-3 dark:bg-dark-900/50">
        <p class="text-xs font-medium text-gray-700 dark:text-dark-200">{{ t('admin.promptAudit.auditPrompt.deliveryTitle') }}</p>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.promptAudit.auditPrompt.deliveryHint') }}</p>
        <code class="mt-2 block rounded bg-white px-3 py-2 text-xs text-gray-700 dark:bg-dark-800 dark:text-dark-200">&lt;user_input&gt; ... &lt;/user_input&gt;</code>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AuditResponseFormat, PromptAuditDraft } from '../types'
import { cloneData, MAX_AUDIT_PROMPT_RUNES } from '../viewModel'

const props = defineProps<{ draft: PromptAuditDraft }>()
const emit = defineEmits<{ (event: 'update:draft', value: PromptAuditDraft): void }>()
const { t } = useI18n()

const characterCount = computed(() => Array.from(props.draft.audit_prompt).length)
const valid = computed(() => props.draft.audit_prompt.trim().length > 0 && characterCount.value <= MAX_AUDIT_PROMPT_RUNES)

function updatePrompt(value: string) {
  emit('update:draft', { ...cloneData(props.draft), audit_prompt: value })
}

function restoreDefault() {
  updatePrompt(props.draft.response_format === 'confidence_json' ? props.draft.default_confidence_audit_prompt : props.draft.default_audit_prompt)
}

function updateFormat(response_format: AuditResponseFormat) {
  emit('update:draft', { ...cloneData(props.draft), response_format })
}

function updateThreshold(confidence_threshold: number) {
  emit('update:draft', { ...cloneData(props.draft), confidence_threshold })
}
</script>

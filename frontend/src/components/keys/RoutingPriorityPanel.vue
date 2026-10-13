<template>
  <section class="overflow-hidden rounded-xl border border-gray-200 bg-gray-50/70 dark:border-dark-600 dark:bg-dark-800/60" data-test="routing-priority-panel" :aria-label="t('keys.routingPriority.title')" :aria-busy="loading">
    <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-600">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h3 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
          <Icon name="arrowsUpDown" size="sm" aria-hidden="true" />
          {{ t('keys.routingPriority.title') }}
        </h3>
        <span class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">{{ t(allowed ? (draft ? 'keys.routingPriority.customized' : 'keys.routingPriority.editable') : 'keys.routingPriority.readOnly') }}</span>
      </div>
      <p class="mt-2 text-xs leading-relaxed text-gray-600 dark:text-dark-300">{{ t(allowed ? 'keys.routingPriority.editNotice' : 'keys.routingPriority.notice') }}</p>
    </div>

    <div class="space-y-3 p-4">
      <p v-if="loading" role="status" class="text-sm text-gray-500 dark:text-dark-300">{{ t('keys.routingPriority.loading') }}</p>
      <div v-else-if="failed" role="alert" class="flex flex-wrap items-center justify-between gap-2">
        <p class="text-sm text-red-700 dark:text-red-300">{{ t('keys.routingPriority.loadFailed') }}</p>
        <button type="button" class="btn btn-secondary min-h-11" @click="load">{{ t('keys.routingPriority.retry') }}</button>
      </div>
      <template v-else-if="priorities">
        <div v-if="allowed" class="flex flex-wrap items-center justify-between gap-2">
          <span class="text-xs font-medium text-gray-600 dark:text-dark-200">{{ t(draft ? 'keys.routingPriority.customSource' : 'keys.routingPriority.followAdmin') }}</span>
          <button v-if="!draft" type="button" data-test="customize-priority" class="btn btn-secondary min-h-11" :disabled="disabled" @click="customize">{{ t('keys.routingPriority.customize') }}</button>
          <button v-else type="button" data-test="reset-priority" class="btn btn-secondary min-h-11" :disabled="disabled" @click="reset">{{ t('keys.routingPriority.reset') }}</button>
        </div>
        <p v-if="!allowed && savedPreference" class="rounded-lg bg-amber-50 p-3 text-xs leading-relaxed text-amber-800 dark:bg-amber-900/20 dark:text-amber-200">{{ t('keys.routingPriority.disabledNotice') }}</p>
        <template v-if="allowed && draft">
          <RoutingPreferenceEditor :model-value="draft" :groups="availableGroups" :disabled="disabled" @update:model-value="updateDraft" />
          <p class="text-xs leading-relaxed text-gray-500 dark:text-dark-300">{{ t('keys.routingPriority.fallbackHint') }}</p>
        </template>
        <template v-else>
        <template v-if="priorities.groups.length > 1">
          <div class="flex flex-wrap items-baseline justify-between gap-2">
            <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ t('keys.routingPriority.defaultOrder') }}</h4>
            <span class="text-xs text-gray-500 dark:text-dark-300">{{ t('keys.routingPriority.highFirst') }}</span>
          </div>
          <ol class="divide-y divide-gray-100 overflow-hidden rounded-lg border border-gray-200 bg-white dark:divide-dark-600 dark:border-dark-600 dark:bg-dark-700">
            <li v-for="(group, index) in priorities.groups" :key="group.id" data-test="default-group" class="flex min-h-11 items-center gap-3 px-3 py-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-xs font-semibold tabular-nums" :class="index === 0 ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-dark-300'">{{ index + 1 }}</span>
              <span class="min-w-0 break-words text-sm font-medium text-gray-800 dark:text-gray-100">{{ group.name }}</span>
            </li>
          </ol>
          <p class="text-xs leading-relaxed text-gray-500 dark:text-dark-300">{{ priorities.default_source === 'administrator' ? t('keys.routingPriority.adminSource') : t('keys.routingPriority.fallbackSource') }}</p>
        </template>
        <p v-else class="text-sm leading-relaxed text-gray-600 dark:text-dark-300">{{ t('keys.routingPriority.empty') }}</p>

        <details v-if="priorities.model_rules.length" class="group rounded-lg border border-amber-200 bg-amber-50/60 dark:border-amber-800/60 dark:bg-amber-900/10">
          <summary class="cursor-pointer px-3 py-3 text-sm font-medium text-amber-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-500 dark:text-amber-200">
            {{ t('keys.routingPriority.exceptionsTitle', { count: priorities.model_rules.length }) }}
          </summary>
          <div class="space-y-3 px-3 pb-3">
            <p class="text-xs leading-relaxed text-amber-900/80 dark:text-amber-200/80">{{ t('keys.routingPriority.exceptionsHint') }}</p>
            <div v-for="rule in priorities.model_rules" :key="rule.model" class="rounded-lg border border-amber-100 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
              <div class="mb-2 flex flex-wrap items-baseline justify-between gap-1">
                <span class="break-all font-mono text-xs font-semibold text-gray-900 dark:text-gray-100">{{ rule.model }}</span>
                <span v-if="rule.matched_rule !== rule.model" class="break-all text-xs text-gray-500 dark:text-dark-300">{{ t('keys.routingPriority.matchedRule', { rule: rule.matched_rule }) }}</span>
              </div>
              <ol class="space-y-2">
                <li v-for="(group, index) in rule.groups" :key="group.id" data-test="model-group" class="flex items-start gap-2 text-sm text-gray-700 dark:text-dark-200">
                  <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded bg-gray-100 text-xs tabular-nums dark:bg-dark-600">{{ index + 1 }}</span>
                  <span class="min-w-0 break-words">{{ group.name }}</span>
                </li>
              </ol>
            </div>
          </div>
        </details>
        </template>
        <p v-if="dirty" role="status" class="text-xs font-medium text-primary-700 dark:text-primary-300">{{ t('keys.routingPriority.unsaved') }}</p>
        <p v-if="validationError" role="alert" class="text-sm text-red-600 dark:text-red-300">{{ validationError }}</p>
        <p class="text-xs leading-relaxed text-gray-500 dark:text-dark-300">{{ t('keys.routingPriority.catalogHint') }}</p>
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { getRoutingPriorities, type RoutingPriorities, type RoutingPriorityGroup } from '@/api/groups'
import { getKeyRoutingPreference, updateKeyRoutingPreference, type RoutingPreference } from '@/api/keyRoutingPolicy'
import RoutingPreferenceEditor from './RoutingPreferenceEditor.vue'

const props = defineProps<{ scope: 'personal' | 'team'; keyId?: number; disabled?: boolean }>()
const { t } = useI18n()
const priorities = ref<RoutingPriorities | null>(null)
const availableGroups = ref<RoutingPriorityGroup[]>([])
const loading = ref(false)
const failed = ref(false)
const allowed = ref(false)
const draft = ref<RoutingPreference | null>(null)
const savedPreference = ref<RoutingPreference | null>(null)
const dirty = ref(false)
const validationError = ref('')
let controller: AbortController | undefined
let generation = 0
const copy = (value: RoutingPreference | null): RoutingPreference | null => value ? JSON.parse(JSON.stringify(value)) : null

async function load() {
  controller?.abort()
  controller = new AbortController()
  const current = ++generation
  loading.value = true
  failed.value = false
  priorities.value = null
  allowed.value = false
  dirty.value = false
  validationError.value = ''
  draft.value = null
  savedPreference.value = null
  try {
    const [data, state] = await Promise.all([
      getRoutingPriorities(props.scope, controller.signal),
      props.keyId ? getKeyRoutingPreference(props.keyId, controller.signal) : Promise.resolve(null),
    ])
    if (current !== generation) return
    priorities.value = data
    allowed.value = state?.allow_user_override ?? data.allow_user_override ?? false
    availableGroups.value = state?.available_groups ?? data.available_groups ?? data.groups
    savedPreference.value = copy(state?.preference ?? null)
    draft.value = copy(savedPreference.value)
  } catch {
    if (current === generation) failed.value = true
  } finally {
    if (current === generation) loading.value = false
  }
}
function customize() {
  if (!allowed.value || props.disabled || !priorities.value) return
  updateDraft({ default_group_order: priorities.value.groups.map(group => group.id), model_rules: [] })
}
function updateDraft(value: RoutingPreference) {
  draft.value = copy(value)
  dirty.value = JSON.stringify(draft.value) !== JSON.stringify(savedPreference.value)
  validationError.value = ''
}
function reset() {
  draft.value = null
  dirty.value = savedPreference.value !== null
  validationError.value = ''
}
function validate(): boolean {
  if (!dirty.value || !draft.value) return true
  const models = draft.value.model_rules.map(rule => rule.model.trim())
  if (models.some(model => !model || model.length > 256 || /[\r\n\t]/.test(model) || (model.includes('*') && (!model.endsWith('*') || model.split('*').length !== 2))) || new Set(models).size !== models.length) {
    validationError.value = t('keys.routingPriority.invalidRules')
    return false
  }
  return true
}
async function save(id: number) {
  if (!dirty.value) return
  if (!validate()) throw new Error(validationError.value)
  try {
    await updateKeyRoutingPreference(id, copy(draft.value))
    savedPreference.value = copy(draft.value)
    dirty.value = false
  } catch (error) {
    validationError.value = t('keys.routingPriority.saveFailed')
    throw error
  }
}
defineExpose({ save, validate })
watch(() => [props.scope, props.keyId], load, { immediate: true })
onUnmounted(() => { generation++; controller?.abort() })
</script>

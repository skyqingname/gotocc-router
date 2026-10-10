<template>
  <div class="space-y-5" data-test="routing-preference-editor">
    <section class="space-y-2">
      <div class="flex items-center justify-between gap-2">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ t('keys.routingPriority.defaultOrder') }}</h4>
        <span class="text-xs text-gray-500 dark:text-dark-300">{{ t('keys.routingPriority.highFirst') }}</span>
      </div>
      <AutoGroupOrderEditor :model-value="modelValue.default_group_order" :groups="groups" :disabled="disabled" @update:model-value="updateOrder" />
    </section>
    <section class="space-y-3 border-t border-gray-200 pt-4 dark:border-dark-600">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ t('admin.smartRoutingPolicy.modelRules') }}</h4>
        <button type="button" class="btn btn-secondary min-h-11" :disabled="disabled || modelValue.model_rules.length >= 256" @click="addRule">
          <Icon name="plus" size="sm" class="mr-1" />{{ t('admin.smartRoutingPolicy.addRule') }}
        </button>
      </div>
      <p class="text-xs leading-relaxed text-gray-500 dark:text-dark-300">{{ t('keys.routingPriority.modelHint') }}</p>
      <div v-for="(rule, index) in modelValue.model_rules" :key="index" class="space-y-2 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
        <div class="flex items-end gap-2">
          <label class="min-w-0 flex-1 text-sm text-gray-700 dark:text-gray-200">
            <span class="mb-1 block">{{ t('admin.smartRoutingPolicy.model') }}</span>
            <input :value="rule.model" class="input w-full" maxlength="256" :disabled="disabled" :placeholder="t('admin.smartRoutingPolicy.modelPlaceholder')" @input="updateRule(index, { model: ($event.target as HTMLInputElement).value })" />
          </label>
          <button type="button" class="btn btn-secondary h-11 w-11 shrink-0 p-0" :disabled="disabled" :aria-label="t('admin.smartRoutingPolicy.removeRule')" @click="removeRule(index)"><Icon name="trash" size="sm" /></button>
        </div>
        <AutoGroupOrderEditor :model-value="rule.group_ids" :groups="groups" :disabled="disabled" @update:model-value="updateRule(index, { group_ids: $event })" />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { RoutingPriorityGroup } from '@/api/groups'
import type { RoutingPreference } from '@/api/keyRoutingPolicy'
import AutoGroupOrderEditor from '@/components/admin/group/AutoGroupOrderEditor.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ modelValue: RoutingPreference; groups: RoutingPriorityGroup[]; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: RoutingPreference] }>()
const { t } = useI18n()
function updateOrder(default_group_order: number[]) {
  emit('update:modelValue', { ...props.modelValue, default_group_order })
}
function updateRule(index: number, patch: Partial<RoutingPreference['model_rules'][number]>) {
  emit('update:modelValue', { ...props.modelValue, model_rules: props.modelValue.model_rules.map((rule, i) => i === index ? { ...rule, ...patch } : rule) })
}
function addRule() {
  if (props.disabled) return
  emit('update:modelValue', { ...props.modelValue, model_rules: [...props.modelValue.model_rules, { model: '', group_ids: [] }] })
}
function removeRule(index: number) {
  if (props.disabled) return
  emit('update:modelValue', { ...props.modelValue, model_rules: props.modelValue.model_rules.filter((_, i) => i !== index) })
}
</script>

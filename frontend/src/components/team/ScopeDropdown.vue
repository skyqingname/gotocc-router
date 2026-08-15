<template>
  <div class="inline-flex rounded-md border border-gray-200 bg-gray-50 p-1 dark:border-dark-700 dark:bg-dark-800" :aria-label="t('team.scopeSwitch')">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="inline-flex min-h-8 items-center gap-1.5 rounded px-3 text-sm font-medium transition-colors"
      :class="modelValue === option.value ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'"
      @click="select(option.value)"
    >
      <Icon :name="option.icon" size="sm" />
      {{ option.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

export type DataScope = 'personal' | 'team'
const props = defineProps<{ modelValue: DataScope }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: DataScope): void; (event: 'change', value: DataScope): void }>()
const { t } = useI18n()
const options = computed(() => [
  { value: 'personal' as const, label: t('team.personalKeys'), icon: 'user' as const },
  { value: 'team' as const, label: t('team.teamKeys'), icon: 'users' as const },
])
const select = (value: DataScope) => { if (value === props.modelValue) return; emit('update:modelValue', value); emit('change', value) }
</script>

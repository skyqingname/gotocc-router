<template>
  <section class="card" aria-labelledby="balance-cashback-title">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 id="balance-cashback-title" class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('balanceCashback.title') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('balanceCashback.description') }}</p>
    </div>
    <div class="space-y-5 px-6 py-5">
      <div class="flex items-center justify-between gap-4">
        <label class="font-medium text-gray-700 dark:text-gray-200">{{ t('balanceCashback.enabled') }}</label>
        <Toggle v-model="enabled" :aria-label="t('balanceCashback.enabled')" />
      </div>
      <div class="grid gap-5 sm:grid-cols-2">
        <label class="block">
          <span class="input-label">{{ t('balanceCashback.minimumAmount') }}</span>
          <input v-model.number="minimumAmount" type="number" min="0" step="any" required class="input w-full" :disabled="!enabled" />
          <span class="mt-2 block text-xs text-gray-500 dark:text-gray-400">{{ t('balanceCashback.minimumHint') }}</span>
        </label>
        <label class="block">
          <span class="input-label">{{ t('balanceCashback.percent') }}</span>
          <input v-model.number="percent" type="number" min="0" step="any" required class="input w-full" :disabled="!enabled" />
          <span class="mt-2 block text-xs text-gray-500 dark:text-gray-400">{{ t('balanceCashback.percentHint') }}</span>
        </label>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import type { BalanceCashbackSettings } from '@/utils/balanceCashback'

const props = defineProps<{ modelValue: BalanceCashbackSettings }>()
const emit = defineEmits<{ 'update:modelValue': [value: BalanceCashbackSettings] }>()
const { t } = useI18n()
const enabled = computed({ get: () => props.modelValue.enabled, set: enabled => emit('update:modelValue', { ...props.modelValue, enabled }) })
const minimumAmount = computed({ get: () => props.modelValue.minimum_amount, set: minimum_amount => emit('update:modelValue', { ...props.modelValue, minimum_amount }) })
const percent = computed({ get: () => props.modelValue.percent, set: percent => emit('update:modelValue', { ...props.modelValue, percent }) })
</script>

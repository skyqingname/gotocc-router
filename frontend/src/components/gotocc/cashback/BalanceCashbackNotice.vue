<template>
  <BaseDialog :show="amount > 0" :title="t('balanceCashback.congratulations')" width="narrow" @close="$emit('close')">
    <div class="flex items-start gap-3 py-2">
      <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400"><Icon name="gift" size="md" /></span>
      <p class="pt-1 text-base leading-7 text-gray-700 dark:text-gray-200">{{ t('balanceCashback.notice', { amount: formatCashbackAmount(amount, locale) }) }}</p>
    </div>
    <template #footer><button type="button" class="btn btn-primary" @click="$emit('close')">{{ t('common.confirm') }}</button></template>
  </BaseDialog>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatCashbackAmount } from '@/utils/balanceCashback'
defineProps<{ amount: number }>()
defineEmits<{ close: [] }>()
const { t, locale } = useI18n()
</script>

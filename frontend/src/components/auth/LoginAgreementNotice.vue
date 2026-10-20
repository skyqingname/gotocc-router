<template>
  <div
    role="status"
    class="rounded-lg border border-primary-100 bg-primary-50/70 p-3 text-sm text-primary-900 dark:border-primary-500/20 dark:bg-primary-500/10 dark:text-primary-100"
  >
    <div class="flex items-start gap-3">
      <Icon name="shield" size="sm" class="mt-0.5 flex-shrink-0 text-primary-600 dark:text-primary-300" />
      <div class="min-w-0 flex-1">
        <p class="font-medium">{{ t('legal.loginAgreementPrompt.noticeTitle') }}</p>
        <p class="mt-1 text-primary-700 dark:text-primary-200/80">
          {{ t('legal.loginAgreementPrompt.noticeDescription') }}
        </p>
        <button
          type="button"
          aria-haspopup="dialog"
          class="mt-3 rounded-md bg-primary-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-primary-700"
          @click="emit('open')"
        >
          {{ t('legal.loginAgreementPrompt.viewTerms') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const emit = defineEmits<{ open: [] }>()
</script>

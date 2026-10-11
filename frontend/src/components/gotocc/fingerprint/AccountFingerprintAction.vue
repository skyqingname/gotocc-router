<template>
  <button
    type="button"
    :disabled="!supportsFingerprintPlatform(account.platform)"
    :title="supportsFingerprintPlatform(account.platform) ? t('fingerprint.description') : t('fingerprint.unsupported')"
    class="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-40 dark:hover:bg-dark-700"
    @click="$emit('select', account)"
  >
    <Icon name="search" size="sm" class="text-sky-500" />
    {{ t('fingerprint.title') }}
  </button>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { Account } from '@/types'
import { supportsFingerprintPlatform } from './accountFingerprint'

defineProps<{ account: Account }>()
defineEmits<{ (event: 'select', account: Account): void }>()
const { t } = useI18n()
</script>

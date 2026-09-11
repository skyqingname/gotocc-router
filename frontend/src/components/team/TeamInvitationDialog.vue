<template>
  <BaseDialog :show="show" :title="t('team.inviteActionTitle')" width="narrow" @close="emit('close')">
    <div v-if="loading" class="flex min-h-48 items-center justify-center">
      <LoadingSpinner />
    </div>

    <div v-else-if="error" class="flex items-start gap-3 rounded-md bg-red-50 p-4 text-red-700 dark:bg-red-950/30 dark:text-red-300">
      <Icon name="exclamationTriangle" size="md" class="mt-0.5 shrink-0" />
      <p class="min-w-0 text-sm leading-6">{{ error }}</p>
    </div>

    <div v-else-if="preview">
      <p class="pb-4 text-sm leading-6 text-gray-500 dark:text-gray-400">
        {{ t('team.inviteActionDescription') }}
      </p>
      <dl class="divide-y divide-gray-100 border-y border-gray-200 dark:divide-dark-700 dark:border-dark-700">
        <div v-for="item in detailItems" :key="item.label" class="grid gap-1 py-3 sm:grid-cols-[7rem_minmax(0,1fr)] sm:gap-4">
          <dt class="text-sm text-gray-500 dark:text-gray-400">{{ item.label }}</dt>
          <dd class="min-w-0 break-words text-sm font-medium text-gray-900 sm:text-right dark:text-white">{{ item.value }}</dd>
        </div>
      </dl>
    </div>

    <template #footer>
      <div class="flex w-full flex-wrap justify-end gap-3">
        <button v-if="error" type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.close') }}</button>
        <template v-else>
          <button type="button" class="btn btn-secondary" :disabled="loading || resolving || !preview" @click="emit('resolve', 'declined')">
            <Icon name="x" size="sm" />
            {{ t('team.decline') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="loading || resolving || !preview" @click="emit('resolve', 'accepted')">
            <Icon name="check" size="sm" />
            {{ t('team.accept') }}
          </button>
        </template>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TeamInvitationPreview } from '@/api/team'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  show: boolean
  loading: boolean
  resolving: boolean
  preview: TeamInvitationPreview | null
  error: string
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'resolve', resolution: 'accepted' | 'declined'): void
}>()

const { t } = useI18n()
const detailItems = computed(() => props.preview ? [
  { label: t('team.invitedTeam'), value: props.preview.team_name },
  { label: t('team.inviter'), value: props.preview.inviter_name },
  { label: t('team.inviterEmail'), value: props.preview.inviter_email },
  { label: t('team.invitationExpiresAt'), value: formatDateTime(props.preview.expires_at) },
] : [])
</script>

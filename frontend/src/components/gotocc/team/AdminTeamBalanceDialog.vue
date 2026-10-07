<template>
  <BaseDialog :show="team !== null" :title="t('team.wallet.editBalance')" width="narrow" @close="close">
    <form v-if="team" id="admin-team-balance" class="space-y-5" @submit.prevent="save">
      <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
        <p class="font-medium text-gray-900 dark:text-white">{{ team.name }}</p>
        <p class="mt-2 text-sm text-gray-600 dark:text-dark-300">{{ t('team.wallet.currentBalance', { amount: formatTeamCost(team.balance) }) }}</p>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('team.wallet.frozen', { amount: formatTeamCost(team.frozen_balance) }) }}</p>
      </div>
      <div>
        <label for="admin-team-new-balance" class="input-label">{{ t('team.wallet.newBalance') }}</label>
        <input id="admin-team-new-balance" v-model.number="balance" class="input" type="number" min="0" step="any" required :disabled="saving" />
        <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('team.wallet.adminBalanceHint') }}</p>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" :disabled="saving" @click="close">{{ t('common.cancel') }}</button>
        <button form="admin-team-balance" type="submit" class="btn btn-primary" :disabled="saving || !validBalance">{{ t('common.save') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import teamsAPI, { type AdminTeam } from '@/api/admin/teams'
import { useAppStore } from '@/stores/app'
import { formatTeamCost } from './teamFormat'

const props = defineProps<{ team: AdminTeam | null }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const balance = ref<number | ''>('')
const saving = ref(false)
const pending = ref<{ operationID: string; balance: number } | null>(null)
watch(() => props.team, team => { balance.value = team?.balance ?? ''; pending.value = null })
const validBalance = computed(() => typeof balance.value === 'number' && Number.isFinite(balance.value) && balance.value >= 0)
const close = () => { if (!saving.value) emit('close') }
const save = async () => {
  if (!props.team || saving.value || !validBalance.value || typeof balance.value !== 'number') return
  saving.value = true
  if (pending.value?.balance !== balance.value) pending.value = { operationID: crypto.randomUUID(), balance: balance.value }
  try {
    await teamsAPI.setBalance(props.team.id, pending.value.operationID, pending.value.balance)
    appStore.showSuccess(t('team.wallet.balanceUpdated'))
    emit('saved')
    emit('close')
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  } finally {
    saving.value = false
  }
}
</script>

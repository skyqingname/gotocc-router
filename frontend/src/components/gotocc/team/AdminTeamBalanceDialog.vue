<template>
  <BaseDialog :show="team !== null" :title="t('team.wallet.editBalance')" width="narrow" @close="close">
    <form v-if="team" id="admin-team-balance" class="space-y-5" @submit.prevent="save">
      <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
        <p class="font-medium text-gray-900 dark:text-white">{{ team.name }}</p>
        <p class="mt-2 text-sm text-gray-600 dark:text-dark-300">{{ t('team.wallet.currentBalance', { amount: formatTeamCost(team.balance) }) }}</p>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('team.wallet.frozen', { amount: formatTeamCost(team.frozen_balance) }) }}</p>
      </div>
      <div>
        <p class="input-label">{{ t('team.wallet.operation') }}</p>
        <div v-segmented class="segmented grid grid-cols-3 gap-1" role="radiogroup" :aria-label="t('team.wallet.operation')">
          <button
            v-for="option in operations"
            :key="option.value"
            type="button"
            role="radio"
            :aria-checked="operation === option.value"
            :disabled="saving"
            :class="['segmented-item py-1.5 text-sm', operation === option.value && 'segmented-item-active']"
            @click="operation = option.value"
          >
            {{ option.label }}
          </button>
        </div>
      </div>
      <div>
        <label for="admin-team-balance-amount" class="input-label">{{ t('team.wallet.adjustAmount') }}</label>
        <div class="flex gap-2">
          <input id="admin-team-balance-amount" v-model.number="amount" class="input" type="number" min="0" step="any" required :disabled="saving" />
          <button
            v-if="operation === 'subtract'"
            type="button"
            class="btn btn-secondary whitespace-nowrap"
            :disabled="saving"
            @click="amount = team.balance"
          >
            {{ t('team.wallet.subtractAll') }}
          </button>
        </div>
        <p v-if="exceedsBalance" class="mt-2 text-xs text-red-600 dark:text-red-400">{{ t('team.wallet.subtractExceeds') }}</p>
        <p v-else-if="validAmount" class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
          {{ t('team.wallet.resultBalance', { amount: formatTeamCost(resultBalance) }) }}
        </p>
        <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('team.wallet.adminBalanceHint') }}</p>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" :disabled="saving" @click="close">{{ t('common.cancel') }}</button>
        <button form="admin-team-balance" type="submit" class="btn btn-primary" :disabled="saving || !validAmount || exceedsBalance">{{ t('common.save') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import teamsAPI, { type AdminTeam, type TeamBalanceOperation } from '@/api/admin/teams'
import { useAppStore } from '@/stores/app'
import { vSegmented } from '../vSegmented'
import { formatTeamCost } from './teamFormat'

const props = defineProps<{ team: AdminTeam | null }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const operations = computed<Array<{ value: TeamBalanceOperation; label: string }>>(() => [
  { value: 'add', label: t('team.wallet.operationAdd') },
  { value: 'subtract', label: t('team.wallet.operationSubtract') },
  { value: 'set', label: t('team.wallet.operationSet') },
])
const operation = ref<TeamBalanceOperation>('add')
const amount = ref<number | ''>('')
const saving = ref(false)
// 同一笔调整在重试时沿用单号；方式或金额变化后视为新的一笔。
const pending = ref<{ operationID: string; operation: TeamBalanceOperation; amount: number } | null>(null)
watch(() => props.team, () => { operation.value = 'add'; amount.value = ''; pending.value = null })
watch(operation, () => { amount.value = '' })
const validAmount = computed(() => typeof amount.value === 'number' && Number.isFinite(amount.value) &&
  (operation.value === 'set' ? amount.value >= 0 : amount.value > 0))
const resultBalance = computed(() => {
  const value = typeof amount.value === 'number' ? amount.value : 0
  const current = props.team?.balance ?? 0
  if (operation.value === 'add') return current + value
  if (operation.value === 'subtract') return current - value
  return value
})
const exceedsBalance = computed(() => operation.value === 'subtract' && validAmount.value && resultBalance.value < 0)
const close = () => { if (!saving.value) emit('close') }
const save = async () => {
  if (!props.team || saving.value || !validAmount.value || exceedsBalance.value || typeof amount.value !== 'number') return
  saving.value = true
  if (pending.value?.operation !== operation.value || pending.value?.amount !== amount.value) {
    pending.value = { operationID: crypto.randomUUID(), operation: operation.value, amount: amount.value }
  }
  try {
    await teamsAPI.adjustBalance(props.team.id, pending.value.operationID, pending.value.operation, pending.value.amount)
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

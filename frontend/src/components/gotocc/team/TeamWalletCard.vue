<template>
  <section class="rounded-2xl border border-primary-200 bg-primary-50/70 p-5 dark:border-primary-500/30 dark:bg-primary-500/10" :aria-label="t('team.wallet.title')">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <p class="text-sm text-primary-700 dark:text-primary-300">{{ t('team.wallet.shared') }}</p>
        <p class="mt-2 text-3xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatTeamCost(context.team.balance) }}</p>
      </div>
      <button v-if="context.membership.role === 'owner' && !adminSupportContext" class="btn btn-primary btn-sm" :disabled="context.team.status !== 'active'" @click="openFunding">
        <Icon name="plus" size="sm" />{{ t('team.wallet.fund') }}
      </button>
    </div>
    <p class="mt-2 text-xs tabular-nums text-gray-600 dark:text-dark-300">{{ t('team.wallet.frozen', { amount: formatTeamCost(context.team.frozen_balance) }) }}</p>
    <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('team.wallet.sharedHint') }}</p>
  </section>

  <BaseDialog :show="showFunding" :title="t('team.wallet.fund')" width="narrow" @close="closeFunding">
    <form id="team-wallet-fund" class="space-y-4" @submit.prevent="fund">
      <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('team.wallet.fundDescription') }}</p>
      <div>
        <label for="team-fund-amount" class="input-label">{{ t('team.wallet.amount') }}</label>
        <input id="team-fund-amount" v-model.number="amount" class="input" type="number" min="0" step="any" required :disabled="funding" />
      </div>
      <p v-if="userView.user" class="text-xs text-gray-500 dark:text-dark-400">{{ t('team.wallet.available', { amount: formatTeamCost(userView.user.balance) }) }}</p>
      <p class="rounded-lg bg-primary-50 p-3 text-sm leading-6 text-primary-800 dark:bg-primary-500/10 dark:text-primary-200">{{ t('team.wallet.irreversible') }}</p>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" :disabled="funding" @click="closeFunding">{{ t('common.cancel') }}</button>
        <button form="team-wallet-fund" type="submit" class="btn btn-primary" :disabled="funding || !validAmount">{{ t('team.wallet.confirm') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { teamAPI, type TeamContext, type TeamFundingResult } from '@/api/team'
import { useUserView } from '@/composables/useUserView'
import { useAppStore } from '@/stores/app'
import { adminSupportContext } from '@/utils/adminSupportContext'
import { formatTeamCost } from './teamFormat'

defineProps<{ context: TeamContext }>()
const emit = defineEmits<{ funded: [result: TeamFundingResult] }>()
const { t } = useI18n()
const userView = useUserView()
const appStore = useAppStore()
const showFunding = ref(false)
const funding = ref(false)
const amount = ref<number | ''>('')
const pending = ref<{ operationID: string; amount: number } | null>(null)
const validAmount = computed(() => typeof amount.value === 'number' && Number.isFinite(amount.value) && amount.value > 0 && userView.user != null && amount.value <= userView.user.balance)
const openFunding = () => { amount.value = ''; pending.value = null; showFunding.value = true }
const closeFunding = () => { if (!funding.value) showFunding.value = false }
const fund = async () => {
  if (!validAmount.value || funding.value || typeof amount.value !== 'number') return
  funding.value = true
  if (pending.value?.amount !== amount.value) pending.value = { operationID: crypto.randomUUID(), amount: amount.value }
  try {
    const result = await teamAPI.fundWallet(pending.value.operationID, pending.value.amount)
    if (userView.user) userView.user = { ...userView.user, balance: result.personal_balance }
    emit('funded', result)
    showFunding.value = false
    pending.value = null
    appStore.showSuccess(t('team.wallet.funded'))
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  } finally {
    funding.value = false
  }
}
</script>

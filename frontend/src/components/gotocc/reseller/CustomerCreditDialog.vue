<template>
  <BaseDialog :show="!!customer" :title="tr('调整客户额度', 'Manage customer credits')" width="normal" @close="emit('close')">
    <form id="customer-credit" class="space-y-5" @submit.prevent="save">
      <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800"><p class="font-medium">{{ customer?.username || customer?.email }}</p><p class="mt-1 text-sm text-gray-500">{{ tr('当前可用额度', 'Available credits') }} <span class="ml-2 font-semibold tabular-nums">{{ customer?.credit_balance }}</span></p></div>
      <label class="block text-sm font-medium">{{ tr('操作类型', 'Operation') }}<Select v-model="input.kind" class="mt-2" :searchable="false" :options="kinds" /></label>
      <label class="block text-sm font-medium">{{ tr('额度数量', 'Credit amount') }}<input v-model.number="input.amount" type="number" min="0" step="any" required class="input mt-2" /></label>
      <p class="text-sm leading-6 text-gray-500">{{ input.kind === 'purchase' ? tr('确认已由你收款后发放购买额度。平台不参与客户收款。', 'Issue purchased credits after you receive payment. The platform does not collect customer payments.') : input.kind === 'gift' ? tr('赠送额度不会立即扣除你的平台余额，实际使用时按你的价格结算成本。', 'Gifting credits does not debit your platform balance. Service costs are charged when the customer uses them.') : tr('只扣减客户当前可用额度，不改变已冻结的任务额度。现金退款由你处理。', 'Deduct available customer credits. Reserved task credits remain unchanged. You handle any cash refund.') }}</p>
      <label class="block text-sm font-medium">{{ tr('备注', 'Note') }}<textarea v-model="input.notes" rows="2" class="input mt-2" /></label>
      <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
    </form>
    <template #footer><button class="btn btn-secondary" :disabled="saving" @click="emit('close')">{{ tr('取消', 'Cancel') }}</button><button class="btn btn-primary" type="submit" form="customer-credit" :disabled="saving || !input.amount || input.amount <= 0">{{ saving ? tr('提交中…', 'Saving…') : tr('确认调整', 'Apply change') }}</button></template>
  </BaseDialog>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import { resellerAPI, type ResellerCustomer, type ResellerCreditInput } from '@/api/reseller'
import { extractApiErrorMessage } from '@/utils/apiError'
const props = defineProps<{ customer: ResellerCustomer | null }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { locale } = useI18n()
const tr = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const kinds = computed(() => [{ value: 'purchase', label: tr('购买额度', 'Purchased credits') }, { value: 'gift', label: tr('赠送福利', 'Gift credits') }, { value: 'deduct', label: tr('扣减额度', 'Deduct credits') }])
const input = reactive<ResellerCreditInput>({ operation_id: '', kind: 'purchase', amount: 0, notes: '' })
const saving = ref(false)
const error = ref('')
watch(() => props.customer, () => { Object.assign(input, { operation_id: crypto.randomUUID(), kind: 'purchase', amount: 0, notes: '' }); error.value = '' })
async function save() {
  if (!props.customer) return
  saving.value = true; error.value = ''
  try { await resellerAPI.changeCredit(props.customer.user_id, { ...input }); emit('saved'); emit('close') }
  catch (e) { error.value = extractApiErrorMessage(e, tr('调整失败', 'Could not update credits')) }
  finally { saving.value = false }
}
</script>

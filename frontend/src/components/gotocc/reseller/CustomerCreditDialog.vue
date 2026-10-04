<template>
  <BaseDialog :show="!!customer" :title="tr('调整客户额度', 'Manage customer credits')" width="normal" @close="emit('close')">
    <form id="customer-credit" class="space-y-5" @submit.prevent="save">
      <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800"><p class="font-medium">{{ customer?.username || customer?.email }}</p><p class="mt-1 text-sm text-gray-500">{{ tr('当前可用额度', 'Available credits') }} <span class="ml-2 font-semibold tabular-nums">{{ customer?.credit_balance }}</span></p></div>
      <div class="grid items-end gap-4" :class="input.kind === 'deduct' ? 'grid-cols-[minmax(0,1fr)_auto]' : 'grid-cols-1'">
        <label class="block text-sm font-medium">{{ tr('操作类型', 'Operation') }}<Select v-model="input.kind" class="mt-2" :searchable="false" :options="kinds" /></label>
        <label v-if="input.kind === 'deduct'" class="flex min-h-10 cursor-pointer items-center gap-2 whitespace-nowrap text-sm"><input v-model="input.deduct_all" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />{{ tr('扣除全部额度', 'Deduct all credits') }}</label>
      </div>
      <label class="block text-sm font-medium">{{ tr('额度数量', 'Credit amount') }}<input v-model.number="input.amount" type="number" min="0" step="any" :required="!input.deduct_all" :disabled="input.deduct_all" class="input mt-2" /></label>
      <label class="block text-sm font-medium">{{ tr('备注', 'Note') }}<textarea v-model="input.notes" rows="2" class="input mt-2" /></label>
      <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
    </form>
    <template #footer><button class="btn btn-secondary" :disabled="saving" @click="emit('close')">{{ tr('取消', 'Cancel') }}</button><button class="btn btn-primary" type="submit" form="customer-credit" :disabled="saving || !canSubmit">{{ saving ? tr('提交中…', 'Saving…') : tr('确认调整', 'Apply change') }}</button></template>
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
const kinds = computed(() => [{ value: 'increase', label: tr('增加', 'Increase') }, { value: 'deduct', label: tr('减少', 'Decrease') }])
const input = reactive<ResellerCreditInput>({ operation_id: '', kind: 'increase', amount: 0, deduct_all: false, notes: '' })
const canSubmit = computed(() => input.deduct_all ? (props.customer?.credit_balance ?? 0) > 0 : input.amount > 0)
const saving = ref(false)
const error = ref('')
watch(() => props.customer, () => { Object.assign(input, { operation_id: crypto.randomUUID(), kind: 'increase', amount: 0, deduct_all: false, notes: '' }); error.value = '' })
watch(() => input.kind, kind => { if (kind !== 'deduct') input.deduct_all = false })
watch(() => input.deduct_all, all => { if (all && props.customer) input.amount = props.customer.credit_balance })
async function save() {
  if (!props.customer) return
  saving.value = true; error.value = ''
  try { await resellerAPI.changeCredit(props.customer.user_id, { ...input, amount: input.deduct_all ? 0 : input.amount }); emit('saved'); emit('close') }
  catch (e) { error.value = extractApiErrorMessage(e, tr('调整失败', 'Could not update credits')) }
  finally { saving.value = false }
}
</script>

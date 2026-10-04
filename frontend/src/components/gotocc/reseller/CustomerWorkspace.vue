<template>
  <BaseDialog :show="show" :title="customer ? `${tr('客户管理', 'Customer')} · ${customer.username || customer.email}` : tr('创建客户', 'Create customer')" width="wide" @close="emit('close')">
    <div v-if="loading" class="py-12 text-center text-sm text-gray-500">{{ tr('正在读取客户资料…', 'Loading customer…') }}</div>
    <template v-else>
      <div v-if="customer" class="mb-5 flex flex-wrap gap-2 border-b border-gray-200 pb-3 dark:border-dark-700">
        <button v-for="item in tabs" :key="item.id" class="rounded-lg px-4 py-2 text-sm" :class="tab === item.id ? 'bg-primary-50 font-semibold text-primary-700 dark:bg-primary-950 dark:text-primary-300' : 'text-gray-500 hover:bg-gray-50 dark:hover:bg-dark-800'" @click="changeTab(item.id)">{{ tr(item.zh, item.en) }}</button>
      </div>
      <p v-if="error" role="alert" class="mb-4 text-sm text-red-600">{{ error }}</p>
      <form v-if="tab === 'profile'" id="reseller-customer-profile" class="space-y-5" @submit.prevent="saveProfile">
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="text-sm font-medium">{{ tr('邮箱', 'Email') }}<input v-model="form.email" type="email" required autocomplete="off" class="input mt-2" /></label>
          <label class="text-sm font-medium">{{ tr('用户名', 'Username') }}<input v-model="form.username" class="input mt-2" autocomplete="off" /></label>
          <label class="text-sm font-medium">{{ tr('状态', 'Status') }}<Select v-model="form.status" class="mt-2" :searchable="false" :options="statusOptions" /></label>
          <label class="text-sm font-medium">{{ customer ? tr('重设密码（留空保留）', 'Reset password (leave blank to keep)') : tr('登录密码', 'Password') }}<input v-model="form.password" type="password" :required="!customer" autocomplete="new-password" class="input mt-2" /></label>
          <label class="text-sm font-medium">{{ tr('并发数', 'Concurrency') }}<input v-model.number="form.concurrency" type="number" min="1" :max="limits.concurrency" required class="input mt-2" /><span class="mt-1 block text-xs font-normal text-gray-500">{{ tr('你的可用上限', 'Your limit') }}: {{ limits.concurrency }}</span></label>
          <label class="text-sm font-medium">{{ tr('每分钟请求数', 'Requests per minute') }}<input v-model.number="form.rpm_limit" type="number" :min="limits.rpm_limit ? 1 : 0" :max="limits.rpm_limit || undefined" required class="input mt-2" /><span class="mt-1 block text-xs font-normal text-gray-500">{{ limits.rpm_limit ? `${tr('你的可用上限', 'Your limit')}: ${limits.rpm_limit}` : tr('0 表示不单独限制', '0 means no individual limit') }}</span></label>
        </div>
        <div>
          <p class="mb-2 text-sm font-medium">{{ tr('可用分组', 'Available groups') }}</p>
          <details class="relative rounded-lg border border-gray-200 dark:border-dark-600">
            <summary class="cursor-pointer px-3 py-2.5 text-sm">{{ tr('选择分组', 'Select groups') }} · {{ form.allowed_groups.length }} {{ tr('个已选', 'selected') }}</summary>
            <div class="space-y-2 border-t border-gray-200 p-3 dark:border-dark-600">
              <input v-model="groupSearch" class="input" :placeholder="tr('搜索分组', 'Search groups')" />
              <div class="max-h-44 space-y-1 overflow-y-auto">
                <label v-for="group in filteredGroups" :key="group.id" class="flex cursor-pointer items-center gap-3 rounded-lg px-2 py-2 text-sm hover:bg-gray-50 dark:hover:bg-dark-800"><input v-model="form.allowed_groups" type="checkbox" :value="group.id" class="h-4 w-4 rounded border-gray-300 text-primary-600" /><span class="flex-1">{{ group.name }}</span><span class="text-xs text-gray-500">{{ group.platform }}</span></label>
                <p v-if="!filteredGroups.length" class="p-3 text-center text-sm text-gray-500">{{ tr('没有可用分组', 'No groups available') }}</p>
              </div>
            </div>
          </details>
          <p class="mt-2 text-xs text-gray-500">{{ tr('客户只能使用选中的分组，服务费用由你的平台余额承担。', 'The customer can use the selected groups. Service costs are charged to your platform balance.') }}</p>
        </div>
        <label class="block text-sm font-medium">{{ tr('客户备注', 'Customer notes') }}<textarea v-model="form.notes" rows="2" class="input mt-2" /></label>
      </form>
      <template v-else-if="tab === 'credits'">
        <CreditLedger :entries="credits?.items ?? []" owner />
        <div class="mt-4 flex justify-between text-sm"><span class="text-gray-500">{{ tr('共', 'Total') }} {{ credits?.total ?? 0 }}</span><div class="flex gap-2"><button class="btn btn-secondary btn-sm" :disabled="page <= 1 || tabLoading" @click="page--; loadTab()">{{ tr('上一页', 'Previous') }}</button><button class="btn btn-secondary btn-sm" :disabled="!credits || page * credits.page_size >= credits.total || tabLoading" @click="page++; loadTab()">{{ tr('下一页', 'Next') }}</button></div></div>
      </template>
      <template v-else-if="tab === 'keys'">
        <form class="mb-5 grid gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-800 sm:grid-cols-2" @submit.prevent="createKey">
          <label class="text-sm font-medium">{{ tr('Key 名称', 'Key name') }}<input v-model="newKey.name" required class="input mt-2" /></label>
          <label class="text-sm font-medium">{{ tr('使用分组', 'Group') }}<Select v-model="newKey.group_id" class="mt-2" :searchable="false" :options="keyGroups" :placeholder="tr('选择分组', 'Choose a group')" /></label>
          <label class="text-sm font-medium">{{ tr('Key 累计额度（0 不限）', 'Key budget (0 = unlimited)') }}<input v-model.number="newKey.quota" type="number" min="0" step="any" required class="input mt-2" /></label>
          <button class="btn btn-primary self-end" :disabled="saving || newKey.group_id === null">{{ tr('创建 Key', 'Create key') }}</button>
        </form>
        <div v-if="createdKey" class="mb-5 rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-800 dark:bg-primary-950"><p class="mb-2 text-sm font-medium">{{ tr('新 Key，请交给客户保存', 'New key — share it with your customer') }}</p><input :value="createdKey" readonly class="input font-mono text-xs" /><button class="btn btn-secondary btn-sm mt-3" @click="copyKey">{{ tr('复制', 'Copy') }}</button></div>
        <div class="overflow-x-auto"><table class="w-full text-left text-sm"><thead class="border-b border-gray-200 text-xs text-gray-500 dark:border-dark-700"><tr><th class="p-3">{{ tr('名称 / 状态', 'Name / status') }}</th><th class="p-3">{{ tr('已用额度', 'Used') }}</th><th class="p-3">{{ tr('额度上限', 'Budget') }}</th><th class="p-3">{{ tr('操作', 'Actions') }}</th></tr></thead><tbody class="divide-y divide-gray-100 dark:divide-dark-700"><tr v-for="key in keys?.items" :key="key.id"><td class="p-3"><p class="font-medium">{{ key.name }}</p><p class="mt-1 text-xs text-gray-500">{{ key.status }}</p></td><td class="p-3 tabular-nums">{{ key.quota_used }}</td><td class="p-3"><input v-model.number="keyQuotas[key.id]" type="number" min="0" step="any" class="input w-28" :aria-label="`${key.name} ${tr('额度上限', 'budget')}`" /></td><td class="p-3"><div class="flex gap-3 whitespace-nowrap"><button class="text-primary-600" :disabled="saving" @click="saveKeyQuota(key)">{{ tr('保存额度', 'Save budget') }}</button><button class="text-primary-600" :disabled="saving" @click="toggleKey(key)">{{ key.status === 'disabled' ? tr('启用', 'Enable') : tr('停用', 'Disable') }}</button></div></td></tr><tr v-if="!keys?.items.length"><td colspan="4" class="py-10 text-center text-gray-500">{{ tr('客户还没有 Key', 'No API keys yet') }}</td></tr></tbody></table></div>
        <div class="mt-4 flex justify-end gap-2"><button class="btn btn-secondary btn-sm" :disabled="page <= 1 || tabLoading" @click="page--; loadTab()">{{ tr('上一页', 'Previous') }}</button><button class="btn btn-secondary btn-sm" :disabled="!keys || page * keys.page_size >= keys.total || tabLoading" @click="page++; loadTab()">{{ tr('下一页', 'Next') }}</button></div>
      </template>
      <template v-else>
        <div class="overflow-x-auto"><table class="w-full text-left text-sm"><thead class="border-b border-gray-200 text-xs text-gray-500 dark:border-dark-700"><tr><th class="p-3">{{ tr('时间', 'Time') }}</th><th class="p-3">{{ tr('模型', 'Model') }}</th><th class="p-3">{{ tr('Token', 'Tokens') }}</th><th class="p-3">{{ tr('客户消费额度', 'Customer charge') }}</th></tr></thead><tbody class="divide-y divide-gray-100 dark:divide-dark-700"><tr v-for="entry in usage?.items" :key="entry.id"><td class="whitespace-nowrap p-3 text-xs">{{ new Date(entry.created_at).toLocaleString(locale) }}</td><td class="p-3">{{ entry.model }}</td><td class="p-3 tabular-nums">{{ entry.input_tokens + entry.output_tokens }}</td><td class="p-3 tabular-nums">{{ entry.actual_cost }}</td></tr><tr v-if="!usage?.items.length"><td colspan="4" class="py-10 text-center text-gray-500">{{ tr('暂无使用记录', 'No usage yet') }}</td></tr></tbody></table></div>
        <div class="mt-4 flex justify-end gap-2"><button class="btn btn-secondary btn-sm" :disabled="page <= 1 || tabLoading" @click="page--; loadTab()">{{ tr('上一页', 'Previous') }}</button><button class="btn btn-secondary btn-sm" :disabled="!usage || page * usage.page_size >= usage.total || tabLoading" @click="page++; loadTab()">{{ tr('下一页', 'Next') }}</button></div>
      </template>
      <p v-if="tabLoading" class="mt-3 text-sm text-gray-500">{{ tr('正在读取…', 'Loading…') }}</p>
    </template>
    <template #footer><button class="btn btn-secondary" @click="emit('close')">{{ tr('关闭', 'Close') }}</button><button v-if="tab === 'profile' && !loading" type="submit" form="reseller-customer-profile" class="btn btn-primary" :disabled="saving">{{ saving ? tr('保存中…', 'Saving…') : tr('保存客户', 'Save customer') }}</button></template>
  </BaseDialog>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import CreditLedger from './CreditLedger.vue'
import { resellerAPI, type ResellerCustomer, type ResellerCustomerInput, type ResellerGroup, type ResellerOwnerLimits, type ResellerPage, type ResellerCreditEntry, type ResellerKeyInput } from '@/api/reseller'
import type { ApiKey, UsageLog } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
const props = defineProps<{ show: boolean; customer: ResellerCustomer | null; groups: ResellerGroup[]; limits: ResellerOwnerLimits }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { locale } = useI18n()
const tr = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const tabs = [{ id: 'profile', zh: '资料与权限', en: 'Profile & access' }, { id: 'keys', zh: 'API Key', en: 'API keys' }, { id: 'credits', zh: '额度流水', en: 'Credits' }, { id: 'usage', zh: '使用记录', en: 'Usage' }]
const tab = ref('profile'); const loading = ref(false); const saving = ref(false); const tabLoading = ref(false); const error = ref(''); const page = ref(1)
const form = reactive<ResellerCustomerInput>({ email: '', username: '', password: '', status: 'active', concurrency: props.limits.concurrency, rpm_limit: props.limits.rpm_limit, allowed_groups: [], notes: '' })
const groupSearch = ref('')
const filteredGroups = computed(() => props.groups.filter(group => group.name.toLowerCase().includes(groupSearch.value.toLowerCase())))
const statusOptions = computed(() => [{ value: 'active', label: tr('启用', 'Active') }, { value: 'disabled', label: tr('停用', 'Disabled') }])
const credits = ref<ResellerPage<ResellerCreditEntry>>(); const keys = ref<ResellerPage<ApiKey>>(); const usage = ref<ResellerPage<UsageLog>>()
const keyQuotas = reactive<Record<number, number>>({})
const newKey = reactive<ResellerKeyInput>({ name: '', group_id: null, routing_mode: 'fixed', quota: 0 })
const createdKey = ref('')
const keyGroups = computed(() => props.groups.filter(group => form.allowed_groups.includes(group.id)).map(group => ({ value: group.id, label: group.name })))
watch(() => props.show, async show => {
  if (!show) { createdKey.value = ''; return }
  tab.value = 'profile'; page.value = 1; error.value = ''; groupSearch.value = ''; createdKey.value = ''; keys.value = undefined; credits.value = undefined; usage.value = undefined
  Object.assign(form, { email: '', username: '', password: '', status: 'active', concurrency: props.limits.concurrency, rpm_limit: props.limits.rpm_limit, allowed_groups: props.groups.map(group => group.id), notes: '' })
  Object.assign(newKey, { name: '', group_id: null, routing_mode: 'fixed', quota: 0 })
  if (!props.customer) return
  loading.value = true
  try {
    const user = await resellerAPI.customer(props.customer.user_id)
    Object.assign(form, { email: user.email, username: user.username, status: user.status, concurrency: user.concurrency, rpm_limit: user.rpm_limit ?? 0, allowed_groups: user.allowed_groups ?? [], notes: props.customer.notes })
  } catch (e) { error.value = extractApiErrorMessage(e, tr('读取客户失败', 'Could not load customer')) }
  finally { loading.value = false }
})
async function saveProfile() {
  saving.value = true; error.value = ''
  try {
    if (props.customer) await resellerAPI.updateCustomer(props.customer.user_id, { ...form })
    else await resellerAPI.createCustomer({ ...form })
    emit('saved'); emit('close')
  } catch (e) { error.value = extractApiErrorMessage(e, tr('保存客户失败', 'Could not save customer')) }
  finally { saving.value = false }
}
async function changeTab(value: string) { tab.value = value; page.value = 1; await loadTab() }
async function loadTab() {
  if (!props.customer || tab.value === 'profile') return
  tabLoading.value = true; error.value = ''
  try {
    if (tab.value === 'credits') credits.value = await resellerAPI.creditEntries(props.customer.user_id, page.value)
    else if (tab.value === 'keys') { keys.value = await resellerAPI.customerKeys(props.customer.user_id, page.value); for (const key of keys.value.items) keyQuotas[key.id] = key.quota }
    else usage.value = await resellerAPI.customerUsage(props.customer.user_id, page.value)
  } catch (e) { error.value = extractApiErrorMessage(e, tr('读取失败', 'Could not load records')) }
  finally { tabLoading.value = false }
}
async function createKey() {
  if (!props.customer) return
  saving.value = true; error.value = ''
  try { createdKey.value = (await resellerAPI.createCustomerKey(props.customer.user_id, { ...newKey })).key; await loadTab() }
  catch (e) { error.value = extractApiErrorMessage(e, tr('创建 Key 失败', 'Could not create key')) }
  finally { saving.value = false }
}
async function updateKey(key: ApiKey, input: { status?: string; quota?: number }) {
  if (!props.customer) return
  saving.value = true; error.value = ''
  try { await resellerAPI.updateCustomerKey(props.customer.user_id, key.id, input); await loadTab() }
  catch (e) { error.value = extractApiErrorMessage(e, tr('修改 Key 失败', 'Could not update key')) }
  finally { saving.value = false }
}
const toggleKey = (key: ApiKey) => updateKey(key, { status: key.status === 'disabled' ? 'active' : 'disabled' })
const saveKeyQuota = (key: ApiKey) => updateKey(key, { quota: keyQuotas[key.id] })
async function copyKey() { try { await navigator.clipboard.writeText(createdKey.value) } catch { error.value = tr('请手动复制新 Key', 'Please copy the new key manually') } }
</script>

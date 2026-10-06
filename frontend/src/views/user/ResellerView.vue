<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="card py-16 text-center text-sm text-gray-500">{{ tr('正在读取站长中心…', 'Loading reseller center…') }}</div>
      <div v-else-if="error && !overview" class="card p-6"><p role="alert" class="text-red-600">{{ error }}</p><button class="btn btn-secondary mt-4" @click="load">{{ tr('重试', 'Retry') }}</button></div>
      <template v-else-if="overview">
        <div class="flex flex-wrap items-end justify-between gap-4">
          <div><h1 class="text-2xl font-semibold tracking-tight">{{ tr('站长中心', 'Reseller center') }}</h1><p class="mt-2 text-sm text-gray-500">{{ tr('管理你的客户、额度与服务成本。客户付款由你收取。', 'Manage your customers, credits and service costs. Customer payments go to you.') }}</p></div>
          <div class="flex flex-wrap items-end gap-3">
            <form class="flex items-end gap-2" @submit.prevent="saveInitialCredit">
              <label class="block text-xs font-medium text-gray-500">{{ tr('初始额度', 'Initial credits') }}<input v-model.number="initialCredit" type="number" min="0" step="any" required :aria-label="tr('新用户初始额度', 'New customer initial credits')" class="input mt-1 w-32" /></label>
              <button type="submit" class="btn btn-secondary" :disabled="initialSaving || initialCredit === overview.profile.initial_credit">{{ initialSaving ? tr('保存中…', 'Saving…') : tr('保存', 'Save') }}</button>
            </form>
            <button class="btn btn-primary" @click="openCustomer(null)">{{ tr('创建客户', 'Create customer') }}</button>
          </div>
        </div>
        <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div class="card border-primary-200 bg-primary-50/40 p-5 dark:border-primary-800 dark:bg-primary-950/20"><p class="text-xs font-medium text-primary-700 dark:text-primary-300">{{ tr('你的平台余额', 'Your platform balance') }}</p><p class="mt-3 break-all text-2xl font-semibold tabular-nums xl:text-xl 2xl:text-2xl">{{ money(overview.limits.balance) }}</p><p class="mt-2 text-xs text-gray-500">{{ tr('承担本人和客户的实际服务成本', 'Pays for your own and customer service usage') }}</p></div>
          <div class="card p-5"><p class="text-xs font-medium text-gray-500">{{ tr('所属客户', 'Your customers') }}</p><p class="mt-3 break-all text-2xl font-semibold tabular-nums xl:text-xl 2xl:text-2xl">{{ overview.summary.customer_count }}</p><p class="mt-2 text-xs text-gray-500">{{ tr('账户、权限与额度由你管理', 'You manage their accounts, access and credits') }}</p></div>
          <div class="card p-5"><p class="text-xs font-medium text-gray-500">{{ tr('客户剩余可用额度', 'Available customer credits') }}</p><p class="mt-3 break-all text-2xl font-semibold tabular-nums xl:text-xl 2xl:text-2xl">{{ money(overview.summary.credit_balance) }}</p><p class="mt-2 text-xs text-gray-500">{{ tr('发放时不扣你的平台余额', 'Issuing credits does not debit your platform balance') }}</p></div>
          <div class="card p-5"><p class="text-xs font-medium text-gray-500">{{ tr('累计客户服务成本', 'Customer service costs') }}</p><p class="mt-3 break-all text-2xl font-semibold tabular-nums xl:text-xl 2xl:text-2xl">{{ money(overview.summary.cost) }}</p><p class="mt-2 text-xs text-gray-500">{{ tr('新结算模式下已扣除的平台余额', 'Platform balance spent under managed billing') }}</p></div>
        </div>
        <section class="card flex flex-col gap-4 p-5 lg:flex-row lg:items-center">
          <div class="lg:w-64 lg:shrink-0"><h2 class="font-semibold">{{ tr('邀请客户', 'Invite customers') }}</h2><p class="mt-1 text-xs leading-5 text-gray-500">{{ tr('通过此链接注册的客户归属于你，充值时会联系你。', 'Customers who register through this link belong to you and contact you for credits.') }}</p></div>
          <div class="flex min-w-0 flex-1 flex-col gap-3 sm:flex-row"><input :value="inviteLink" readonly class="input min-w-0 flex-1 font-mono text-sm" :aria-label="tr('站长邀请链接', 'Invitation link')" /><button class="btn btn-secondary shrink-0" :disabled="!inviteLink" @click="copyInvite">{{ tr('复制邀请链接', 'Copy invite link') }}</button></div>
        </section>
        <div class="flex overflow-x-auto border-b border-gray-200 dark:border-dark-700" role="tablist" :aria-label="tr('站长中心', 'Reseller center')"><button v-for="tab in tabs" :key="tab.id" role="tab" :aria-selected="activeTab === tab.id" class="shrink-0 whitespace-nowrap border-b-2 px-5 py-3 text-sm font-medium" :class="activeTab === tab.id ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500'" @click="activeTab = tab.id">{{ tr(tab.zh, tab.en) }}</button></div>
        <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
        <section v-if="activeTab === 'customers'" class="card overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 p-5"><div><h2 class="font-semibold">{{ tr('客户管理', 'Customers') }}</h2><p class="mt-1 text-xs text-gray-500">{{ tr('管理客户额度、权限与 Key。', 'Manage customer credits, access and API keys.') }}</p></div><form class="flex gap-2" @submit.prevent="customerPage = 1; loadCustomers()"><input v-model="search" class="input" :placeholder="tr('搜索邮箱、用户名或备注', 'Search customers')" /><button class="btn btn-secondary shrink-0">{{ tr('搜索', 'Search') }}</button></form></div>
          <div class="overflow-x-auto"><table class="w-full text-left text-sm"><thead class="border-y border-gray-100 bg-gray-50 text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-800"><tr><th class="px-5 py-3">{{ tr('客户', 'Customer') }}</th><th class="px-5 py-3">{{ tr('可用 / 冻结额度', 'Available / reserved') }}</th><th class="px-5 py-3">{{ tr('消费额度 / 你的成本', 'Credits used / your cost') }}</th><th class="px-5 py-3">{{ tr('备注', 'Notes') }}</th><th class="px-5 py-3">{{ tr('操作', 'Actions') }}</th></tr></thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700"><tr v-for="customer in customers?.items" :key="customer.user_id">
              <td class="px-5 py-4"><div class="flex items-center gap-2"><p class="font-medium">{{ customer.username || customer.email }}</p><span class="rounded-full px-2 py-0.5 text-xs" :class="customer.status === 'active' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700'">{{ customer.status === 'active' ? tr('启用', 'Active') : tr('停用', 'Disabled') }}</span></div><p class="mt-1 text-xs text-gray-500">{{ customer.email }}</p></td>
              <td class="px-5 py-4 tabular-nums"><p class="font-medium">{{ money(customer.credit_balance) }}</p><p class="mt-1 text-xs text-gray-500">{{ tr('冻结', 'Reserved') }} {{ money(customer.frozen_credit) }}</p></td>
              <td class="px-5 py-4 tabular-nums"><p>{{ money(customer.charged) }}</p><p class="mt-1 text-xs text-gray-500">{{ tr('成本', 'Cost') }} {{ money(customer.cost) }}</p></td>
              <td class="max-w-48 break-words px-5 py-4 text-xs text-gray-500">{{ customer.notes || '—' }}</td>
              <td class="px-5 py-4"><div class="flex gap-3 whitespace-nowrap"><button class="font-medium text-primary-600" @click="creditCustomer = customer">{{ tr('额度', 'Credits') }}</button><button class="text-primary-600" @click="openCustomer(customer)">{{ tr('管理', 'Manage') }}</button><button class="text-gray-600 dark:text-gray-300" @click="openPricing(customer)">{{ tr('定价', 'Pricing') }}</button></div></td>
            </tr><tr v-if="!customers?.items.length"><td colspan="5" class="px-5 py-14 text-center text-gray-500">{{ tr('创建客户或分享邀请链接，开始管理客户额度。', 'Create a customer or share your invitation link to get started.') }}</td></tr></tbody>
          </table></div>
          <div class="flex items-center justify-between border-t border-gray-100 px-5 py-4 text-xs dark:border-dark-700"><span>{{ tr('共', 'Total') }} {{ customers?.total ?? 0 }}</span><div class="flex gap-2"><button class="btn btn-secondary btn-sm" :disabled="customerPage <= 1" @click="customerPage--; loadCustomers()">{{ tr('上一页', 'Previous') }}</button><button class="btn btn-secondary btn-sm" :disabled="!customers || customerPage * customers.page_size >= customers.total" @click="customerPage++; loadCustomers()">{{ tr('下一页', 'Next') }}</button></div></div>
        </section>
        <section v-else-if="activeTab === 'pricing'" class="card p-6">
          <div class="flex flex-wrap items-start justify-between gap-4"><div><h2 class="font-semibold">{{ tr('默认客户定价', 'Default customer pricing') }}</h2><p class="mt-2 max-w-2xl text-sm leading-6 text-gray-500">{{ tr('以你自己的分组价格为基础，乘以站长倍率得到客户价格。可为每个分组单独设置，也可在客户管理中覆盖单个客户。', 'Customer prices are your group prices multiplied by your markup. Configure group overrides here or individual overrides from Customers.') }}</p></div><button class="btn btn-primary" @click="openPricing(null)">{{ tr('编辑默认定价', 'Edit default pricing') }}</button></div>
          <div class="mt-6 rounded-xl bg-gray-50 p-5 dark:bg-dark-800"><p class="text-xs text-gray-500">{{ tr('默认倍率', 'Default multiplier') }}</p><p class="mt-2 text-3xl font-semibold tabular-nums">{{ pricing?.default_multiplier ?? overview.profile.default_multiplier }}×</p></div>
          <div class="mt-5 space-y-3 text-sm leading-6 text-gray-500"><p>{{ tr('1× 不产生消费加价。', '1× adds no consumption margin.') }}</p><p>{{ tr('改价只影响之后进入系统的请求。已创建的异步任务保留原价格和收益归属；失败释放预占时不记差价。', 'New prices apply to later requests. Existing async tasks retain their quoted prices and ownership. Released failed tasks earn no margin.') }}</p><p>{{ tr('客户向你付款，由你发放额度。客户使用时按客户价格扣额度，同时按你的平台价格扣余额；赠送额度的使用成本也由你承担。', 'You collect customer payments and issue credits. Usage consumes customer credits and your platform balance at their respective prices, including gifted credits.') }}</p></div>
        </section>

        <CustomerContactSettings v-else-if="activeTab === 'contact'" :profile="overview.profile" @updated="updateProfile" />
        <ResellerAnnouncements v-else-if="activeTab === 'announcements'" :profile="overview.profile" @updated="updateProfile" />
        <section v-else class="card overflow-hidden">
          <div class="flex items-center justify-between p-5"><div><h2 class="font-semibold">{{ tr('消费与成本记录', 'Usage and service costs') }}</h2><p class="mt-1 text-xs text-gray-500">{{ tr('客户消耗的额度与实际平台成本分别记录。线下收款和实际利润由你核对。', 'Customer credits and platform costs are recorded separately. Reconcile your own payments and profit.') }}</p></div><button class="btn btn-secondary btn-sm" @click="loadEarnings">{{ tr('刷新', 'Refresh') }}</button></div>
          <div class="overflow-x-auto"><table class="w-full text-left text-sm"><thead class="border-y border-gray-100 bg-gray-50 text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-800"><tr><th class="px-5 py-3">{{ tr('时间 / 客户', 'Time / customer') }}</th><th class="px-5 py-3">{{ tr('分组 / 模型', 'Group / model') }}</th><th class="px-5 py-3">{{ tr('客户消费额度', 'Customer credits') }}</th><th class="px-5 py-3">{{ tr('平台成本', 'Platform cost') }}</th><th class="px-5 py-3">{{ tr('结算方式', 'Settlement') }}</th></tr></thead><tbody class="divide-y divide-gray-100 dark:divide-dark-700"><tr v-for="entry in earnings?.items" :key="entry.id"><td class="px-5 py-4"><p>{{ entry.username || `#${entry.customer_id}` }}</p><p class="mt-1 whitespace-nowrap text-xs text-gray-500">{{ date(entry.created_at) }}</p></td><td class="px-5 py-4"><p>{{ entry.group_name }}</p><p class="mt-1 text-xs text-gray-500">{{ entry.model }}</p></td><td class="px-5 py-4 tabular-nums">{{ money(entry.charged) }}</td><td class="px-5 py-4 tabular-nums">{{ money(entry.cost) }}</td><td class="px-5 py-4 text-xs text-gray-500">{{ entry.settlement_type === 'managed_credit' ? tr('站长余额结算', 'Reseller funded') : tr('历史客户直接扣费', 'Historical direct charge') }}</td></tr><tr v-if="!earnings?.items.length"><td colspan="5" class="py-14 text-center text-gray-500">{{ tr('暂无消费记录', 'No usage yet') }}</td></tr></tbody></table></div>
          <div class="flex items-center justify-between border-t border-gray-100 px-5 py-4 text-xs dark:border-dark-700"><span>{{ tr('共', 'Total') }} {{ earnings?.total ?? 0 }}</span><div class="flex gap-2"><button class="btn btn-secondary btn-sm" :disabled="earningsPage <= 1" @click="earningsPage--; loadEarnings()">{{ tr('上一页', 'Previous') }}</button><button class="btn btn-secondary btn-sm" :disabled="!earnings || earningsPage * earnings.page_size >= earnings.total" @click="earningsPage++; loadEarnings()">{{ tr('下一页', 'Next') }}</button></div></div>
        </section>
      </template>
    </div>
    <BaseDialog :show="pricingOpen" :title="priceCustomer ? `${tr('客户定价', 'Customer pricing')} · ${priceCustomer.username || priceCustomer.email}` : tr('默认客户定价', 'Default pricing')" width="wide" @close="pricingOpen = false">
      <form id="reseller-prices" class="space-y-5" @submit.prevent="savePricing">
        <p class="text-sm leading-6 text-gray-500">{{ tr('倍率相对于你的分组价格。留空的分组继承整体倍率；客户整体倍率留空时，继承你的默认客户定价。', 'Multipliers are relative to your group prices. Empty group rows inherit the overall multiplier. An empty customer overall multiplier inherits your defaults.') }}</p>
        <label class="block text-sm font-medium">{{ tr('整体倍率', 'Overall multiplier') }}<input v-model="overallInput" type="number" :min="defaults.minimum_multiplier" step="any" :required="!priceCustomer" class="input mt-2 w-48" :placeholder="priceCustomer ? tr('继承默认定价', 'Inherit defaults') : ''" /></label>
        <div class="overflow-x-auto rounded-xl border border-gray-200 dark:border-dark-600"><table class="w-full text-sm"><thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800"><tr><th class="px-4 py-3">{{ tr('分组', 'Group') }}</th><th class="px-4 py-3">{{ tr('你的基础倍率', 'Your base rate') }}</th><th class="px-4 py-3">{{ tr('客户加价倍率', 'Customer multiplier') }}</th><th class="px-4 py-3">{{ tr('客户基础倍率', 'Customer base rate') }}</th></tr></thead><tbody class="divide-y divide-gray-100 dark:divide-dark-700"><tr v-for="group in pricing?.groups" :key="group.id"><td class="px-4 py-3"><p class="font-medium">{{ group.name }}</p><p class="text-xs text-gray-500">{{ group.platform }}</p></td><td class="px-4 py-3 tabular-nums">{{ baseRate(group) }}×</td><td class="px-4 py-3"><input v-model="groupInputs[group.id]" type="number" :min="defaults.minimum_multiplier" step="any" class="input w-36" :aria-label="`${group.name} ${tr('倍率', 'multiplier')}`" :placeholder="`${tr('继承', 'Inherit')} ${inheritedMultiplier(group.id)}×`" /></td><td class="px-4 py-3 font-medium tabular-nums text-primary-600">{{ Number((baseRate(group) * effectiveMultiplier(group.id)).toFixed(8)) }}×</td></tr></tbody></table></div>
        <p class="text-xs leading-5 text-gray-500">{{ tr('视频平台预览视频基础倍率，其他平台预览文本基础倍率。图片和视频的独立费率也应用客户加价；文本时段系数继续按原规则计算。', 'Video groups preview video rates; other groups preview text rates. Independent media rates receive the same markup. Text time-window factors retain their existing behavior.') }}</p>
        <p v-if="dialogError" role="alert" class="text-sm text-red-600">{{ dialogError }}</p>
      </form>
      <template #footer><button class="btn btn-secondary" @click="pricingOpen = false">{{ tr('取消', 'Cancel') }}</button><button type="submit" form="reseller-prices" class="btn btn-primary" :disabled="saving">{{ saving ? tr('保存中…', 'Saving…') : tr('保存定价', 'Save pricing') }}</button></template>
    </BaseDialog>

    <CustomerWorkspace v-if="overview" :show="workspaceOpen" :customer="workspaceCustomer" :groups="pricing?.groups ?? []" :limits="overview.limits" @close="workspaceOpen = false" @saved="refreshCustomerData" />
    <CustomerCreditDialog :customer="creditCustomer" @close="creditCustomer = null" @saved="refreshCustomerData" />
  </AppLayout>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import CustomerWorkspace from '@/components/gotocc/reseller/CustomerWorkspace.vue'
import CustomerCreditDialog from '@/components/gotocc/reseller/CustomerCreditDialog.vue'
import CustomerContactSettings from '@/components/gotocc/reseller/CustomerContactSettings.vue'
import ResellerAnnouncements from '@/components/gotocc/reseller/ResellerAnnouncements.vue'
import { resellerAPI, type ResellerOverview, type ResellerProfile, type ResellerPage, type ResellerCustomer, type ResellerEarning, type ResellerPrices, type ResellerGroup } from '@/api/reseller'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useAppStore } from '@/stores/app'
import defaults from '../../../../reseller-defaults.json'
const { locale } = useI18n(); const tr = (zh: string,en: string) => locale.value.startsWith('zh') ? zh : en
const app = useAppStore(); const loading = ref(true); const saving = ref(false); const error = ref(''); const dialogError = ref('')
const initialCredit = ref(defaults.default_initial_credit); const initialSaving = ref(false)
const overview = ref<ResellerOverview>(); const customers = ref<ResellerPage<ResellerCustomer>>(); const earnings = ref<ResellerPage<ResellerEarning>>(); const pricing = ref<ResellerPrices>()
const activeTab = ref('customers'); const tabs = [{id:'customers',zh:'客户管理',en:'Customers'},{id:'pricing',zh:'客户定价',en:'Pricing'},{id:'earnings',zh:'消费与成本',en:'Usage & costs'},{id:'contact',zh:'客户展示',en:'Customer display'},{id:'announcements',zh:'公告管理',en:'Announcements'}]
const search = ref(''); const customerPage = ref(1); const earningsPage = ref(1)
const pricingOpen = ref(false); const priceCustomer = ref<ResellerCustomer|null>(null); const overallInput = ref(''); const groupInputs = ref<Record<number,string>>({})
const workspaceOpen = ref(false); const workspaceCustomer = ref<ResellerCustomer|null>(null); const creditCustomer = ref<ResellerCustomer|null>(null)
function openCustomer(customer: ResellerCustomer|null) { workspaceCustomer.value = customer; workspaceOpen.value = true }
async function refreshCustomerData() { await Promise.all([loadCustomers(), loadEarnings()]); overview.value = await resellerAPI.overview() }
const inviteLink = computed(() => overview.value?.invitation_url ?? '')
const money = (v: number) => new Intl.NumberFormat(locale.value, {style:'currency',currency:'USD',minimumFractionDigits:2,maximumFractionDigits:8}).format(v)
const date = (v: string) => new Date(v).toLocaleString(locale.value)
const baseRate = (g: ResellerGroup) => g.platform === 'video' && g.video_independent ? g.video_multiplier : g.base_multiplier
function updateProfile(profile: ResellerProfile) { if (overview.value) overview.value.profile = profile }
async function load() { loading.value = true; error.value = ''; try { overview.value = await resellerAPI.overview(); initialCredit.value = overview.value.profile.initial_credit; await Promise.all([loadCustomers(),loadEarnings(),loadPrices()]) } catch(e) { error.value = extractApiErrorMessage(e,tr('读取站长中心失败','Could not load reseller center')) } finally { loading.value = false } }
async function loadCustomers() { try { customers.value = await resellerAPI.customers(customerPage.value,search.value) } catch(e) { error.value = extractApiErrorMessage(e,tr('读取客户失败','Could not load customers')) } }
async function loadEarnings() { try { earnings.value = await resellerAPI.earnings(earningsPage.value) } catch(e) { error.value = extractApiErrorMessage(e,tr('读取消费记录失败','Could not load usage')) } }
async function loadPrices() { pricing.value = await resellerAPI.prices() }
async function saveInitialCredit() {
  initialSaving.value = true
  try {
    const profile = await resellerAPI.saveInitialCredit(initialCredit.value)
    if (overview.value) overview.value.profile = profile
    initialCredit.value = profile.initial_credit
    app.showSuccess(tr('初始额度已保存', 'Initial credits saved'))
  } catch (e) { app.showError(extractApiErrorMessage(e, tr('保存初始额度失败', 'Could not save initial credits'))) }
  finally { initialSaving.value = false }
}
async function copyInvite() { try { await navigator.clipboard.writeText(inviteLink.value); app.showSuccess(tr('邀请链接已复制','Invitation link copied')) } catch { app.showError(tr('请手动复制上方链接','Copy the link manually')) } }
async function openPricing(customer: ResellerCustomer|null) {
  error.value = ''; dialogError.value = ''
  try { await loadPrices(); priceCustomer.value = customer; const rows = pricing.value!.prices.filter(p => p.customer_id === (customer?.user_id ?? null)); overallInput.value = customer ? String(rows.find(p => p.group_id === null)?.multiplier ?? '') : String(pricing.value!.default_multiplier); groupInputs.value = Object.fromEntries(pricing.value!.groups.map(g => [g.id, String(rows.find(p => p.group_id === g.id)?.multiplier ?? '')])); pricingOpen.value = true } catch(e) { error.value = extractApiErrorMessage(e,tr('读取定价失败','Could not load pricing')) }
}
function inheritedMultiplier(groupID: number) { if (overallInput.value !== '') return Number(overallInput.value); return pricing.value?.prices.find(p => p.customer_id === null && p.group_id === groupID)?.multiplier ?? pricing.value?.default_multiplier ?? defaults.default_multiplier }
function effectiveMultiplier(groupID: number) { return groupInputs.value[groupID] === '' ? inheritedMultiplier(groupID) : Number(groupInputs.value[groupID]) }
async function savePricing() {
  saving.value = true; dialogError.value = ''
  try { await resellerAPI.savePrices(priceCustomer.value?.user_id ?? null, overallInput.value === '' ? null : Number(overallInput.value), (pricing.value?.groups ?? []).map(g => ({customer_id:priceCustomer.value?.user_id ?? null,group_id:g.id,multiplier:groupInputs.value[g.id] === '' ? null : Number(groupInputs.value[g.id])}))); pricingOpen.value = false; await loadPrices(); overview.value = await resellerAPI.overview(); app.showSuccess(tr('客户定价已保存，后续请求使用新价格','Pricing saved for subsequent requests')) } catch(e) { dialogError.value = extractApiErrorMessage(e,tr('保存定价失败','Could not save pricing')) } finally { saving.value = false }
}
watch(activeTab, tab => { if (tab === 'earnings') void loadEarnings() })
onMounted(load)
</script>

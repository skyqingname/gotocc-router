<template>
  <AppLayout>
    <div class="space-y-6">
      <div><h1 class="text-2xl font-semibold">{{ tr('我的额度', 'My credits') }}</h1><p class="mt-2 text-sm text-gray-500">{{ tr('购买额度、赠送福利与消费记录。', 'Purchased credits, gifts and usage records.') }}</p></div>
      <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
      <div v-if="data" class="grid gap-4 sm:grid-cols-3">
        <div class="card p-6"><p class="text-sm text-gray-500">{{ tr('可用额度', 'Available credits') }}</p><p class="mt-3 text-3xl font-semibold tabular-nums">{{ money(data.account.credit_balance) }}</p></div>
        <div class="card p-6"><p class="text-sm text-gray-500">{{ tr('任务冻结额度', 'Reserved credits') }}</p><p class="mt-3 text-3xl font-semibold tabular-nums">{{ money(data.account.frozen_credit) }}</p></div>
        <div class="card border-primary-200 bg-primary-50/40 p-6 dark:border-primary-800 dark:bg-primary-950/20"><p class="font-medium">{{ tr('请联系站长充值', 'Contact your reseller for credits') }}</p><p class="mt-2 text-sm leading-6 text-gray-500">{{ data.account.owner_name }}<br />{{ tr('额度购买、福利和现金退款由所属站长处理。', 'Your reseller handles credit purchases, gifts and cash refunds.') }}</p></div>
      </div>
      <section class="card overflow-hidden">
        <div class="flex items-center justify-between p-5"><h2 class="font-semibold">{{ tr('额度流水', 'Credit history') }}</h2><button class="btn btn-secondary btn-sm" :disabled="loading" @click="load">{{ tr('刷新', 'Refresh') }}</button></div>
        <CreditLedger :entries="data?.items ?? []" />
        <div class="flex items-center justify-between border-t border-gray-100 p-5 text-sm dark:border-dark-700"><span class="text-gray-500">{{ loading ? tr('读取中…', 'Loading…') : `${tr('共', 'Total')} ${data?.total ?? 0}` }}</span><div class="flex gap-2"><button class="btn btn-secondary btn-sm" :disabled="page <= 1 || loading" @click="page--; load()">{{ tr('上一页', 'Previous') }}</button><button class="btn btn-secondary btn-sm" :disabled="!data || page * data.page_size >= data.total || loading" @click="page++; load()">{{ tr('下一页', 'Next') }}</button></div></div>
      </section>
    </div>
  </AppLayout>
</template>
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import CreditLedger from '@/components/gotocc/reseller/CreditLedger.vue'
import { resellerAPI, type MyResellerCredits } from '@/api/reseller'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorMessage } from '@/utils/apiError'
const { locale } = useI18n(); const auth = useAuthStore()
const tr = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const money = (value: number) => value.toLocaleString(locale.value, { maximumFractionDigits: 8 })
const data = ref<MyResellerCredits>(); const page = ref(1); const loading = ref(false); const error = ref('')
async function load() {
  loading.value = true; error.value = ''
  try { data.value = await resellerAPI.myCredits(page.value); if (auth.user) { auth.user.reseller_customer = data.value.account; auth.user.balance = data.value.account.credit_balance; auth.user.frozen_balance = data.value.account.frozen_credit } }
  catch (e) { error.value = extractApiErrorMessage(e, tr('无法读取额度流水', 'Could not load credit history')) }
  finally { loading.value = false }
}
onMounted(load)
</script>

<template>
  <div class="overflow-x-auto">
    <table class="w-full text-left text-sm">
      <thead class="border-y border-gray-100 bg-gray-50 text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-800">
        <tr><th class="px-4 py-3">{{ tr('时间 / 类型', 'Time / type') }}</th><th class="px-4 py-3">{{ tr('可用额度变动', 'Available change') }}</th><th class="px-4 py-3">{{ tr('冻结变动', 'Reserved change') }}</th><th class="px-4 py-3">{{ tr('变动后可用', 'Available after') }}</th><th v-if="owner" class="px-4 py-3">{{ tr('备注 / 模型', 'Note / model') }}</th></tr>
      </thead>
      <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
        <tr v-for="entry in entries" :key="entry.id">
          <td class="px-4 py-3"><p class="font-medium">{{ kindLabel(entry.kind) }}</p><p class="mt-1 whitespace-nowrap text-xs text-gray-500">{{ new Date(entry.created_at).toLocaleString(locale) }}</p></td>
          <td class="px-4 py-3 font-medium tabular-nums" :class="entry.amount > 0 ? 'text-emerald-600 dark:text-emerald-400' : ''">{{ entry.amount > 0 ? '+' : '' }}{{ money(entry.amount) }}</td>
          <td class="px-4 py-3 tabular-nums text-gray-500">{{ entry.frozen_amount > 0 ? '+' : '' }}{{ money(entry.frozen_amount) }}</td>
          <td class="px-4 py-3 tabular-nums">{{ money(entry.balance_after) }}</td>
          <td v-if="owner" class="max-w-64 break-words px-4 py-3 text-xs text-gray-500">{{ entry.notes || entry.model || '—' }}</td>
        </tr>
        <tr v-if="!entries.length"><td :colspan="owner ? 5 : 4" class="px-4 py-12 text-center text-gray-500">{{ tr('还没有额度流水', 'No credit entries yet') }}</td></tr>
      </tbody>
    </table>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ResellerCreditEntry } from '@/api/reseller'
defineProps<{ entries: ResellerCreditEntry[]; owner?: boolean }>()
const { locale } = useI18n()
const tr = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const money = (value: number) => value.toLocaleString(locale.value, { maximumFractionDigits: 8 })
function kindLabel(kind: string) {
  const labels: Record<string, [string, string]> = { increase: ['增加额度', 'Credit increase'], initial: ['初始额度', 'Initial credits'], purchase: ['购买额度', 'Purchased credits'], gift: ['赠送额度', 'Gift credits'], deduct: ['减少额度', 'Credit decrease'], consume: ['消费扣除', 'Usage charge'], reserve: ['任务预占', 'Task reservation'], capture: ['任务结算', 'Task settlement'], release: ['预占释放', 'Reservation release'] }
  const label = labels[kind]
  return label ? tr(...label) : kind
}
</script>

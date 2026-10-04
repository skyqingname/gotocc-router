<template>
  <div class="card">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
      <h2 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
        <Icon name="document" size="sm" class="text-primary-600 dark:text-primary-400" />
        {{ t('team.usageDetails') }}
        <span class="text-xs font-normal text-gray-500 dark:text-dark-400">{{ t('team.recordCount', { count: total }) }}</span>
      </h2>
      <div class="flex flex-wrap items-center gap-2">
        <button type="button" class="btn btn-secondary btn-sm" :disabled="exporting || total === 0" @click="exportCsv">
          <Icon name="download" size="sm" :class="{ 'animate-pulse': exporting }" />
          {{ exporting ? t('team.exporting') : t('team.exportCsv') }}
        </button>
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="min-w-full text-sm">
        <thead class="text-left text-xs text-gray-500 dark:text-dark-400">
          <tr class="border-b border-gray-100 dark:border-dark-700">
            <th class="whitespace-nowrap px-5 py-3 font-medium">{{ t('team.time') }}</th>
            <th v-if="isOwner" class="whitespace-nowrap px-3 py-3 font-medium">{{ t('team.keyOwner') }}</th>
            <th class="whitespace-nowrap px-3 py-3 font-medium">{{ t('team.keys') }}</th>
            <th class="whitespace-nowrap px-3 py-3 font-medium">{{ t('team.model') }}</th>
            <th class="whitespace-nowrap px-3 py-3 text-right font-medium">Token</th>
            <th class="whitespace-nowrap px-5 py-3 text-right font-medium">{{ t('team.cost') }}</th>
          </tr>
        </thead>
        <tbody v-if="loading && items.length === 0">
          <tr v-for="row in pageSize" :key="row"><td :colspan="isOwner ? 6 : 5" class="px-5 py-3"><Skeleton :height="16" /></td></tr>
        </tbody>
        <tbody v-else-if="items.length === 0">
          <tr><td :colspan="isOwner ? 6 : 5" class="px-5 py-12 text-center text-gray-500 dark:text-dark-400">{{ t('team.noUsage') }}</td></tr>
        </tbody>
        <tbody v-else class="divide-y divide-gray-50 dark:divide-dark-700/60" :class="{ 'opacity-60': loading }">
          <tr v-for="item in items" :key="item.id" class="transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/40">
            <td class="whitespace-nowrap px-5 py-3 text-gray-500 dark:text-dark-400">{{ formatDateTime(item.created_at) }}</td>
            <td v-if="isOwner" class="max-w-48 truncate px-3 py-3 text-gray-700 dark:text-dark-200" :title="item.actor_email">{{ item.actor_email }}</td>
            <td class="whitespace-nowrap px-3 py-3 text-gray-700 dark:text-dark-200">{{ item.api_key_name }}</td>
            <td class="px-3 py-3">
              <span class="flex min-w-0 items-center gap-2">
                <ModelIcon :model="item.model" size="16px" />
                <span class="max-w-56 truncate font-medium text-gray-900 dark:text-white" :title="item.model">{{ item.model }}</span>
              </span>
            </td>
            <td class="whitespace-nowrap px-3 py-3 text-right tabular-nums text-gray-600 dark:text-dark-300">
              {{ formatTokensK(item.input_tokens) }} / {{ formatTokensK(item.output_tokens) }}
            </td>
            <td class="whitespace-nowrap px-5 py-3 text-right font-medium tabular-nums text-gray-900 dark:text-white">{{ formatTeamCost(item.actual_cost) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="total > pageSize" class="border-t border-gray-100 px-5 py-3 dark:border-dark-700">
      <Pagination v-model:page="page" :total="total" :page-size="pageSize" :show-page-size-selector="false" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Pagination from '@/components/common/Pagination.vue'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import { teamAPI, type TeamUsageLog, type TeamUsageQuery } from '@/api/team'
import { formatDateTime, formatTokensK } from '@/utils/format'
import { formatTeamCost } from './teamFormat'

// 团队用量明细：成员和 Key 筛选来自概览顶部，成员只看到自己的记录；导出当前筛选下的全部记录。
const PAGE_SIZE = 10
// 导出时每次取的条数，与接口单页上限一致。
const EXPORT_BATCH = 100

const props = defineProps<{
  range: { from: string; to: string }
  memberId: number | null
  keyId: number | null
  isOwner: boolean
}>()

const { t } = useI18n()
const page = ref(1)
const pageSize = PAGE_SIZE
const items = ref<TeamUsageLog[]>([])
const total = ref(0)
const loading = ref(false)
const exporting = ref(false)

const query = (offset: number, limit: number): TeamUsageQuery => {
  const result: TeamUsageQuery = { from: props.range.from, to: props.range.to, limit, offset }
  if (props.memberId !== null) result.member_id = props.memberId
  if (props.keyId !== null) result.api_key_id = props.keyId
  return result
}

// 用递增序号丢弃过期响应，快速翻页或切换筛选时只保留最后一次结果。
let requestSeq = 0

const load = async () => {
  const seq = ++requestSeq
  loading.value = true
  const result = await teamAPI.usageLogs(query((page.value - 1) * pageSize, pageSize)).finally(() => {
    if (seq === requestSeq) loading.value = false
  })
  if (seq !== requestSeq) return
  items.value = result.items
  total.value = result.total
}

watch([() => props.range.from, () => props.range.to, () => props.memberId, () => props.keyId], () => {
  page.value = 1
  void load()
}, { immediate: true })
watch(page, () => void load())

// 逐页取完当前筛选下的全部记录，生成带 BOM 的 CSV，Excel 打开中文不乱码。
const exportCsv = async () => {
  exporting.value = true
  const rows: TeamUsageLog[] = []
  try {
    while (rows.length < total.value) {
      const result = await teamAPI.usageLogs(query(rows.length, EXPORT_BATCH))
      if (result.items.length === 0) break
      rows.push(...result.items)
    }
  } finally {
    exporting.value = false
  }
  const escape = (value: string | number) => `"${String(value).replace(/"/g, '""')}"`
  const header = [t('team.time'), t('team.keyOwner'), t('team.keys'), t('team.model'), t('team.inputTokens'), t('team.outputTokens'), t('team.costUsd')]
  const lines = rows.map((row) => [formatDateTime(row.created_at), row.actor_email, row.api_key_name, row.model, row.input_tokens, row.output_tokens, row.actual_cost.toFixed(6)].map(escape).join(','))
  const blob = new Blob([`﻿${[header.map(escape).join(','), ...lines].join('\n')}`], { type: 'text/csv;charset=utf-8' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `team-usage-${props.range.from}-${props.range.to}.csv`
  link.click()
  URL.revokeObjectURL(link.href)
}
</script>

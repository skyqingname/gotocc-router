<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('canvasCard.adminHint') }}</p>
          <div class="flex gap-2">
            <RouterLink to="/admin/redeem" class="btn btn-secondary">{{ t('canvasCard.backToRedeem') }}</RouterLink>
            <button class="btn btn-primary" @click="showImport = true">{{ t('canvasCard.import') }}</button>
          </div>
        </div>
        <div class="mt-4 grid grid-cols-2 gap-3 lg:grid-cols-4">
          <div v-for="item in stock" :key="item.denomination" class="card flex items-center justify-between gap-3 p-4">
            <div><p class="text-xs text-gray-500">{{ t('canvasCard.denomination') }}</p><p class="mt-1 text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ item.denomination }}</p></div>
            <div class="text-right text-xs leading-6"><p class="text-primary-700 dark:text-primary-300">{{ t('canvasCard.availableStock') }} <strong class="tabular-nums">{{ item.available }}</strong></p><p class="text-gray-500">{{ t('canvasCard.issuedStock') }} {{ item.issued }}</p></div>
          </div>
        </div>
      </template>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <Select v-model="denomination" :options="filterAmounts" :aria-label="t('canvasCard.denomination')" class="w-40" :disabled="loading" @change="load(1)" />
          <Select v-model="status" :options="statusOptions" :aria-label="t('common.status')" class="w-40" :disabled="loading" @change="load(1)" />
          <button class="btn btn-secondary ml-auto" :disabled="loading" @click="load(page)">{{ t('common.refresh') }}</button>
        </div>
        <p v-if="loadError" role="alert" class="mt-2 text-sm text-red-600">{{ loadError }}</p>
      </template>
      <template #table>
        <DataTable :columns="columns" :data="inventory?.items ?? []" :loading="loading" :empty-text="t('canvasCard.noInventory')">
          <template #cell-code="{ row }"><code class="block max-w-xs break-all whitespace-normal text-sm">{{ row.code }}</code></template>
          <template #cell-status="{ row }"><span class="badge" :class="row.issued_at ? 'badge-gray' : 'badge-success'">{{ t(row.issued_at ? 'canvasCard.issuedStock' : 'canvasCard.availableStock') }}</span></template>
          <template #cell-recipient="{ row }"><span v-if="row.user_id">#{{ row.user_id }}</span><span v-else>—</span></template>
          <template #cell-funding="{ row }"><span v-if="row.funding_source">{{ t(row.funding_source === 'team' ? 'canvasCard.team' : 'canvasCard.personal') }}<span v-if="row.team_id"> · #{{ row.team_id }}</span></span><span v-else>—</span></template>
          <template #cell-created_at="{ row }">{{ formatDateTime(row.created_at) }}</template>
          <template #cell-issued_at="{ row }">{{ row.issued_at ? formatDateTime(row.issued_at) : '—' }}</template>
        </DataTable>
      </template>
      <template #pagination>
        <div v-if="inventory" class="flex items-center justify-between gap-3 text-sm text-gray-500">
          <span>{{ t('common.total') }}: {{ inventory.total }}</span>
          <div class="flex items-center gap-3"><button class="btn btn-secondary btn-sm" :disabled="loading || page <= 1" @click="load(page - 1)">{{ t('pagination.previous') }}</button><span>{{ page }} / {{ inventory.pages }}</span><button class="btn btn-secondary btn-sm" :disabled="loading || page >= inventory.pages" @click="load(page + 1)">{{ t('pagination.next') }}</button></div>
        </div>
      </template>
    </TablePageLayout>
    <BaseDialog :show="showImport" :title="t('canvasCard.import')" width="normal" @close="closeImport">
      <form id="canvas-card-import" class="space-y-4" @submit.prevent="importCards">
        <div><label for="canvas-card-amount" class="input-label">{{ t('canvasCard.denomination') }}</label><Select id="canvas-card-amount" v-model="importAmount" :options="amountOptions" :disabled="importing" /></div>
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('canvasCard.importHint', { count: parameters.import_batch_limit }) }}</p>
        <textarea v-model="codesText" class="input min-h-52 font-mono" :aria-label="t('canvasCard.code')" :placeholder="t('canvasCard.importPlaceholder')" :disabled="importing" autocomplete="off" spellcheck="false" required />
        <p class="text-xs text-gray-500">{{ t('canvasCard.selectedCount', { count: codes.length }) }}</p>
        <p v-if="importError" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ importError }}</p>
      </form>
      <template #footer><div class="flex justify-end gap-3"><button class="btn btn-secondary" :disabled="importing" @click="closeImport">{{ t('common.cancel') }}</button><button form="canvas-card-import" type="submit" class="btn btn-primary" :disabled="importing || !codes.length || codes.length > parameters.import_batch_limit">{{ t(importing ? 'canvasCard.importing' : 'canvasCard.import') }}</button></div></template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Select from '@/components/common/Select.vue'
import { canvasCardAPI, type CanvasCardStock, type CanvasCardItem, type CanvasCardPage } from '@/api/canvasCard'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import parameters from '../../../../canvas-card-exchange.json'

const { t } = useI18n()
const app = useAppStore()
const stock = ref<CanvasCardStock[]>([])
const inventory = ref<CanvasCardPage<CanvasCardItem> | null>(null)
const denomination = ref(0)
const status = ref('')
const page = ref(1)
const loading = ref(false)
const loadError = ref('')
const showImport = ref(false)
const importing = ref(false)
const importAmount = ref(parameters.denominations[0])
const codesText = ref('')
const importError = ref('')
const codes = computed(() => codesText.value.split(/\r?\n/).map(code => code.trim()).filter(Boolean))
const amountOptions = computed(() => parameters.denominations.map(value => ({ value, label: String(value) })))
const filterAmounts = computed(() => [{ value: 0, label: t('canvasCard.allAmounts') }, ...amountOptions.value])
const statusOptions = computed(() => [{ value: '', label: t('canvasCard.allStatuses') }, { value: 'available', label: t('canvasCard.availableStock') }, { value: 'issued', label: t('canvasCard.issuedStock') }])
const columns = computed(() => [
  { key: 'denomination', label: t('canvasCard.denomination') }, { key: 'code', label: t('canvasCard.code') },
  { key: 'status', label: t('common.status') }, { key: 'recipient', label: t('canvasCard.recipient') },
  { key: 'funding', label: t('canvasCard.funding') }, { key: 'created_at', label: t('canvasCard.importedAt') }, { key: 'issued_at', label: t('canvasCard.issuedAt') },
])
async function load(target: number) {
  loading.value = true
  loadError.value = ''
  try { const [counts, list] = await Promise.all([canvasCardAPI.stock(), canvasCardAPI.inventory(target, denomination.value, status.value)]); stock.value = counts; inventory.value = list; page.value = list.page }
  catch (error: any) { loadError.value = error.message || t('common.error') }
  finally { loading.value = false }
}
function closeImport() { if (!importing.value) showImport.value = false }
async function importCards() {
  if (importing.value) return
  importing.value = true
  importError.value = ''
  try {
    const result = await canvasCardAPI.import(importAmount.value, codes.value)
    codesText.value = ''
    showImport.value = false
    app.showSuccess(t('canvasCard.imported', { count: result.imported }))
    await load(1)
  } catch (error: any) { importError.value = error.status === 0 || error.status >= 500 ? t('canvasCard.importUnknown') : error.message || t('common.error') }
  finally { importing.value = false }
}
onMounted(() => { void load(1) })
</script>

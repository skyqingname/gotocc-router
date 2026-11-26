<template>
  <div class="flex h-full min-h-0 flex-col" :class="{ 'error-compact': compact, 'workspace-body gap-4': workspace }" :style="{ '--error-table-width': compactTableWidth + 'px' }">
    <div class="card flex min-h-0 flex-1 flex-col overflow-hidden" :class="{ 'workspace-table-card workspace-shell': workspace }">
      <IpGeoBatchToolbar :class="{ 'workspace-fixed': workspace }" :ips="rows.map((r) => r.client_ip)" @failed="emit('ipGeoBatchFailed')" />

      <DataTable
        :class="{ 'workspace-table-viewport': workspace }"
        :columns="columns"
        :data="rows"
        :loading="loading"
        :active-row-key="activeRowKey"
        clickable-rows
        server-side-sort
        default-sort-key="created_at"
        default-sort-order="desc"
        @sort="onSort"
        @rowClick="(row) => emit('openErrorDetail', row.id)"
      >
        <template #cell-created_at="{ row }">
          <button v-if="compact" type="button" class="text-left text-sm text-primary-600 underline decoration-dotted surface-control" :aria-label="t('inspection.view')" @click.stop="emit('openErrorDetail', row.id)">{{ formatDateTime(row.created_at) }}</button>
          <span
            v-else
            class="text-sm surface-muted"
            :title="row.request_id || row.client_request_id"
          >{{ formatDateTime(row.created_at) }}</span>
        </template>

        <template #cell-type="{ row }">
          <div class="flex flex-col items-start gap-0.5">
            <span class="inline-flex items-center rounded px-2 py-0.5 text-xs font-semibold" :class="rowDiagnosis(row).badgeClass">
              {{ rowDiagnosis(row).ownerLabel }} · {{ rowDiagnosis(row).subLabel }}
            </span>
          </div>
        </template>

        <template #cell-endpoint="{ row }">
          <div class="max-w-[320px] space-y-1 text-xs">
            <div class="break-all surface-muted">
              <span class="font-medium surface-muted">{{ t('usage.inbound') }}:</span>
              <span class="ml-1">{{ row.inbound_endpoint?.trim() || '-' }}</span>
            </div>
            <div v-if="row.upstream_endpoint" class="break-all surface-muted">
              <span class="font-medium surface-muted">{{ t('usage.upstream') }}:</span>
              <span class="ml-1">{{ row.upstream_endpoint?.trim() || '-' }}</span>
            </div>
          </div>
        </template>

        <template #cell-platform="{ row }">
          <span class="text-sm text-gray-900 dark:text-white">{{ row.platform || '-' }}</span>
        </template>

        <template #cell-model="{ row }">
          <div v-if="hasModelMapping(row)" class="space-y-0.5 text-xs">
            <div class="break-all font-medium text-gray-900 dark:text-white">{{ row.requested_model }}</div>
            <div class="break-all surface-muted"><span class="mr-0.5">↳</span>{{ row.upstream_model }}</div>
          </div>
          <span v-else-if="displayModel(row)" class="text-sm font-medium text-gray-900 dark:text-white">{{ displayModel(row) }}</span>
          <span v-else class="text-sm surface-muted">-</span>
        </template>

        <template #cell-group="{ row }">
          <span
            v-if="row.group_id"
            class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium bg-indigo-100/[0.15] text-indigo-800 dark:bg-indigo-900/25 dark:text-indigo-200"
            :title="t('admin.ops.errorLog.id') + ' ' + row.group_id"
          >
            {{ row.group_name || '#' + row.group_id }}
          </span>
          <span v-else class="text-sm surface-muted">-</span>
        </template>

        <template #cell-user="{ row }">
          <div v-if="row.user_id" class="text-sm" :class="compact ? 'max-w-[180px]' : ''">
            <button
              v-if="userClickable && row.user_email"
              class="max-w-full truncate font-medium text-primary-600 underline decoration-dashed underline-offset-2 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 surface-control"
              :title="row.user_email + ' · ' + t('admin.usage.clickToViewBalance')"
              @click.stop="emit('userClick', row.user_id, row.user_email)"
            >
              {{ row.user_email }}
            </button>
            <span v-else class="font-medium text-gray-900 dark:text-white" :class="compact ? 'block truncate' : ''" :title="row.user_email">{{ row.user_email || '#' + row.user_id }}</span>
            <span v-if="!compact" class="ml-1 surface-muted">#{{ row.user_id }}</span>
            <div v-if="compact && !columns.some(column => column.key === 'api_key')" class="truncate text-xs surface-muted" :title="row.api_key_name">
              {{ row.api_key_name || (row.api_key_id ? '#' + row.api_key_id : '—') }}
              <span v-if="row.api_key_deleted"> · {{ t('admin.ops.errorLog.keyDeletedBadge') }}</span>
            </div>
          </div>
          <span v-else class="text-sm surface-muted">-</span>
        </template>

        <template #cell-api_key="{ row }">
          <div v-if="row.api_key_id || row.api_key_name" class="text-sm">
            <span class="text-gray-900 dark:text-white">{{ row.api_key_name || '#' + row.api_key_id }}</span>
            <span
              v-if="row.api_key_deleted"
              class="ml-1 inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-rose-100/[0.15] text-rose-600 ring-1 ring-inset ring-rose-200 dark:bg-rose-500/20 dark:text-rose-400 dark:ring-rose-500/30"
            >{{ t('admin.ops.errorLog.keyDeletedBadge') }}</span>
          </div>
          <span v-else class="text-sm surface-muted">-</span>
        </template>

        <template #cell-account="{ row }">
          <span
            v-if="row.account_id"
            class="text-sm text-gray-900 dark:text-white"
            :class="compact ? 'block max-w-[160px] truncate' : ''"
            :title="(row.account_name || '') + ' · ' + t('admin.ops.errorLog.accountId') + ' ' + row.account_id"
          >{{ row.account_name || '#' + row.account_id }}</span>
          <span v-else class="text-sm surface-muted">-</span>
          <div v-if="compact && !columns.some(column => column.key === 'group')" class="max-w-[160px] truncate text-xs surface-muted" :title="row.group_name">{{ row.group_name || (row.group_id ? '#' + row.group_id : '—') }}</div>
        </template>

        <template #cell-category="{ row }">
          <span class="text-sm text-gray-900 dark:text-white">
            {{ t('usage.errors.categories.' + mapErrorCategory(row.phase, row.type)) }}
          </span>
        </template>

        <template #cell-status="{ row }">
          <div class="flex flex-wrap items-center gap-1.5">
            <template v-if="rowDiagnosis(row).gatewayStatus && rowDiagnosis(row).upstreamStatus && rowDiagnosis(row).gatewayStatus !== rowDiagnosis(row).upstreamStatus">
              <span class="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium" :class="getStatusClass(rowDiagnosis(row).gatewayStatus!)">
                {{ isZhLocale ? '网关' : 'GW' }} {{ rowDiagnosis(row).gatewayStatus }}
              </span>
              <span class="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium" :class="getStatusClass(rowDiagnosis(row).upstreamStatus!)">
                {{ isZhLocale ? '上游' : 'UP' }} {{ rowDiagnosis(row).upstreamStatus }}
              </span>
            </template>
            <span v-else class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium" :class="getStatusClass(row.status_code)">
              {{ row.status_code === 499 ? (isZhLocale ? '499 断连' : '499 Closed') : row.status_code }}
            </span>
            <span
              v-if="row.severity"
              :class="['rounded px-1.5 py-0.5 text-[10px] font-medium', getSeverityClass(row.severity)]"
            >{{ row.severity }}</span>
            <span
              v-if="row.request_type != null && row.request_type > 0"
              class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium surface-inset text-gray-800 dark:text-gray-200"
            >{{ formatRequestType(row.request_type) }}</span>
          </div>
        </template>

        <template #cell-message="{ row }">
          <div
            class="max-w-[420px] space-y-0.5 py-0.5"
            :title="[rowDiagnosis(row).headline, rowDiagnosis(row).filterHint, rowDiagnosis(row).detail].filter(Boolean).join('\n')"
          >
            <div class="truncate text-xs font-medium text-gray-900 dark:text-gray-100">
              {{ rowDiagnosis(row).headline }}
            </div>
            <div v-if="rowDiagnosis(row).filterHint" class="truncate text-[11px] text-amber-700 dark:text-amber-300">
              {{ rowDiagnosis(row).filterHint }}
            </div>
            <div v-if="rowDiagnosis(row).detail" class="truncate font-mono text-[11px] surface-muted">
              {{ formatSmartMessage(rowDiagnosis(row).detail) }}
            </div>
          </div>
        </template>

        <template #cell-user_agent="{ row }">
          <div v-if="row.user_agent" class="flex max-w-[320px] items-center gap-1.5" @click.stop>
            <span class="truncate text-sm surface-muted" :title="row.user_agent">{{ row.user_agent }}</span>
            <button
              type="button"
              class="shrink-0 rounded p-0.5 transition-colors hover:surface-hover hover:text-gray-600 dark:hover:text-gray-300 surface-control surface-muted"
              :class="copiedUserAgent === row.user_agent ? 'text-green-500 hover:text-green-500' : ''"
              :title="copiedUserAgent === row.user_agent ? t('keys.copied') : t('keys.copyToClipboard')"
              @click="copyUserAgent(row.user_agent)"
            >
              <Icon :name="copiedUserAgent === row.user_agent ? 'check' : 'copy'" size="sm" class="h-3.5 w-3.5" />
            </button>
          </div>
          <span v-else class="text-sm surface-muted">-</span>
        </template>

        <template #cell-client_ip="{ row }">
          <div @click.stop>
            <div v-if="row.client_ip">
              <span class="text-sm font-mono surface-muted">{{ row.client_ip }}</span>
              <IpGeoCell :ip="row.client_ip" />
            </div>
            <span v-else class="text-sm surface-muted">-</span>
          </div>
        </template>

        <template #cell-actions="{ row }">
          <button
            type="button"
            class="rounded p-1 transition-colors hover:surface-hover hover:text-primary-600 dark:hover:text-primary-400 surface-control surface-muted"
            :title="t('admin.ops.errorLog.details')"
            @click.stop="emit('openErrorDetail', row.id)"
          >
            <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
          </button>
        </template>

        <template #empty><EmptyState :message="t('admin.ops.errorLog.noErrors')" /></template>
      </DataTable>
    </div>

    <div class="flex-shrink-0" :class="{ 'workspace-fixed': workspace }">
      <Pagination
        v-if="workspace || total > 0"
        :total="total"
        :page="page"
        :page-size="pageSize"
        @update:page="emit('update:page', $event)"
        @update:pageSize="emit('update:pageSize', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Pagination from '@/components/common/Pagination.vue'
import IpGeoCell from '@/components/common/IpGeoCell.vue'
import IpGeoBatchToolbar from '@/components/common/IpGeoBatchToolbar.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import type { OpsErrorLog } from '@/api/admin/ops'
import type { Column } from '@/components/common/types'
import { getSeverityClass, formatDateTime } from '../utils/opsFormatters'
import { diagnoseOpsError } from '../utils/opsErrorDiagnosis'
import { mapErrorCategory } from '@/utils/errorCategory'
import { mapErrorSortKey, statusCodeBadgeClass } from '@/utils/errorBadges'

const { t, locale } = useI18n()
const isZhLocale = computed(() => String(locale.value || 'zh').startsWith('zh'))
function rowDiagnosis(row: OpsErrorLog) {
  return diagnoseOpsError(row, isZhLocale.value)
}
const appStore = useAppStore()
const copiedUserAgent = ref<string | null>(null)

const copyUserAgent = async (userAgent: string) => {
  try {
    await navigator.clipboard.writeText(userAgent)
    copiedUserAgent.value = userAgent
    appStore.showSuccess(t('admin.usage.userAgentCopied'))
    window.setTimeout(() => {
      if (copiedUserAgent.value === userAgent) copiedUserAgent.value = null
    }, 2000)
  } catch {
    appStore.showError(t('common.copyFailed'))
  }
}

// 列序对齐管理端用量明细:身份(用户→Key→账号)→ 请求形态(平台→模型→端点→分组→类型)
// → 结果(状态→消息)→ 时间→UA→IP→操作
const allColumns = computed<Column[]>(() => [
  { key: 'user', label: t('admin.ops.errorLog.user') },
  { key: 'api_key', label: t('admin.ops.errorLog.apiKey') },
  { key: 'account', label: t('admin.ops.errorLog.account') },
  { key: 'platform', label: t('admin.ops.errorLog.platform') },
  { key: 'model', label: t('admin.ops.errorLog.model'), sortable: true },
  { key: 'endpoint', label: t('admin.ops.errorLog.endpoint') },
  { key: 'group', label: t('admin.ops.errorLog.group') },
  { key: 'type', label: t('admin.ops.errorLog.type') },
  { key: 'category', label: t('usage.errors.category') },
  { key: 'status', label: t('admin.ops.errorLog.status'), sortable: true },
  { key: 'message', label: t('admin.ops.errorLog.message') },
  { key: 'created_at', label: t('admin.ops.errorLog.time'), sortable: true },
  { key: 'user_agent', label: t('usage.userAgent') },
  { key: 'client_ip', label: t('admin.ops.errorLog.ip') },
  { key: 'actions', label: t('admin.ops.errorLog.action') },
])

// 传入 visibleColumnKeys 时按其过滤(列设置);未传则全量(Ops 弹窗等使用方)
const columns = computed<Column[]>(() => {
  const visibleColumns = props.visibleColumnKeys
    ? allColumns.value.filter((c) => props.visibleColumnKeys!.includes(c.key))
    : allColumns.value
  if (props.compact) return [
    ...visibleColumns.filter(column => column.key === 'created_at'),
    ...visibleColumns.filter(column => column.key !== 'created_at'),
  ]
  if (!props.summaryFirst) return visibleColumns

  return [
    ...visibleColumns.filter((c) => c.key === 'created_at'),
    ...visibleColumns.filter((c) => c.key === 'message'),
    ...visibleColumns.filter((c) => c.key !== 'created_at' && c.key !== 'message'),
  ]
})

const compactWidths: Record<string, number> = { created_at: 124, user: 140, account: 140, model: 160, category: 100, status: 120, message: 240 }
const compactTableWidth = computed(() => columns.value.reduce((width, column) => width + (compactWidths[column.key] ?? 160), 0))

function hasModelMapping(log: OpsErrorLog): boolean {
  const requested = String(log.requested_model || '').trim()
  const upstream = String(log.upstream_model || '').trim()
  return !!requested && !!upstream && requested !== upstream
}

function displayModel(log: OpsErrorLog): string {
  const upstream = String(log.upstream_model || '').trim()
  if (upstream) return upstream
  const requested = String(log.requested_model || '').trim()
  if (requested) return requested
  return String(log.model || '').trim()
}

function formatRequestType(type: number | null | undefined): string {
  switch (type) {
    case 1: return t('admin.ops.errorLog.requestTypeSync')
    case 2: return t('admin.ops.errorLog.requestTypeStream')
    case 3: return t('admin.ops.errorLog.requestTypeWs')
    default: return ''
  }
}

interface Props {
  rows: OpsErrorLog[]
  total: number
  loading: boolean
  page: number
  pageSize: number
  /** 用户邮箱可点击(emit userClick),仅在有弹窗承接的使用方开启 */
  userClickable?: boolean
  /** 列设置:仅显示这些 key 的列;不传则全量 */
  visibleColumnKeys?: string[]
  /** 运维弹窗优先展示时间和响应内容 */
  summaryFirst?: boolean
  /** Keep filters and pagination outside the bounded table viewport. */
  workspace?: boolean
  compact?: boolean
  activeRowKey?: number
}

interface Emits {
  (e: 'openErrorDetail', id: number): void
  (e: 'update:page', value: number): void
  (e: 'update:pageSize', value: number): void
  (e: 'ipGeoBatchFailed'): void
  (e: 'sort', sortBy: string, sortOrder: 'asc' | 'desc'): void
  (e: 'userClick', userId: number, email?: string): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

function onSort(key: string, order: 'asc' | 'desc') {
  emit('sort', mapErrorSortKey(key), order)
}

const getStatusClass = statusCodeBadgeClass

function formatSmartMessage(msg: string): string {
  if (!msg) return ''

  if (msg.startsWith('{') || msg.startsWith('[')) {
    try {
      const obj = JSON.parse(msg)
      if (obj?.error?.message) return String(obj.error.message)
      if (obj?.message) return String(obj.message)
      if (obj?.detail) return String(obj.detail)
      if (typeof obj === 'object') return JSON.stringify(obj).substring(0, 150)
    } catch {
      // ignore parse error
    }
  }

  if (msg.includes('context deadline exceeded')) return t('admin.ops.errorLog.commonErrors.contextDeadlineExceeded')
  if (msg.includes('connection refused')) return t('admin.ops.errorLog.commonErrors.connectionRefused')
  if (msg.toLowerCase().includes('rate limit')) return t('admin.ops.errorLog.commonErrors.rateLimit')

  return msg.length > 200 ? msg.substring(0, 200) + '...' : msg
}
</script>

<style scoped>
.error-compact :deep(table) { table-layout: fixed; min-width: var(--error-table-width); }
.error-compact :deep(th), .error-compact :deep(td) { width: 160px; padding-inline: 10px; white-space: normal; overflow-wrap: anywhere; }
.error-compact :deep([data-field="created_at"]) { width: 124px; }
.error-compact :deep([data-field="user"]), .error-compact :deep([data-field="account"]) { width: 140px; }
.error-compact :deep([data-field="category"]) { width: 100px; }
.error-compact :deep([data-field="status"]) { width: 120px; }
.error-compact :deep([data-field="message"]) { width: 240px; }
.error-compact :deep(td [class*="max-w-"]) { max-width: 100%; }
@media (max-width: 767px) { .error-compact :deep([data-field]) { width: auto; padding-inline: 0; } }
</style>

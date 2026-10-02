<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-full md:w-80">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              v-model="filters.search"
              type="text"
              class="input pl-10"
              :placeholder="t('admin.agents.searchPlaceholder')"
              @input="debounceLoad"
            />
          </div>
          <button
            class="btn btn-secondary px-2 md:px-3"
            :disabled="loading"
            :title="t('common.refresh')"
            @click="loadApplications"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="applications" :loading="loading" row-key="user_id">
          <template #cell-user="{ row }">
            <UserCell
              :id="row.user_id"
              :email="row.email"
              :username="row.username"
              :clickable="true"
              @open="openUser"
            />
          </template>
          <template #cell-status="{ row }">
            <span
              class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
              :class="statusClass(row.status)"
            >
              {{ t(`admin.agents.status.${row.status}`) }}
            </span>
          </template>
          <template #cell-source="{ row }">
            {{ t(`admin.agents.source.${row.source}`) }}
          </template>
          <template #cell-created_at="{ row }">
            {{ formatDateTime(row.created_at) || '—' }}
          </template>
          <template #cell-applied_at="{ row }">
            {{ formatDateTime(row.applied_at) || '—' }}
          </template>
          <template #cell-reviewed_at="{ row }">{{ formatDateTime(row.reviewed_at) || '—' }}</template>
        </DataTable>

        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
        />
      </template>
    </TablePageLayout>

  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, onBeforeUnmount, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { agentsAPI, type AgentApplication, type AgentStatus } from '@/api/admin/agents'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const loading = ref(false)
const applications = ref<AgentApplication[]>([])
const filters = reactive({ search: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
let debounceTimer: ReturnType<typeof setTimeout> | null = null

const columns = computed<Column[]>(() => [
  { key: 'user', label: t('admin.agents.columns.user') },
  { key: 'status', label: t('admin.agents.columns.status') },
  { key: 'source', label: t('admin.agents.columns.source') },
  { key: 'created_at', label: t('admin.agents.columns.registeredAt') },
  { key: 'applied_at', label: t('admin.agents.columns.appliedAt') },
  { key: 'reviewed_at', label: t('admin.agents.columns.activatedAt') },
])

function statusClass(_status: AgentStatus): string {
 return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
}

async function loadApplications(): Promise<void> {
  loading.value = true
  try {
    const res = await agentsAPI.listApplications({
      page: pagination.page,
      page_size: pagination.page_size,
      search: filters.search.trim(),
    })
    applications.value = res.items || []
    pagination.total = res.total || 0
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.agents.errors', t('admin.agents.errors.loadFailed')))
  } finally {
    loading.value = false
  }
}

function debounceLoad(): void {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => reloadFromFirstPage(), 300)
}

function reloadFromFirstPage(): void {
  pagination.page = 1
  void loadApplications()
}

function handlePageChange(page: number): void {
  pagination.page = page
  void loadApplications()
}

function openUser(userId: number): void {
  void router.push(`/admin/users?highlight=${userId}`)
}

// Rendered identically to the affiliate records table so reviewer identity reads
// the same way across the affiliate admin pages.
const UserCell = defineComponent({
  props: {
    id: { type: Number, required: true },
    email: { type: String, default: '' },
    username: { type: String, default: '' },
    clickable: { type: Boolean, default: false },
  },
  emits: ['open'],
  setup(cellProps, { emit }) {
    return () =>
      h('div', { class: 'space-y-0.5' }, [
        h('div', { class: 'font-mono text-sm text-gray-900 dark:text-white' }, `#${cellProps.id}`),
        h(
          cellProps.clickable ? 'button' : 'div',
          {
            class: cellProps.clickable
              ? 'max-w-56 truncate text-left text-sm font-medium text-primary-600 hover:text-primary-700 hover:underline dark:text-primary-400 dark:hover:text-primary-300'
              : 'max-w-56 truncate text-sm text-gray-700 dark:text-gray-300',
            type: cellProps.clickable ? 'button' : undefined,
            onClick: cellProps.clickable ? () => emit('open', cellProps.id) : undefined,
          },
          cellProps.email || '-',
        ),
        h('div', { class: 'max-w-56 truncate text-sm text-gray-500 dark:text-dark-400' }, cellProps.username || '-'),
      ])
  },
})

onMounted(() => {
  void loadApplications()
})
onBeforeUnmount(() => { if (debounceTimer) clearTimeout(debounceTimer) })
</script>

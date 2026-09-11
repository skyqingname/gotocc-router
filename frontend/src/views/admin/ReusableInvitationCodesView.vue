<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center justify-end gap-2">
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading"
            :title="t('common.refresh')"
            @click="loadCodes"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button
            data-test="open-create-reusable-code"
            type="button"
            class="btn btn-primary"
            @click="openCreateDialog"
          >
            <Icon name="plus" size="sm" />
            <span>{{ t('admin.reusableInvitationCodes.create') }}</span>
          </button>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="codes"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="id"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-code="{ value }">
            <div class="flex min-w-48 items-center gap-2">
              <code class="font-mono text-sm font-medium text-gray-900 dark:text-gray-100">{{ value }}</code>
              <button
                type="button"
                class="rounded p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200"
                :title="t('keys.copyToClipboard')"
                @click="copyToClipboard(value)"
              >
                <Icon name="copy" size="sm" />
              </button>
            </div>
          </template>

          <template #cell-status="{ value }">
            <span :class="['badge', value === 'active' ? 'badge-success' : 'badge-gray']">
              {{ t(`admin.reusableInvitationCodes.status.${value}`) }}
            </span>
          </template>

          <template #cell-max_uses="{ value }">
            {{ value === 0 ? t('admin.reusableInvitationCodes.unlimited') : value }}
          </template>

          <template #cell-expires_at="{ value }">
            <span class="whitespace-nowrap text-gray-500 dark:text-dark-400">
              {{ value ? formatDateTime(value) : t('admin.reusableInvitationCodes.neverExpires') }}
            </span>
          </template>

          <template #cell-notes="{ value }">
            <span class="block max-w-64 truncate text-gray-500 dark:text-dark-400" :title="value || undefined">
              {{ value || '-' }}
            </span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="whitespace-nowrap text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-2 whitespace-nowrap">
              <button type="button" class="btn btn-secondary btn-sm" @click="openUsesDialog(row)">
                <Icon name="eye" size="sm" />
                <span>{{ t('admin.reusableInvitationCodes.uses') }}</span>
              </button>
              <button
                v-if="row.status === 'active'"
                data-test="disable-reusable-code"
                type="button"
                class="btn btn-danger btn-sm"
                @click="openDisableDialog(row)"
              >
                <Icon name="xCircle" size="sm" />
                <span>{{ t('admin.reusableInvitationCodes.disable') }}</span>
              </button>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          :total="pagination.total"
          :page="pagination.page"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showCreateDialog"
      :title="t('admin.reusableInvitationCodes.createTitle')"
      width="normal"
      @close="closeCreateDialog"
    >
      <form data-test="create-reusable-code-form" class="space-y-4" @submit.prevent="handleCreate">
        <div>
          <label for="reusable-code" class="input-label">
            {{ t('admin.reusableInvitationCodes.columns.code') }}
          </label>
          <input
            id="reusable-code"
            v-model.trim="createForm.code"
            data-test="reusable-code-input"
            type="text"
            class="input font-mono"
            minlength="3"
            maxlength="64"
            autocomplete="off"
            required
          />
        </div>
        <div>
          <label for="reusable-code-max-uses" class="input-label">
            {{ t('admin.reusableInvitationCodes.columns.maxUses') }}
          </label>
          <input
            id="reusable-code-max-uses"
            v-model.number="createForm.max_uses"
            type="number"
            min="0"
            class="input"
          />
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.reusableInvitationCodes.maxUsesHint') }}
          </p>
        </div>
        <div>
          <label for="reusable-code-expires-at" class="input-label">
            {{ t('admin.reusableInvitationCodes.columns.expiresAt') }}
          </label>
          <input
            id="reusable-code-expires-at"
            v-model="createForm.expires_at_local"
            type="datetime-local"
            class="input"
          />
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.reusableInvitationCodes.expiresHint') }}
          </p>
        </div>
        <div>
          <label for="reusable-code-notes" class="input-label">
            {{ t('admin.reusableInvitationCodes.columns.notes') }}
          </label>
          <textarea
            id="reusable-code-notes"
            v-model.trim="createForm.notes"
            rows="3"
            class="input resize-y"
          ></textarea>
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="btn btn-secondary" @click="closeCreateDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" class="btn btn-primary" :disabled="creating">
            <Icon v-if="!creating" name="plus" size="sm" />
            <span>{{ creating ? t('common.submitting') : t('common.create') }}</span>
          </button>
        </div>
      </form>
    </BaseDialog>

    <BaseDialog
      :show="showUsesDialog"
      :title="t('admin.reusableInvitationCodes.usesTitle')"
      width="wide"
      @close="closeUsesDialog"
    >
      <div v-if="usesLoading" class="py-8 text-center text-sm text-gray-500">
        {{ t('common.loading') }}
      </div>
      <div v-else-if="uses.length === 0" class="py-8 text-center text-sm text-gray-500">
        {{ t('admin.reusableInvitationCodes.noUses') }}
      </div>
      <div v-else class="max-h-[60vh] overflow-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-600">
          <thead class="sticky top-0 bg-gray-50 dark:bg-dark-800">
            <tr class="text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
              <th class="px-3 py-2">{{ t('admin.reusableInvitationCodes.useColumns.userId') }}</th>
              <th class="px-3 py-2">{{ t('admin.reusableInvitationCodes.useColumns.email') }}</th>
              <th class="px-3 py-2">{{ t('admin.reusableInvitationCodes.useColumns.source') }}</th>
              <th class="px-3 py-2">{{ t('admin.reusableInvitationCodes.useColumns.usedAt') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="use in uses" :key="use.id" class="text-sm text-gray-700 dark:text-gray-300">
              <td class="whitespace-nowrap px-3 py-2 font-mono">{{ use.user_id }}</td>
              <td class="px-3 py-2">{{ use.email || '-' }}</td>
              <td class="whitespace-nowrap px-3 py-2">{{ use.auth_source || '-' }}</td>
              <td class="whitespace-nowrap px-3 py-2">{{ formatDateTime(use.used_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </BaseDialog>

    <ConfirmDialog
      :show="showDisableDialog"
      :title="t('admin.reusableInvitationCodes.disableTitle')"
      :message="t('admin.reusableInvitationCodes.disableConfirm')"
      :confirm-text="t('admin.reusableInvitationCodes.disable')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="confirmDisable"
      @cancel="closeDisableDialog"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI, type ReusableInvitationCode, type ReusableInvitationCodeUse } from '@/api/admin'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import type { Column } from '@/components/common/types'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const codes = ref<ReusableInvitationCode[]>([])
const uses = ref<ReusableInvitationCodeUse[]>([])
const loading = ref(false)
const creating = ref(false)
const usesLoading = ref(false)
const showCreateDialog = ref(false)
const showDisableDialog = ref(false)
const showUsesDialog = ref(false)
const disablingCode = ref<ReusableInvitationCode | null>(null)

const pagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0 })
const sortState = reactive({ sort_by: 'id', sort_order: 'desc' as 'asc' | 'desc' })
const createForm = reactive({ code: '', max_uses: 0, expires_at_local: '', notes: '' })

const columns = computed<Column[]>(() => [
  { key: 'code', label: t('admin.reusableInvitationCodes.columns.code') },
  { key: 'status', label: t('admin.reusableInvitationCodes.columns.status'), sortable: true },
  { key: 'max_uses', label: t('admin.reusableInvitationCodes.columns.maxUses'), sortable: true },
  { key: 'used_count', label: t('admin.reusableInvitationCodes.columns.usedCount'), sortable: true },
  { key: 'expires_at', label: t('admin.reusableInvitationCodes.columns.expiresAt'), sortable: true },
  { key: 'notes', label: t('admin.reusableInvitationCodes.columns.notes') },
  { key: 'created_at', label: t('admin.reusableInvitationCodes.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('admin.reusableInvitationCodes.columns.actions') }
])

async function loadCodes() {
  loading.value = true
  try {
    const response = await adminAPI.reusableInvitationCodes.list(
      pagination.page,
      pagination.page_size,
      sortState
    )
    codes.value = response.items
    pagination.total = response.total
  } catch (error) {
    appStore.showError(t('admin.reusableInvitationCodes.loadFailed'))
    console.error('Error loading reusable invitation codes:', error)
  } finally {
    loading.value = false
  }
}

function resetCreateForm() {
  Object.assign(createForm, { code: '', max_uses: 0, expires_at_local: '', notes: '' })
}

function openCreateDialog() {
  resetCreateForm()
  showCreateDialog.value = true
}

function closeCreateDialog() {
  showCreateDialog.value = false
}

async function handleCreate() {
  const code = createForm.code.trim()
  if (!code) return

  creating.value = true
  try {
    await adminAPI.reusableInvitationCodes.create({
      code,
      max_uses: createForm.max_uses || 0,
      expires_at: createForm.expires_at_local
        ? new Date(createForm.expires_at_local).toISOString()
        : null,
      notes: createForm.notes.trim()
    })
    appStore.showSuccess(t('admin.reusableInvitationCodes.createSuccess'))
    showCreateDialog.value = false
    pagination.page = 1
    await loadCodes()
  } catch (error) {
    appStore.showError(t('admin.reusableInvitationCodes.createFailed'))
    console.error('Error creating reusable invitation code:', error)
  } finally {
    creating.value = false
  }
}

function openDisableDialog(code: ReusableInvitationCode) {
  disablingCode.value = code
  showDisableDialog.value = true
}

function closeDisableDialog() {
  showDisableDialog.value = false
  disablingCode.value = null
}

async function confirmDisable() {
  const code = disablingCode.value
  if (!code) return

  try {
    await adminAPI.reusableInvitationCodes.disable(code.id)
    appStore.showSuccess(t('admin.reusableInvitationCodes.disableSuccess'))
    closeDisableDialog()
    await loadCodes()
  } catch (error) {
    appStore.showError(t('admin.reusableInvitationCodes.disableFailed'))
    console.error('Error disabling reusable invitation code:', error)
  }
}

async function openUsesDialog(code: ReusableInvitationCode) {
  showUsesDialog.value = true
  usesLoading.value = true
  uses.value = []
  try {
    uses.value = await adminAPI.reusableInvitationCodes.listUses(code.id, 50)
  } catch (error) {
    appStore.showError(t('admin.reusableInvitationCodes.usesLoadFailed'))
    console.error('Error loading reusable invitation code uses:', error)
  } finally {
    usesLoading.value = false
  }
}

function closeUsesDialog() {
  showUsesDialog.value = false
  uses.value = []
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadCodes()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadCodes()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  void loadCodes()
}

onMounted(() => void loadCodes())
</script>

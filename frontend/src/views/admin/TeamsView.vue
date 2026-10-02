<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="relative w-full sm:w-80">
          <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input v-model="searchQuery" class="input pl-10" :placeholder="t('team.searchPlaceholder')" />
        </div>
      </template>
      <template #actions>
        <div class="flex items-center justify-end gap-2">
          <button class="btn btn-secondary px-3" :disabled="loading" :title="t('common.refresh')" @click="loadTeams">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          </button>
          <button class="btn btn-primary" @click="showCreate = true"><Icon name="plus" size="sm" />{{ t('team.create') }}</button>
        </div>
      </template>
      <template #table>
        <DataTable :columns="columns" :data="paginatedTeams" :loading="loading" row-key="id" :actions-count="4" default-sort-key="created_at" default-sort-order="desc">
          <template #cell-id="{ value }"><span class="font-mono text-xs text-gray-500">#{{ value }}</span></template>
          <template #cell-name="{ value }"><span class="font-medium text-gray-900 dark:text-white">{{ value }}</span></template>
          <template #cell-owner_email="{ value }"><span class="block max-w-72 truncate text-gray-700 dark:text-gray-300" :title="value">{{ value }}</span></template>
          <template #cell-member_count="{ row }"><span class="tabular-nums text-gray-700 dark:text-gray-300">{{ row.member_count }} / {{ row.member_limit }}</span></template>
          <template #cell-status="{ value }"><span class="badge" :class="value === 'active' ? 'badge-success' : 'badge-danger'">{{ value === 'active' ? t('team.statusActive') : t('team.statusSuspended') }}</span></template>
          <template #cell-created_at="{ value }"><span class="whitespace-nowrap text-gray-500">{{ formatDateTime(value) }}</span></template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button class="row-action" :title="t('team.viewDetails')" @click="openDetails(row)"><Icon name="eye" size="sm" /><span>{{ t('team.viewDetails') }}</span></button>
              <button class="row-action" :title="t('team.viewStatistics')" @click="openStatistics(row)"><Icon name="chart" size="sm" /><span>{{ t('team.viewStatistics') }}</span></button>
              <button class="row-action" :title="t('common.edit')" @click="openEdit(row)"><Icon name="edit" size="sm" /><span>{{ t('common.edit') }}</span></button>
              <button class="row-action" :disabled="statusUpdatingID === row.id" :title="row.status === 'active' ? t('team.pause') : t('team.resume')" @click="toggleStatus(row)"><Icon :name="row.status === 'active' ? 'ban' : 'play'" size="sm" /><span>{{ row.status === 'active' ? t('team.pause') : t('team.resume') }}</span></button>
              <button class="row-action text-red-600 dark:text-red-400" :title="t('team.dissolve')" @click="dissolvingTeam = row"><Icon name="trash" size="sm" /><span>{{ t('team.dissolve') }}</span></button>
            </div>
          </template>
          <template #empty>
            <EmptyState :title="t('team.noTeams')" :description="t('team.noTeamsDescription')" :action-text="t('team.create')" @action="showCreate = true" />
          </template>
        </DataTable>
      </template>
      <template #pagination>
        <Pagination v-if="filteredTeams.length" :page="page" :total="filteredTeams.length" :page-size="pageSize" @update:page="page = $event" @update:page-size="handlePageSizeChange" />
      </template>
    </TablePageLayout>

    <BaseDialog :show="showCreate" :title="t('team.createTitle')" width="narrow" @close="showCreate = false">
      <form id="admin-team-create" class="space-y-4" @submit.prevent="createTeam">
        <div><label class="input-label">{{ t('team.name') }}</label><input v-model.trim="createForm.name" class="input" required maxlength="100" /></div>
        <div><label class="input-label">{{ t('team.ownerUserID') }}</label><input v-model.number="createForm.owner_user_id" class="input" type="number" min="1" required /></div>
        <div><label class="input-label">{{ t('team.memberCapacity') }}</label><input v-model.number="createForm.member_limit" class="input" type="number" min="0" required /></div>
      </form>
      <template #footer><div class="flex justify-end gap-3"><button class="btn btn-secondary" @click="showCreate = false">{{ t('common.cancel') }}</button><button form="admin-team-create" class="btn btn-primary" :disabled="saving">{{ t('common.create') }}</button></div></template>
    </BaseDialog>

    <BaseDialog :show="Boolean(editingTeam)" :title="t('team.editTitle')" width="narrow" @close="editingTeam = null">
      <form id="admin-team-edit" class="space-y-4" @submit.prevent="saveEdit">
        <div><label class="input-label">{{ t('team.name') }}</label><input v-model.trim="editForm.name" class="input" required maxlength="100" /></div>
        <div><label class="input-label">{{ t('team.memberCapacity') }}</label><input v-model.number="editForm.member_limit" class="input" type="number" min="0" required /><p class="mt-1.5 text-xs text-gray-500">{{ t('team.memberCapacityDescription') }}</p></div>
      </form>
      <template #footer><div class="flex justify-end gap-3"><button class="btn btn-secondary" @click="editingTeam = null">{{ t('common.cancel') }}</button><button form="admin-team-edit" class="btn btn-primary" :disabled="saving">{{ t('common.save') }}</button></div></template>
    </BaseDialog>

    <BaseDialog :show="Boolean(detailsTeam)" :title="t('team.detailsTitle', { name: detailsTeam?.name || '' })" width="wide" @close="closeDetails">
      <div v-if="detailsLoading" class="flex justify-center py-16"><LoadingSpinner /></div>
      <div v-else-if="detailsTeam" class="space-y-6">
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div class="metric"><span>{{ t('team.owner') }}</span><strong :title="detailsTeam.owner_email">{{ detailsTeam.owner_email }}</strong></div>
          <div class="metric"><span>{{ t('team.members') }}</span><strong>{{ detailsTeam.member_count }} / {{ detailsTeam.member_limit }}</strong></div>
          <div class="metric"><span>{{ t('common.status') }}</span><strong>{{ detailsTeam.status === 'active' ? t('team.statusActive') : t('team.statusSuspended') }}</strong></div>
          <div class="metric"><span>{{ t('team.createdAt') }}</span><strong>{{ formatDateTime(detailsTeam.created_at) }}</strong></div>
        </div>
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.members') }}</h3>
          <div class="mt-3 divide-y divide-gray-100 overflow-hidden rounded-md border border-gray-200 dark:divide-dark-700 dark:border-dark-700">
            <div v-for="member in detailsMembers" :key="member.user_id" class="flex items-center justify-between gap-3 px-4 py-3">
              <div class="min-w-0"><p class="truncate font-medium text-gray-900 dark:text-white">{{ member.username || member.email }}</p><p class="truncate text-xs text-gray-500">{{ member.email }}</p></div>
              <span class="badge" :class="member.role === 'owner' ? 'badge-primary' : 'badge-gray'">{{ member.role === 'owner' ? t('team.owner') : t('team.member') }}</span>
            </div>
          </div>
        </div>
        <div v-if="memberOptions.length" class="border-t border-gray-200 pt-5 dark:border-dark-700">
          <label class="input-label">{{ t('team.transferTitle') }}</label>
          <div class="flex flex-col gap-3 sm:flex-row"><Select v-model="transferUserID" class="flex-1" :options="memberOptions" /><button class="btn btn-secondary" :disabled="!transferUserID" @click="forceTransfer"><Icon name="swap" size="sm" />{{ t('team.transfer') }}</button></div>
        </div>
      </div>
      <template #footer><div class="flex justify-end"><button class="btn btn-secondary" @click="closeDetails">{{ t('common.close') }}</button></div></template>
    </BaseDialog>

    <BaseDialog :show="Boolean(statisticsTeam)" :title="t('team.statisticsTitle', { name: statisticsTeam?.name || '' })" width="extra-wide" @close="statisticsTeam = null">
      <div v-if="statisticsLoading" class="flex justify-center py-16"><LoadingSpinner /></div>
      <div v-else class="space-y-5">
        <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <div v-for="metric in statisticsMetrics" :key="metric.label" class="metric"><span>{{ metric.label }}</span><strong>{{ metric.value }}</strong></div>
        </div>
        <TeamMemberUsageCharts :series="statisticsSeries" />
      </div>
      <template #footer><div class="flex justify-end"><button class="btn btn-secondary" @click="statisticsTeam = null">{{ t('common.close') }}</button></div></template>
    </BaseDialog>

    <ConfirmDialog :show="Boolean(dissolvingTeam)" :title="t('team.dissolveTitle')" :message="t('team.dissolveMessage')" danger @cancel="dissolvingTeam = null" @confirm="dissolveTeam" />
    <TotpStepUpDialog :controller="stepUp" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AdminTeam } from '@/api/admin/teams'
import type { TeamMembership, TeamUsageSummary } from '@/api/team'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import TeamMemberUsageCharts from '@/components/charts/TeamMemberUsageCharts.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import type { Column } from '@/components/common/types'
import { useStepUp, isStepUpCancelled } from '@/composables/useStepUp'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const stepUp = useStepUp()
const teams = ref<AdminTeam[]>([])
const loading = ref(false)
const saving = ref(false)
const statusUpdatingID = ref<number | null>(null)
const searchQuery = ref('')
const page = ref(1)
const pageSize = ref(getPersistedPageSize())
const showCreate = ref(false)
const editingTeam = ref<AdminTeam | null>(null)
const detailsTeam = ref<AdminTeam | null>(null)
const detailsMembers = ref<TeamMembership[]>([])
const detailsLoading = ref(false)
const transferUserID = ref<number | null>(null)
const statisticsTeam = ref<AdminTeam | null>(null)
const statisticsLoading = ref(false)
const statisticsSummary = ref<TeamUsageSummary | null>(null)
const statisticsSeries = ref<Array<{ userID: number; label: string; summary: TeamUsageSummary }>>([])
const dissolvingTeam = ref<AdminTeam | null>(null)
const createForm = reactive({ owner_user_id: 0, name: '', member_limit: 10 })
const editForm = reactive({ name: '', member_limit: 10 })
let detailsSequence = 0
let statisticsSequence = 0

const columns = computed<Column[]>(() => [
  { key: 'id', label: 'ID', sortable: true },
  { key: 'name', label: t('team.name'), sortable: true },
  { key: 'owner_email', label: t('team.owner'), sortable: true },
  { key: 'member_count', label: t('team.members'), sortable: true },
  { key: 'status', label: t('common.status'), sortable: true },
  { key: 'created_at', label: t('team.createdAt'), sortable: true },
  { key: 'actions', label: t('common.actions') },
])
const filteredTeams = computed(() => { const query = searchQuery.value.trim().toLowerCase(); return query ? teams.value.filter((team) => team.name.toLowerCase().includes(query) || team.owner_email.toLowerCase().includes(query) || String(team.id).includes(query)) : teams.value })
const paginatedTeams = computed(() => filteredTeams.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const memberOptions = computed<SelectOption[]>(() => detailsMembers.value.filter((member) => member.role === 'member').map((member) => ({ value: member.user_id, label: member.username || member.email })))
const statisticsMetrics = computed(() => [
  { label: t('team.totalCost'), value: `$${Number(statisticsSummary.value?.actual_cost || 0).toFixed(4)}` },
  { label: t('team.requests'), value: Number(statisticsSummary.value?.request_count || 0).toLocaleString() },
  { label: t('team.inputTokens'), value: Number(statisticsSummary.value?.input_tokens || 0).toLocaleString() },
  { label: t('team.outputTokens'), value: Number(statisticsSummary.value?.output_tokens || 0).toLocaleString() },
])

watch(searchQuery, () => { page.value = 1 })
watch(filteredTeams, (items) => { page.value = Math.min(page.value, Math.max(1, Math.ceil(items.length / pageSize.value))) })
const loadTeams = async () => { loading.value = true; try { teams.value = await adminAPI.teams.list() } catch (error: any) { appStore.showError(error?.message || t('common.error')) } finally { loading.value = false } }
const handlePageSizeChange = (value: number) => { pageSize.value = value; page.value = 1 }
const createTeam = async () => { saving.value = true; try { await adminAPI.teams.create({ ...createForm }); showCreate.value = false; createForm.owner_user_id = 0; createForm.name = ''; await loadTeams(); appStore.showSuccess(t('team.created')) } catch (error: any) { appStore.showError(error?.message || t('common.error')) } finally { saving.value = false } }
const openEdit = (team: AdminTeam) => { editingTeam.value = team; editForm.name = team.name; editForm.member_limit = team.member_limit }
const saveEdit = async () => { if (!editingTeam.value) return; saving.value = true; try { await adminAPI.teams.update(editingTeam.value.id, { name: editForm.name, member_limit: editForm.member_limit }); editingTeam.value = null; await loadTeams(); appStore.showSuccess(t('team.updated')) } catch (error: any) { appStore.showError(error?.message || t('common.error')) } finally { saving.value = false } }
const toggleStatus = async (team: AdminTeam) => { statusUpdatingID.value = team.id; try { await adminAPI.teams.update(team.id, { status: team.status === 'active' ? 'suspended' : 'active' }); await loadTeams(); appStore.showSuccess(t('team.operationSuccess')) } catch (error: any) { appStore.showError(error?.message || t('common.error')) } finally { statusUpdatingID.value = null } }
const openDetails = async (team: AdminTeam) => { const sequence = ++detailsSequence; detailsTeam.value = team; detailsMembers.value = []; transferUserID.value = null; detailsLoading.value = true; try { const members = await adminAPI.teams.members(team.id); if (sequence === detailsSequence) detailsMembers.value = members } catch (error: any) { if (sequence === detailsSequence) appStore.showError(error?.message || t('common.error')) } finally { if (sequence === detailsSequence) detailsLoading.value = false } }
const closeDetails = () => { detailsSequence++; detailsTeam.value = null; detailsMembers.value = []; transferUserID.value = null }
const openStatistics = async (team: AdminTeam) => { const sequence = ++statisticsSequence; statisticsTeam.value = team; statisticsLoading.value = true; statisticsSummary.value = null; statisticsSeries.value = []; try { const [members, summary] = await Promise.all([adminAPI.teams.members(team.id), adminAPI.teams.usage(team.id)]); const memberSummaries = await Promise.all(members.map((member) => adminAPI.teams.usage(team.id, { member_id: member.user_id }))); if (sequence !== statisticsSequence) return; statisticsSummary.value = summary; statisticsSeries.value = members.map((member, index) => ({ userID: member.user_id, label: member.username || member.email, summary: memberSummaries[index] })) } catch (error: any) { if (sequence === statisticsSequence) appStore.showError(error?.message || t('common.error')) } finally { if (sequence === statisticsSequence) statisticsLoading.value = false } }
const forceTransfer = async () => { if (!detailsTeam.value || !transferUserID.value) return; try { await stepUp.run(() => adminAPI.teams.forceTransfer(detailsTeam.value!.id, transferUserID.value!)); const id = detailsTeam.value.id; await loadTeams(); await openDetails(teams.value.find((team) => team.id === id) || detailsTeam.value); appStore.showSuccess(t('team.operationSuccess')) } catch (error: any) { if (!isStepUpCancelled(error)) appStore.showError(error?.message || t('common.error')) } }
const dissolveTeam = async () => { const team = dissolvingTeam.value; dissolvingTeam.value = null; if (!team) return; try { await stepUp.run(() => adminAPI.teams.dissolve(team.id)); await loadTeams(); appStore.showSuccess(t('team.operationSuccess')) } catch (error: any) { if (!isStepUpCancelled(error)) appStore.showError(error?.message || t('common.error')) } }

onMounted(loadTeams)
</script>

<style scoped>
.row-action { @apply flex flex-col items-center gap-0.5 rounded-md p-1.5 text-xs text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-dark-700 dark:hover:text-white; }
.metric { @apply min-w-0 rounded-md border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-800; }
.metric span { @apply block text-xs text-gray-500 dark:text-gray-400; }
.metric strong { @apply mt-1 block truncate text-sm font-semibold text-gray-900 dark:text-white; }
</style>

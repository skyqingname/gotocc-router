<template>
  <AppLayout>
    <div class="space-y-6">
      <section v-if="transferToken" class="border-b border-gray-200 pb-5 dark:border-dark-700">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('team.transferActionTitle') }}</h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('team.transferActionDescription') }}</p>
        <div class="mt-4 flex flex-wrap gap-3">
          <button class="btn btn-primary" :disabled="resolvingToken" @click="resolveTransfer('accepted')">
            <Icon name="check" size="sm" />{{ t('team.accept') }}
          </button>
          <button class="btn btn-secondary" :disabled="resolvingToken" @click="resolveTransfer('declined')">
            <Icon name="x" size="sm" />{{ t('team.decline') }}
          </button>
        </div>
      </section>

      <div v-if="loading" class="flex justify-center py-20"><LoadingSpinner /></div>

      <section v-else-if="!teamContext" class="mx-auto max-w-xl py-10">
        <div class="mb-8 text-center">
          <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-md bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-400">
            <Icon name="users" size="xl" />
          </div>
          <h1 class="mt-5 text-2xl font-semibold text-gray-900 dark:text-white">{{ t('team.noTeam') }}</h1>
          <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('team.createDescription') }}</p>
          <button v-if="selfServiceEnabled" class="btn btn-secondary mt-5" type="button" @click="startTeamGuide">
            <Icon name="questionCircle" size="sm" />
            {{ t('team.guideButton') }}
          </button>
        </div>
        <form v-if="selfServiceEnabled" class="space-y-4" data-tour="team-create-form" @submit.prevent="createTeam">
          <div>
            <label class="input-label">{{ t('team.name') }}</label>
            <input v-model.trim="createName" class="input" maxlength="100" required />
          </div>
          <button type="submit" class="btn btn-primary w-full" :disabled="submitting">
            <Icon name="plus" size="sm" />{{ t('team.create') }}
          </button>
        </form>
        <p v-else class="rounded-md border border-gray-200 bg-gray-50 px-4 py-3 text-center text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
          {{ t('team.selfServiceDisabled') }}
        </p>
      </section>

      <template v-else>
        <header class="flex flex-wrap items-start justify-between gap-4">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-3">
              <h1 class="break-words text-2xl font-semibold text-gray-900 dark:text-white">{{ teamContext.team.name }}</h1>
              <span class="badge" :class="teamContext.team.status === 'active' ? 'badge-success' : 'badge-danger'">
                {{ teamContext.team.status === 'active' ? t('team.statusActive') : t('team.statusSuspended') }}
              </span>
            </div>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ isOwner ? t('team.owner') : t('team.member') }} · {{ t('team.memberCount', { count: teamContext.team.member_count + 1 }) }}
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <button class="btn btn-secondary" type="button" @click="startTeamGuide">
              <Icon name="questionCircle" size="sm" />
              {{ t('team.guideButton') }}
            </button>
            <button class="btn btn-secondary px-3" :disabled="refreshing" :title="t('common.refresh')" @click="refreshAll">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': refreshing }" />
            </button>
          </div>
        </header>

        <nav class="flex gap-1 overflow-x-auto border-b border-gray-200 dark:border-dark-700" :aria-label="t('team.title')">
          <button
            v-for="tab in visibleTabs"
            :key="tab.value"
            type="button"
            class="shrink-0 border-b-2 px-4 py-3 text-sm font-medium transition-colors"
            :class="activeTab === tab.value ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'"
            :data-tour="teamTabTourMarker(tab.value)"
            @click="activeTab = tab.value"
          >
            {{ tab.label }}
          </button>
        </nav>

        <section v-if="activeTab === 'overview'" class="space-y-5">
          <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
            <div v-for="metric in usageMetrics" :key="metric.label" class="card min-w-0 p-4">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ metric.label }}</p>
              <p class="mt-2 truncate text-xl font-semibold text-gray-900 dark:text-white">{{ metric.value }}</p>
            </div>
          </div>

          <div v-if="!isOwner" class="card p-5" data-tour="team-limit-progress">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.limitProgress') }}</h2>
            <div class="mt-5 grid gap-5 md:grid-cols-3">
              <div v-for="limit in memberLimits" :key="limit.label">
                <div class="flex items-center justify-between gap-3 text-sm">
                  <span class="text-gray-600 dark:text-gray-300">{{ limit.label }}</span>
                  <span class="font-medium text-gray-900 dark:text-white">{{ limit.display }}</span>
                </div>
                <div class="mt-2 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div class="h-full rounded-full" :class="limit.percent >= 100 ? 'bg-red-500' : limit.percent >= 80 ? 'bg-amber-500' : 'bg-emerald-500'" :style="{ width: `${limit.percent}%` }" />
                </div>
              </div>
            </div>
          </div>

          <TeamMemberUsageCharts v-if="isOwner" :series="memberSeriesForChart" :loading="usageLoading" data-tour="team-member-usage-charts" />

          <div class="card overflow-hidden" data-tour="team-usage-records">
            <div class="flex items-center justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.recentUsage') }}</h2>
              <span class="text-xs text-gray-500">{{ t('team.last30DaysStatistics') }}</span>
            </div>
            <div v-if="usageLogs.length === 0" class="py-12 text-center text-sm text-gray-500">{{ t('team.noUsage') }}</div>
            <div v-else class="overflow-x-auto">
              <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
                <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400">
                  <tr><th class="px-4 py-3">{{ t('team.keyOwner') }}</th><th class="px-4 py-3">{{ t('team.keys') }}</th><th class="px-4 py-3">{{ t('team.model') }}</th><th class="px-4 py-3 text-right">{{ t('team.cost') }}</th><th class="px-4 py-3">{{ t('team.time') }}</th></tr>
                </thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                  <tr v-for="item in usageLogs" :key="item.id">
                    <td class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-gray-300">{{ item.actor_email }}</td>
                    <td class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-gray-300">{{ item.api_key_name }}</td>
                    <td class="max-w-64 truncate px-4 py-3 font-medium text-gray-900 dark:text-white" :title="item.model">{{ item.model }}</td>
                    <td class="whitespace-nowrap px-4 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatMoney(item.actual_cost, 4) }}</td>
                    <td class="whitespace-nowrap px-4 py-3 text-gray-500">{{ formatDateTime(item.created_at) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </section>

        <section v-else-if="activeTab === 'members'" class="card overflow-hidden">
          <div class="flex items-center justify-between gap-4 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.members') }}</h2>
            <span class="text-sm text-gray-500">{{ members.length }}</span>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div v-for="member in members" :key="member.user_id" class="flex flex-col gap-4 p-5 lg:flex-row lg:items-center lg:justify-between">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="truncate font-medium text-gray-900 dark:text-white">{{ member.username || member.email }}</p>
                  <span class="badge" :class="member.role === 'owner' ? 'badge-primary' : 'badge-gray'">{{ member.role === 'owner' ? t('team.owner') : t('team.member') }}</span>
                </div>
                <p class="mt-1 truncate text-sm text-gray-500">{{ member.email }}</p>
                <p v-if="member.role === 'member'" class="mt-2 text-xs text-gray-500">
                  {{ t('team.daily') }} {{ formatLimit(member.daily_usage_usd, member.daily_limit_usd) }} ·
                  {{ t('team.weekly') }} {{ formatLimit(member.weekly_usage_usd, member.weekly_limit_usd) }} ·
                  {{ t('team.monthly') }} {{ formatLimit(member.monthly_usage_usd, member.monthly_limit_usd) }}
                </p>
              </div>
              <div v-if="isOwner && member.role === 'member'" class="flex flex-wrap gap-2">
                <button class="btn btn-secondary btn-sm" @click="openLimitEditor(member)"><Icon name="edit" size="sm" />{{ t('team.editLimits') }}</button>
                <button class="btn btn-secondary btn-sm" @click="askTransfer(member)"><Icon name="swap" size="sm" />{{ t('team.transfer') }}</button>
                <button class="btn btn-danger btn-sm" @click="askRemove(member)"><Icon name="trash" size="sm" />{{ t('team.remove') }}</button>
              </div>
            </div>
          </div>
        </section>

        <section v-else-if="activeTab === 'keys'" class="space-y-4">
          <div class="flex justify-end">
            <RouterLink class="btn btn-primary" :to="{ path: '/keys', query: { scope: 'team' } }"><Icon name="plus" size="sm" />{{ t('team.createKey') }}</RouterLink>
          </div>
          <div class="card overflow-hidden">
            <div v-if="teamKeys.length === 0" class="py-12 text-center text-sm text-gray-500">{{ t('team.noKeys') }}</div>
            <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
              <div v-for="key in teamKeys" :key="key.id" class="flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="font-medium text-gray-900 dark:text-white">{{ key.name }}</p>
                    <span class="badge" :class="key.team_owner_disabled || key.status !== 'active' ? 'badge-gray' : 'badge-success'">{{ key.team_owner_disabled ? t('team.ownerDisabled') : statusLabel(key.status) }}</span>
                  </div>
                  <p class="mt-1 truncate font-mono text-xs text-gray-500">{{ key.masked_key }}</p>
                  <p class="mt-1 text-xs text-gray-500">{{ key.user_email }} · {{ key.group_name || t('team.noGroup') }}</p>
                </div>
                <div class="flex flex-wrap gap-2">
                  <button v-if="key.team_owner_disabled || key.status !== 'active'" class="btn btn-secondary btn-sm" @click="askEnableKey(key)"><Icon name="play" size="sm" />{{ t('team.enable') }}</button>
                  <button v-else class="btn btn-secondary btn-sm" @click="askDisableKey(key)"><Icon name="ban" size="sm" />{{ t('team.disable') }}</button>
                  <button class="btn btn-danger btn-sm" @click="askDeleteKey(key)"><Icon name="trash" size="sm" />{{ t('common.delete') }}</button>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section v-else-if="activeTab === 'invitations'" class="space-y-4">
          <form class="flex flex-col gap-3 sm:flex-row" @submit.prevent="sendInvitation">
            <input v-model.trim="inviteEmail" type="email" class="input flex-1" :placeholder="t('team.inviteEmail')" required />
            <button class="btn btn-primary" :disabled="submitting"><Icon name="mail" size="sm" />{{ t('team.sendInvite') }}</button>
          </form>
          <div class="card divide-y divide-gray-100 dark:divide-dark-700">
            <div v-if="invitations.length === 0" class="py-12 text-center text-sm text-gray-500">{{ t('team.noInvitations') }}</div>
            <div v-for="invitation in invitations" :key="invitation.id" class="flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between">
              <div><p class="font-medium text-gray-900 dark:text-white">{{ invitation.email }}</p><p class="mt-1 text-sm text-gray-500">{{ statusLabel(invitation.status) }} · {{ formatDateTime(invitation.expires_at) }}</p></div>
              <div v-if="invitation.status === 'pending'" class="flex gap-2"><button class="btn btn-secondary btn-sm" @click="reissueInvitation(invitation.id)"><Icon name="refresh" size="sm" />{{ t('team.reissue') }}</button><button class="btn btn-danger btn-sm" @click="revokeInvitation(invitation.id)"><Icon name="x" size="sm" />{{ t('team.revoke') }}</button></div>
            </div>
          </div>
        </section>

        <section v-else class="space-y-5">
          <div v-if="isOwner" class="card p-5">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.name') }}</h2>
            <form class="mt-4 flex flex-col gap-3 sm:flex-row" @submit.prevent="renameTeam"><input v-model.trim="renameName" class="input flex-1" required maxlength="100" /><button class="btn btn-primary" :disabled="submitting">{{ t('team.rename') }}</button></form>
          </div>
          <div v-if="isOwner" class="card p-5">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.defaultMemberLimits') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('team.defaultMemberLimitsDescription') }}</p>
            <form class="mt-5" @submit.prevent="saveDefaultMemberLimits">
              <div class="grid gap-4 md:grid-cols-3"><div v-for="field in defaultLimitFields" :key="field.key"><label class="input-label">{{ field.label }}</label><input v-model.number="defaultLimitForm[field.key]" type="number" min="0" step="0.01" class="input" required /></div></div>
              <div class="mt-5 flex justify-end"><button class="btn btn-primary" :disabled="submitting">{{ t('team.saveDefaultLimits') }}</button></div>
            </form>
          </div>
          <div class="card overflow-hidden">
            <div v-if="isOwner" class="flex flex-col gap-4 px-5 py-5 sm:flex-row sm:items-center sm:justify-between">
              <div><h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('team.teamStatus') }}</h2><p class="mt-1 text-sm text-gray-500">{{ t('team.teamStatusDescription') }}</p></div>
              <button class="btn btn-secondary" @click="askSetStatus"><Icon :name="teamContext.team.status === 'active' ? 'ban' : 'play'" size="sm" />{{ teamContext.team.status === 'active' ? t('team.pause') : t('team.resume') }}</button>
            </div>
            <div class="flex flex-col gap-4 border-t border-gray-200 bg-gray-50/70 px-5 py-5 dark:border-dark-700 dark:bg-dark-800/40 sm:flex-row sm:items-center sm:justify-between">
              <div><h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ isOwner ? t('team.dissolve') : t('team.leave') }}</h2><p class="mt-1 text-sm text-gray-500">{{ isOwner ? t('team.dissolveMessage') : t('team.leaveMessage') }}</p></div>
              <button class="btn btn-danger" @click="isOwner ? askDissolve() : askLeave()"><Icon :name="isOwner ? 'trash' : 'arrowRight'" size="sm" />{{ isOwner ? t('team.dissolve') : t('team.leave') }}</button>
            </div>
          </div>
        </section>
      </template>
    </div>

    <TeamInvitationDialog :show="Boolean(invitationToken)" :loading="invitationPreviewLoading" :resolving="resolvingToken" :preview="invitationPreview" :error="invitationPreviewError" @close="closeInvitationDialog" @resolve="resolveInvitation" />

    <BaseDialog :show="Boolean(limitTarget)" :title="t('team.editLimits')" width="narrow" @close="limitTarget = null">
      <form id="team-limit-form" class="space-y-4" @submit.prevent="saveLimits">
        <div v-for="field in limitFields" :key="field.key"><label class="input-label">{{ field.label }}</label><input v-model.number="limitForm[field.key]" type="number" min="0" step="0.01" class="input" /></div>
        <div class="border-t border-gray-200 pt-4 dark:border-dark-700"><p class="input-label">{{ t('team.resetUsage') }}</p><div class="flex flex-wrap gap-4 text-sm text-gray-600 dark:text-gray-300"><label v-for="period in resetPeriods" :key="period.key" class="flex items-center gap-2"><input v-model="resetForm[period.key]" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />{{ period.label }}</label></div></div>
      </form>
      <template #footer><div class="flex justify-end gap-3"><button class="btn btn-secondary" @click="limitTarget = null">{{ t('common.cancel') }}</button><button form="team-limit-form" type="submit" class="btn btn-primary">{{ t('team.saveLimits') }}</button></div></template>
    </BaseDialog>

    <ConfirmDialog :show="Boolean(confirmAction)" :title="confirmAction?.title || ''" :message="confirmAction?.message || ''" :danger="confirmAction?.danger" @cancel="confirmAction = null" @confirm="runConfirmedAction" />
    <TotpStepUpDialog :controller="stepUp" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import TeamInvitationDialog from '@/components/team/TeamInvitationDialog.vue'
import TeamMemberUsageCharts from '@/components/charts/TeamMemberUsageCharts.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import { teamAPI, type TeamAPIKey, type TeamContext, type TeamInvitation, type TeamInvitationPreview, type TeamMembership, type TeamMemberUsageSeries, type TeamUsageLog, type TeamUsageSummary } from '@/api/team'
import { useAppStore } from '@/stores/app'
import { useOnboardingStore } from '@/stores/onboarding'
import { useStepUp, isStepUpCancelled } from '@/composables/useStepUp'
import { formatDateTime } from '@/utils/format'

type TeamTab = 'overview' | 'members' | 'keys' | 'invitations' | 'settings'
type LimitKey = 'daily_limit_usd' | 'weekly_limit_usd' | 'monthly_limit_usd'
type ResetKey = 'daily' | 'weekly' | 'monthly'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const onboardingStore = useOnboardingStore()
const stepUp = useStepUp()
const loading = ref(true)
const refreshing = ref(false)
const submitting = ref(false)
const resolvingToken = ref(false)
const usageLoading = ref(false)
const teamContext = ref<TeamContext | null>(null)
const members = ref<TeamMembership[]>([])
const invitations = ref<TeamInvitation[]>([])
const teamKeys = ref<TeamAPIKey[]>([])
const usageSummary = ref<TeamUsageSummary | null>(null)
const memberSeries = ref<TeamMemberUsageSeries[]>([])
const usageLogs = ref<TeamUsageLog[]>([])
const invitationPreview = ref<TeamInvitationPreview | null>(null)
const invitationPreviewLoading = ref(false)
const invitationPreviewError = ref('')
const activeTab = ref<TeamTab>('overview')
const createName = ref('')
const renameName = ref('')
const inviteEmail = ref('')
const defaultLimitForm = reactive<Record<LimitKey, number>>({ daily_limit_usd: 0, weekly_limit_usd: 0, monthly_limit_usd: 0 })
const invitationToken = computed(() => typeof route.query.invitation === 'string' ? route.query.invitation : '')
const transferToken = computed(() => typeof route.query.transfer === 'string' ? route.query.transfer : '')
const isOwner = computed(() => teamContext.value?.membership.role === 'owner')
const isTeamActive = computed(() => teamContext.value?.team.status === 'active')
const selfServiceEnabled = computed(() => appStore.cachedPublicSettings?.team_self_service_enabled !== false)
const visibleTabs = computed(() => {
  const tabs: Array<{ value: TeamTab; label: string }> = []
  if (isTeamActive.value) tabs.push({ value: 'overview', label: t('team.overview') })
  if (isTeamActive.value && isOwner.value) tabs.push({ value: 'members', label: t('team.members') })
  tabs.push({ value: 'keys', label: t('team.keys') })
  if (isTeamActive.value && isOwner.value) tabs.push({ value: 'invitations', label: t('team.invitations') })
  tabs.push({ value: 'settings', label: t('team.settings') })
  return tabs
})
const teamTabTourMarkers: Partial<Record<TeamTab, string>> = {
  members: 'team-members',
  invitations: 'team-invitations',
  settings: 'team-settings-tab',
}
const teamTabTourMarker = (tab: TeamTab) => teamTabTourMarkers[tab]
const formatMoney = (value: number, digits = 2) => `$${Number(value || 0).toFixed(digits)}`
const formatLimit = (used: number, limit: number) => limit > 0 ? `${formatMoney(used)} / ${formatMoney(limit)}` : `${formatMoney(used)} / ${t('team.unlimited')}`
const usageMetrics = computed(() => [
  { label: t('team.totalCost'), value: formatMoney(usageSummary.value?.actual_cost || 0, 4) },
  { label: t('team.requests'), value: Number(usageSummary.value?.request_count || 0).toLocaleString() },
  { label: t('team.inputTokens'), value: Number(usageSummary.value?.input_tokens || 0).toLocaleString() },
  { label: t('team.outputTokens'), value: Number(usageSummary.value?.output_tokens || 0).toLocaleString() },
])
const memberSeriesForChart = computed(() => memberSeries.value.map((item) => ({ userID: item.actor_user_id, label: item.display_name, summary: item.summary })))
const memberLimits = computed(() => {
  const membership = teamContext.value?.membership
  if (!membership) return []
  return [
    { label: t('team.daily'), used: membership.daily_usage_usd, limit: membership.daily_limit_usd },
    { label: t('team.weekly'), used: membership.weekly_usage_usd, limit: membership.weekly_limit_usd },
    { label: t('team.monthly'), used: membership.monthly_usage_usd, limit: membership.monthly_limit_usd },
  ].map((item) => ({ ...item, display: formatLimit(item.used, item.limit), percent: item.limit > 0 ? Math.min(100, (item.used / item.limit) * 100) : 0 }))
})

const isNoTeamError = (error: any) => error?.reason === 'TEAM_NOT_FOUND' || error?.reason === 'TEAM_MEMBERSHIP_REQUIRED' || error?.code === 'TEAM_NOT_FOUND' || error?.code === 'TEAM_MEMBERSHIP_REQUIRED'
const applyContext = (context: TeamContext) => {
  teamContext.value = context
  if (!context.team || context.team.status !== 'active') activeTab.value = 'settings'
  renameName.value = context.team.name
  defaultLimitForm.daily_limit_usd = context.team.default_daily_limit_usd
  defaultLimitForm.weekly_limit_usd = context.team.default_weekly_limit_usd
  defaultLimitForm.monthly_limit_usd = context.team.default_monthly_limit_usd
}
const loadContext = async () => {
  try { applyContext(await teamAPI.current()) } catch (error) { if (isNoTeamError(error)) teamContext.value = null; else throw error }
}
const loadTeamData = async () => {
  if (!teamContext.value) return
  usageLoading.value = true
  try {
    const requests = [teamAPI.keys().then((value) => { teamKeys.value = value })]
    if (!isTeamActive.value) {
      members.value = []
      invitations.value = []
      usageSummary.value = null
      memberSeries.value = []
      usageLogs.value = []
      await Promise.all(requests)
      return
    }

    requests.push(teamAPI.usage().then((value) => { usageSummary.value = value }))
    requests.push(teamAPI.usageLogs({ limit: 10 }).then((value) => { usageLogs.value = value.items }))
    requests.push(teamAPI.memberUsage().then((value) => { memberSeries.value = value }))
    requests.push(teamAPI.members().then((value) => { members.value = value }))
    if (isOwner.value) requests.push(teamAPI.invitations().then((value) => { invitations.value = value }))
    await Promise.all(requests)
  } finally { usageLoading.value = false }
}
const refreshAll = async () => { refreshing.value = true; try { await loadContext(); await loadTeamData() } catch (error: any) { appStore.showError(error?.message || t('team.loadFailed')) } finally { refreshing.value = false } }
const createTeam = async () => { submitting.value = true; try { applyContext(await teamAPI.create(createName.value)); await loadTeamData(); appStore.showSuccess(t('team.created')) } catch (error: any) { appStore.showError(error?.message || t('common.error')) } finally { submitting.value = false } }
const renameTeam = async () => { submitting.value = true; try { applyContext(await teamAPI.rename(renameName.value)); appStore.showSuccess(t('team.updated')) } catch (error: any) { appStore.showError(error?.message || t('common.error')) } finally { submitting.value = false } }
const saveDefaultMemberLimits = async () => { submitting.value = true; try { applyContext(await teamAPI.updateDefaultMemberLimits({ default_daily_limit_usd: defaultLimitForm.daily_limit_usd, default_weekly_limit_usd: defaultLimitForm.weekly_limit_usd, default_monthly_limit_usd: defaultLimitForm.monthly_limit_usd })); appStore.showSuccess(t('team.defaultLimitsUpdated')) } catch (error: any) { appStore.showError(error?.message || t('common.error')) } finally { submitting.value = false } }
const sendInvitation = async () => { submitting.value = true; try { await teamAPI.invite(inviteEmail.value); inviteEmail.value = ''; invitations.value = await teamAPI.invitations(); appStore.showSuccess(t('team.operationSuccess')) } catch (error: any) { appStore.showError(error?.message || t('common.error')) } finally { submitting.value = false } }
const reissueInvitation = async (id: number) => { try { await teamAPI.reissueInvitation(id); invitations.value = await teamAPI.invitations(); appStore.showSuccess(t('team.operationSuccess')) } catch (error: any) { appStore.showError(error?.message || t('common.error')) } }
const revokeInvitation = async (id: number) => { try { await teamAPI.revokeInvitation(id); invitations.value = await teamAPI.invitations(); appStore.showSuccess(t('team.operationSuccess')) } catch (error: any) { appStore.showError(error?.message || t('common.error')) } }

const limitTarget = ref<TeamMembership | null>(null)
const limitForm = reactive<Record<LimitKey, number>>({ daily_limit_usd: 0, weekly_limit_usd: 0, monthly_limit_usd: 0 })
const resetForm = reactive<Record<ResetKey, boolean>>({ daily: false, weekly: false, monthly: false })
const limitFields = computed(() => [{ key: 'daily_limit_usd' as const, label: t('team.daily') }, { key: 'weekly_limit_usd' as const, label: t('team.weekly') }, { key: 'monthly_limit_usd' as const, label: t('team.monthly') }])
const defaultLimitFields = computed(() => [{ key: 'daily_limit_usd' as const, label: t('team.defaultDailyLimit') }, { key: 'weekly_limit_usd' as const, label: t('team.defaultWeeklyLimit') }, { key: 'monthly_limit_usd' as const, label: t('team.defaultMonthlyLimit') }])
const resetPeriods = computed(() => [{ key: 'daily' as const, label: t('team.daily') }, { key: 'weekly' as const, label: t('team.weekly') }, { key: 'monthly' as const, label: t('team.monthly') }])
const openLimitEditor = (member: TeamMembership) => { limitTarget.value = member; limitForm.daily_limit_usd = member.daily_limit_usd; limitForm.weekly_limit_usd = member.weekly_limit_usd; limitForm.monthly_limit_usd = member.monthly_limit_usd; resetForm.daily = resetForm.weekly = resetForm.monthly = false }
const saveLimits = async () => { if (!limitTarget.value) return; try { await teamAPI.updateLimits(limitTarget.value.user_id, { ...limitForm }); if (resetForm.daily || resetForm.weekly || resetForm.monthly) await teamAPI.resetUsage(limitTarget.value.user_id, { ...resetForm }); limitTarget.value = null; await refreshAll(); appStore.showSuccess(t('team.operationSuccess')) } catch (error: any) { appStore.showError(error?.message || t('common.error')) } }

type ConfirmAction = { title: string; message: string; danger?: boolean; run: () => Promise<void> }
const confirmAction = ref<ConfirmAction | null>(null)
const runConfirmedAction = async () => { const action = confirmAction.value; confirmAction.value = null; if (!action) return; try { await action.run(); appStore.showSuccess(t('team.operationSuccess')) } catch (error: any) { if (!isStepUpCancelled(error)) appStore.showError(error?.message || t('common.error')) } }
const askRemove = (member: TeamMembership) => { confirmAction.value = { title: t('team.removeTitle'), message: t('team.removeMessage'), danger: true, run: async () => { await teamAPI.removeMember(member.user_id); await refreshAll() } } }
const askTransfer = (member: TeamMembership) => { confirmAction.value = { title: t('team.transferTitle'), message: t('team.transferMessage'), run: async () => { await stepUp.run(() => teamAPI.startTransfer(member.user_id)) } } }
const askLeave = () => { confirmAction.value = { title: t('team.leaveTitle'), message: t('team.leaveMessage'), danger: true, run: async () => { await teamAPI.leave(); teamContext.value = null } } }
const askDissolve = () => { confirmAction.value = { title: t('team.dissolveTitle'), message: t('team.dissolveMessage'), danger: true, run: async () => { await stepUp.run(() => teamAPI.dissolve()); teamContext.value = null } } }
const askSetStatus = () => { if (!teamContext.value) return; const status = teamContext.value.team.status === 'active' ? 'suspended' : 'active'; confirmAction.value = { title: status === 'suspended' ? t('team.pauseTitle') : t('team.resumeTitle'), message: status === 'suspended' ? t('team.pauseMessage') : t('team.resumeMessage'), danger: status === 'suspended', run: async () => { applyContext(await stepUp.run(() => teamAPI.setStatus(status))); await loadTeamData() } } }
const askDisableKey = (key: TeamAPIKey) => { confirmAction.value = { title: t('team.disableKeyTitle'), message: t('team.disableKeyMessage'), run: async () => { await teamAPI.disableKey(key.id); teamKeys.value = await teamAPI.keys() } } }
const askEnableKey = (key: TeamAPIKey) => { confirmAction.value = { title: t('team.enableKeyTitle'), message: t('team.enableKeyMessage'), run: async () => { await teamAPI.enableKey(key.id); teamKeys.value = await teamAPI.keys() } } }
const askDeleteKey = (key: TeamAPIKey) => { confirmAction.value = { title: t('team.deleteKeyTitle'), message: t('team.deleteKeyMessage'), danger: true, run: async () => { await teamAPI.deleteKey(key.id); teamKeys.value = await teamAPI.keys() } } }

const statusLabel = (status: string) => ({ active: t('team.statusActive'), suspended: t('team.statusSuspended'), inactive: t('team.statusInactive'), disabled: t('team.statusDisabled'), pending: t('team.statusPending'), accepted: t('team.statusAccepted'), declined: t('team.statusDeclined'), revoked: t('team.statusRevoked'), expired: t('team.statusExpired') })[status] || status
const startTeamGuide = () => onboardingStore.startTeamGuide({
  isOwner: Boolean(isOwner.value),
  hasTeam: Boolean(teamContext.value),
})
const loadInvitationPreview = async () => { if (!invitationToken.value) return; invitationPreviewLoading.value = true; invitationPreviewError.value = ''; try { invitationPreview.value = await teamAPI.previewInvitation(invitationToken.value) } catch (error: any) { invitationPreviewError.value = error?.message || t('team.invitationLoadFailed') } finally { invitationPreviewLoading.value = false } }
const clearActionQuery = async (key: 'invitation' | 'transfer') => { const query = { ...route.query }; delete query[key]; await router.replace({ query }) }
const closeInvitationDialog = async () => { invitationPreview.value = null; invitationPreviewError.value = ''; await clearActionQuery('invitation') }
const resolveInvitation = async (resolution: 'accepted' | 'declined') => { resolvingToken.value = true; try { await teamAPI.resolveInvitation(invitationToken.value, resolution); await closeInvitationDialog(); await refreshAll(); appStore.showSuccess(t('team.operationSuccess')) } catch (error: any) { appStore.showError(error?.message || t('common.error')) } finally { resolvingToken.value = false } }
const resolveTransfer = async (resolution: 'accepted' | 'declined') => { resolvingToken.value = true; try { await stepUp.run(() => teamAPI.resolveTransfer(transferToken.value, resolution)); await clearActionQuery('transfer'); await refreshAll(); appStore.showSuccess(t('team.operationSuccess')) } catch (error: any) { if (!isStepUpCancelled(error)) appStore.showError(error?.message || t('common.error')) } finally { resolvingToken.value = false } }

onMounted(async () => {
  const previewRequest = loadInvitationPreview()
  try { await loadContext(); await loadTeamData() } catch (error: any) { appStore.showError(error?.message || t('team.loadFailed')) } finally { loading.value = false }
  await previewRequest
})
</script>

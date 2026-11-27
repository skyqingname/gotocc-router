<template>
  <BaseDialog :show="show" :title="apiKey?.name ?? ''" width="normal" @close="emit('close')">
    <template #header-actions>
      <span v-if="apiKey" :key="apiKey.status" class="badge gc-reveal" :class="statusBadgeClass(apiKey.status)">{{ t('keys.status.' + apiKey.status) }}</span>
    </template>

    <div v-if="apiKey" class="space-y-5" data-test="key-detail">
      <section class="space-y-3">
        <div class="detail-secret">
          <DecodeText :value="apiKey.key" :revealed="revealed" class="min-w-0 flex-1 text-[13px] leading-6" data-test="key-detail-secret" />
          <button type="button" class="detail-icon-btn" :aria-label="revealed ? t('keyConsole.detail.hide') : t('keyConsole.detail.reveal')" :aria-pressed="revealed" data-test="key-detail-reveal" @click="revealed = !revealed">
            <Icon :name="revealed ? 'eyeOff' : 'eye'" size="sm" />
          </button>
          <button type="button" class="detail-icon-btn" :aria-label="t('keyConsole.detail.copyKey')" @click="emit('copy')">
            <Icon name="copy" size="sm" />
          </button>
        </div>
        <div class="detail-quick">
          <button v-if="showCcs && canConfigure" type="button" class="btn btn-primary" data-test="key-detail-ccs" @click="emit('ccswitch')">
            <Icon name="upload" size="sm" />
            {{ t('keyConsole.ccs.oneClick') }}
          </button>
          <button type="button" class="btn btn-secondary" @click="emit('copy')">
            <Icon name="clipboard" size="sm" />
            {{ t('keyConsole.detail.copyKey') }}
          </button>
        </div>
        <div class="detail-links">
          <button v-if="canConfigure" type="button" class="detail-link" data-test="key-detail-use" @click="emit('use')">
            <Icon name="terminal" size="xs" />{{ t('keys.useKey') }}
          </button>
          <button type="button" class="detail-link" @click="emit('test')">
            <Icon name="beaker" size="xs" />{{ t('keys.testKey') }}
          </button>
        </div>
      </section>

      <dl class="detail-metrics gc-num">
        <div>
          <dt>{{ t('keyConsole.stats.today') }}</dt>
          <dd><span v-if="usageLoading" class="gc-skeleton inline-block h-5 w-16"></span><span v-else>{{ withCreditUnit((usage?.today_actual_cost ?? 0).toFixed(4)) }}</span></dd>
        </div>
        <div>
          <dt>{{ t('keyConsole.stats.last30') }}</dt>
          <dd><span v-if="usageLoading" class="gc-skeleton inline-block h-5 w-16"></span><span v-else>{{ withCreditUnit((usage?.total_actual_cost ?? 0).toFixed(4)) }}</span></dd>
        </div>
        <div>
          <dt>{{ t('keyConsole.detail.remaining') }}</dt>
          <dd :class="apiKey.quota > 0 && apiKey.quota_used >= apiKey.quota ? 'text-red-500' : ''">
            {{ apiKey.quota > 0 ? withCreditUnit(Math.max(apiKey.quota - apiKey.quota_used, 0).toFixed(2)) : t('keyConsole.detail.unlimited') }}
          </dd>
        </div>
      </dl>

      <dl class="detail-facts">
        <div>
          <dt>{{ t('keyConsole.detail.group') }}</dt>
          <dd>
            <span v-if="apiKey.routing_mode === 'auto'" class="text-violet-700 dark:text-violet-300">{{ t('keys.routingMode.auto') }}</span>
            <span v-else-if="apiKey.group_binding_mode === 'all'">{{ t('keys.aggregateAll') }}</span>
            <span v-else-if="apiKey.group_binding_mode === 'custom'">{{ t('keys.aggregateCustomCount', { count: apiKey.aggregate_group_ids?.length || 0 }) }}</span>
            <GroupBadge
              v-else-if="apiKey.group"
              :name="apiKey.group.name"
              :platform="apiKey.group.platform"
              :subscription-type="apiKey.group.subscription_type"
              :rate-multiplier="apiKey.group.rate_multiplier"
              :user-rate-multiplier="userRate"
              :show-rate="apiKey.group.platform !== 'composite'"
              :peak-rate-enabled="apiKey.group.peak_rate_enabled"
              :peak-start="apiKey.group.peak_start"
              :peak-end="apiKey.group.peak_end"
              :peak-rate-multiplier="apiKey.group.peak_rate_multiplier"
              :truncate="false"
            />
            <span v-else class="surface-muted">{{ t('keys.noGroup') }}</span>
          </dd>
        </div>
        <div>
          <dt>{{ t('keyConsole.detail.models') }}</dt>
          <dd data-test="key-detail-models">
            <span class="flex flex-wrap items-center gap-2">
              <span>{{ t('keyConsole.detail.modelsFollowGroup') }}</span>
              <span v-if="modelState === 'loading'" class="gc-skeleton inline-block h-4 w-20"></span>
              <template v-else-if="modelState === 'error'">
                <span class="text-red-500">{{ t('keyConsole.models.failed') }}</span>
                <button type="button" class="detail-text-btn" @click="loadModels">{{ t('keyConsole.models.retry') }}</button>
              </template>
              <span v-else-if="modelState === 'ready' && models.length === 0" class="surface-muted">{{ t('keyConsole.models.none') }}</span>
              <button v-else-if="modelState === 'ready'" type="button" class="detail-text-btn" :aria-expanded="modelsOpen" @click="modelsOpen = !modelsOpen">
                {{ modelsOpen ? t('keyConsole.detail.collapse') : t('keyConsole.detail.modelCount', { count: models.length }) }}
              </button>
              <button v-else type="button" class="detail-text-btn" data-test="key-detail-load-models" @click="loadModels">{{ t('keyConsole.detail.showModels') }}</button>
            </span>
          </dd>
        </div>
        <div class="gc-collapse detail-models" :data-open="modelsOpen && models.length > 0" :inert="!modelsOpen || undefined">
          <div>
            <div class="flex flex-wrap gap-1.5 pt-1">
              <code v-for="model in models" :key="model" class="detail-chip">{{ model }}</code>
            </div>
          </div>
        </div>
        <div>
          <dt>{{ t('keyConsole.detail.ip') }}</dt>
          <dd>
            <span v-if="!apiKey.ip_whitelist?.length && !apiKey.ip_blacklist?.length">{{ t('keyConsole.detail.unrestricted') }}</span>
            <span v-else class="flex flex-col gap-0.5">
              <span v-if="apiKey.ip_whitelist?.length" :title="apiKey.ip_whitelist.join('\n')">{{ t('keyConsole.detail.whitelist', { count: apiKey.ip_whitelist.length }) }}</span>
              <span v-if="apiKey.ip_blacklist?.length" :title="apiKey.ip_blacklist.join('\n')">{{ t('keyConsole.detail.blacklist', { count: apiKey.ip_blacklist.length }) }}</span>
            </span>
          </dd>
        </div>
        <div v-if="rateLimits.length">
          <dt>{{ t('keys.rateLimitColumn') }}</dt>
          <dd class="flex flex-col gap-0.5 gc-num">
            <span v-for="limit in rateLimits" :key="limit.window">{{ limit.window }} · {{ withCreditUnit(limit.used.toFixed(2)) }} / {{ withCreditUnit(limit.limit.toFixed(2)) }}</span>
          </dd>
        </div>
        <div>
          <dt>{{ t('keys.lastUsedAt') }}</dt>
          <dd class="gc-num">
            {{ apiKey.last_used_at ? formatDateTime(apiKey.last_used_at) : t('keyConsole.detail.neverUsed') }}
            <span v-if="apiKey.last_used_ip" class="surface-muted"> · {{ apiKey.last_used_ip }}</span>
          </dd>
        </div>
        <div>
          <dt>{{ t('keys.created') }}</dt>
          <dd class="gc-num">{{ formatDateTime(apiKey.created_at) }}</dd>
        </div>
        <div>
          <dt>{{ t('keys.expiresAt') }}</dt>
          <dd class="gc-num" :class="apiKey.expires_at && new Date(apiKey.expires_at) < new Date() ? 'text-red-500' : ''">
            {{ apiKey.expires_at ? formatDateTime(apiKey.expires_at) : t('keys.noExpiration') }}
          </dd>
        </div>
      </dl>
    </div>

    <template #footer>
      <div v-if="apiKey" class="detail-footer">
        <div class="flex flex-wrap gap-2">
          <button v-support-readonly type="button" class="btn btn-secondary" data-test="key-detail-edit" @click="emit('edit')">
            <Icon name="edit" size="sm" />{{ t('common.edit') }}
          </button>
          <button v-support-readonly type="button" class="btn btn-secondary" :disabled="toggling" data-test="key-detail-toggle" @click="emit('toggle')">
            <svg v-if="toggling" class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true"><circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="3" class="opacity-25" /><path d="M21 12a9 9 0 0 0-9-9" stroke="currentColor" stroke-width="3" stroke-linecap="round" /></svg>
            <Icon v-else :name="apiKey.status === 'active' ? 'ban' : 'checkCircle'" size="sm" />
            {{ apiKey.status === 'active' ? t('keys.disable') : t('keys.enable') }}
          </button>
          <button type="button" class="btn btn-secondary" data-test="key-detail-logs" @click="emit('logs')">
            <Icon name="document" size="sm" />{{ t('keyConsole.detail.logs') }}
          </button>
        </div>
        <button v-support-readonly type="button" class="btn btn-secondary detail-danger" data-test="key-detail-delete" @click="emit('delete')">
          <Icon name="trash" size="sm" />{{ t('common.delete') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import DecodeText from '@/components/gotocc/motion/DecodeText.vue'
import { supportReadonly as vSupportReadonly } from '@/directives/supportReadonly'
import { useKeyModels } from '@/composables/useKeyModels'
import { formatDateTime } from '@/utils/format'
import { withCreditUnit } from '@/utils/credits'
import type { ApiKey } from '@/types'
import type { BatchApiKeyUsageStats } from '@/api/usage'

const props = defineProps<{
  show: boolean
  apiKey: ApiKey | null
  usage?: BatchApiKeyUsageStats
  usageLoading: boolean
  userRate: number | null
  canConfigure: boolean
  showCcs: boolean
  toggling: boolean
}>()
const emit = defineEmits<{
  close: []
  copy: []
  ccswitch: []
  use: []
  test: []
  edit: []
  toggle: []
  logs: []
  delete: []
}>()

const { t } = useI18n()
const { models, state: modelState, load, reset } = useKeyModels()
const revealed = ref(false)
const modelsOpen = ref(false)

const STATUS_BADGES: Record<string, string> = { active: 'badge-success', quota_exhausted: 'badge-warning', expired: 'badge-danger' }
const statusBadgeClass = (status: ApiKey['status']) => STATUS_BADGES[status] ?? 'badge-gray'

const rateLimits = computed(() => {
  const key = props.apiKey
  if (!key) return []
  return [
    { window: '5h', limit: key.rate_limit_5h, used: key.usage_5h },
    { window: '1d', limit: key.rate_limit_1d, used: key.usage_1d },
    { window: '7d', limit: key.rate_limit_7d, used: key.usage_7d }
  ].filter(item => item.limit > 0)
})

// Listing models calls the gateway with this key, which the gateway records as a
// use of the key; it therefore only runs when asked for.
async function loadModels() {
  if (!props.apiKey) return
  await load(props.apiKey.key)
  modelsOpen.value = models.value.length > 0
}

watch(() => [props.show, props.apiKey?.id] as const, () => {
  revealed.value = false
  modelsOpen.value = false
  reset()
}, { immediate: true })
</script>

<style scoped>
.detail-secret {
  display: flex; align-items: center; gap: 4px; padding: 10px 8px 10px 14px;
  border: 1px solid var(--surface-edge); border-radius: var(--surface-radius-control);
  background: var(--surface-soft); color: var(--surface-text);
}
.detail-icon-btn {
  display: inline-flex; flex: none; height: 30px; width: 30px; align-items: center; justify-content: center;
  border-radius: var(--surface-radius-tag); color: var(--surface-muted);
  transition: background-color var(--motion-fast) var(--motion-ease), color var(--motion-fast) var(--motion-ease);
}
.detail-icon-btn:hover { background: var(--surface-hover); color: var(--surface-text); }
.detail-quick { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
.detail-links { display: flex; flex-wrap: wrap; gap: 16px; }
.detail-link, .detail-text-btn { display: inline-flex; align-items: center; gap: 4px; font-size: 12px; font-weight: 600; color: theme('colors.primary.600'); }
.dark :is(.detail-link, .detail-text-btn) { color: theme('colors.primary.300'); }
.detail-link:hover, .detail-text-btn:hover { text-decoration: underline; }
.detail-metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
.detail-metrics > div { min-width: 0; padding: 10px 12px; border-radius: var(--surface-radius-control); background: var(--surface-soft); }
.detail-metrics dt { font-size: 12px; color: var(--surface-muted); }
.detail-metrics dd { margin-top: 2px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 15px; font-weight: 650; color: var(--surface-text); }
.detail-facts { display: flex; flex-direction: column; }
.detail-facts > div:not(.detail-models) {
  display: grid; grid-template-columns: 7.5rem minmax(0, 1fr); gap: 12px; padding-block: 9px;
  border-bottom: 1px solid var(--surface-edge); font-size: 13px;
}
.detail-facts > div:last-child { border-bottom: 0; }
.detail-facts dt { color: var(--surface-muted); }
.detail-facts dd { min-width: 0; color: var(--surface-text); overflow-wrap: anywhere; }
.detail-models > div { padding-left: calc(7.5rem + 12px); }
.detail-chip { padding: 1px 6px; border-radius: var(--surface-radius-xs); background: var(--surface-soft); font-size: 11px; color: var(--surface-text); }
.detail-footer { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px; }
.detail-danger { color: theme('colors.red.600'); }
.dark .detail-danger { color: theme('colors.red.400'); }
@media (max-width: 640px) {
  .detail-quick { grid-template-columns: minmax(0, 1fr); }
  .detail-metrics > div { padding: 8px 10px; }
  .detail-metrics dd { font-size: 13px; }
  .detail-facts > div:not(.detail-models) { grid-template-columns: 6rem minmax(0, 1fr); }
  .detail-models > div { padding-left: 0; }
}
</style>

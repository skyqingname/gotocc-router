<template>
  <AppLayout appearance="user">
    <div class="ss-page" data-test="service-status">
      <section class="card ss-hero">
        <div class="ss-hero-main">
          <div class="min-w-0">
            <h2 class="ss-title">
              <span class="ss-summary-dot" :data-status="overall" aria-hidden="true"></span>
              <span :key="summary" class="gc-reveal">{{ summary }}</span>
            </h2>
            <p class="ss-lead">{{ t('serviceStatus.lead', { n: snapshot?.recent_requests ?? '' }) }}</p>
          </div>
          <div class="ss-refresh">
            <span v-if="snapshot" :key="snapshot.generated_at" class="ss-updated gc-num gc-reveal" :data-tip="t('serviceStatus.tips.updatedAt', { n: REFRESH_SECONDS })">
              {{ t('serviceStatus.updatedAt', { time: clock.format(new Date(snapshot.generated_at)) }) }}
            </span>
            <button type="button" class="ss-refresh-btn" :disabled="loading" :aria-label="t('serviceStatus.refresh')" :data-tip="t('serviceStatus.tips.updatedAt', { n: REFRESH_SECONDS })" @click="load">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            </button>
          </div>
        </div>
        <div class="ss-legend">
          <span v-for="item in legend" :key="item.status" class="ss-legend-item" :data-status="item.status">
            <i aria-hidden="true"></i>{{ t(`serviceStatus.status.${item.status}`) }}<span class="surface-muted">{{ item.text }}</span>
          </span>
        </div>
      </section>

      <div v-if="platforms.length > 1" class="ss-filters" role="group">
        <button
          type="button"
          class="ss-chip"
          :aria-pressed="!platform"
          @click="platform = ''"
        >
          {{ t('serviceStatus.filters.all') }}<span class="ss-chip-count gc-num">{{ snapshot!.groups.length }}</span>
        </button>
        <button
          v-for="item in platforms"
          :key="item.platform"
          type="button"
          class="ss-chip"
          :aria-pressed="platform === item.platform"
          :data-tip="t('serviceStatus.tips.filter', { platform: platformLabel(item.platform), n: item.count })"
          @click="platform = item.platform"
        >
          <PlatformIcon :platform="item.platform" size="sm" />{{ platformLabel(item.platform) }}<span class="ss-chip-count gc-num">{{ item.count }}</span>
        </button>
      </div>

      <div v-if="failed && !snapshot" class="card ss-failed">
        <span>{{ t('serviceStatus.loadFailed') }}</span>
        <button type="button" class="btn btn-secondary btn-sm" @click="load">{{ t('serviceStatus.retry') }}</button>
      </div>
      <div v-else-if="!snapshot" class="ss-grid">
        <div v-for="index in 6" :key="index" class="gc-skeleton h-[300px]"></div>
      </div>
      <div v-else-if="!snapshot.groups.length" class="card ss-failed">{{ t('serviceStatus.empty') }}</div>
      <div v-else class="ss-grid">
        <GroupStatusCard
          v-for="group in visibleGroups"
          :key="group.id"
          :group="group"
          :timezone="snapshot.timezone"
          @models="modelsGroup = $event"
          @use="useGroup"
        />
      </div>
    </div>
    <GroupModelsDialog :group="modelsGroup" @close="modelsGroup = null" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GroupStatusCard from '@/components/gotocc/status/GroupStatusCard.vue'
import GroupModelsDialog from '@/components/gotocc/status/GroupModelsDialog.vue'
import { serviceStatusAPI, type GroupHealth, type GroupStatus, type ServiceStatusSnapshot } from '@/api/serviceStatus'
import { platformLabel } from '@/utils/platformColors'
import { getLocale } from '@/i18n'
import serviceStatusConfig from '../../../../service-status.json'

// 服务状态：按分组的被动检测，只看真实请求。页头一句总结和更新时间，平台筛选，分组卡网格。
// 刷新间隔与成功率阈值来自根目录 service-status.json；服务端同样按它缓存。
const REFRESH_SECONDS = serviceStatusConfig.refresh_seconds
const { t } = useI18n()
const router = useRouter()
const snapshot = shallowRef<ServiceStatusSnapshot | null>(null)
const loading = ref(false)
const failed = ref(false)
const platform = ref('')
const modelsGroup = shallowRef<GroupStatus | null>(null)
const clock = computed(() => new Intl.DateTimeFormat(getLocale(), { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }))
const percent = (value: number) => new Intl.NumberFormat(getLocale(), { style: 'percent', maximumFractionDigits: 1 }).format(value)

const platforms = computed(() => {
  const counts = new Map<GroupStatus['platform'], number>()
  for (const group of snapshot.value?.groups ?? []) counts.set(group.platform, (counts.get(group.platform) ?? 0) + 1)
  return [...counts].map(([name, count]) => ({ platform: name, count }))
})
const visibleGroups = computed(() => (snapshot.value?.groups ?? []).filter(group => !platform.value || group.platform === platform.value))

const counts = computed(() => {
  const result: Record<GroupHealth, number> = { normal: 0, degraded: 0, outage: 0, idle: 0 }
  for (const group of snapshot.value?.groups ?? []) result[group.status]++
  return result
})
const overall = computed<GroupHealth>(() => counts.value.outage ? 'outage' : counts.value.degraded ? 'degraded' : counts.value.normal ? 'normal' : 'idle')
const summary = computed(() => {
  if (!snapshot.value) return t('serviceStatus.title')
  const parts = [
    counts.value.outage ? t('serviceStatus.summary.outage', { n: counts.value.outage }) : '',
    counts.value.degraded ? t('serviceStatus.summary.degraded', { n: counts.value.degraded }) : '',
  ].filter(Boolean)
  if (parts.length) return parts.join(t('serviceStatus.summary.separator'))
  return counts.value.normal ? t('serviceStatus.summary.allNormal') : t('serviceStatus.summary.idle')
})
const legend = computed(() => [
  { status: 'normal', text: t('serviceStatus.legend.normal', { rate: percent(serviceStatusConfig.normal_success_rate) }) },
  { status: 'degraded', text: t('serviceStatus.legend.degraded', { low: percent(serviceStatusConfig.degraded_success_rate), high: percent(serviceStatusConfig.normal_success_rate) }) },
  { status: 'outage', text: t('serviceStatus.legend.outage', { rate: percent(serviceStatusConfig.degraded_success_rate) }) },
  { status: 'idle', text: t('serviceStatus.legend.idle') },
])

async function load() {
  loading.value = true
  try {
    snapshot.value = await serviceStatusAPI.getSnapshot()
    failed.value = false
    if (platform.value && !snapshot.value.groups.some(group => group.platform === platform.value)) platform.value = ''
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

function useGroup(group: GroupStatus) {
  void router.push({ path: '/keys', query: { create: '1', group: String(group.id) } })
}

let timer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  void load()
  timer = setInterval(() => { if (!document.hidden) void load() }, REFRESH_SECONDS * 1000)
})
onBeforeUnmount(() => clearInterval(timer))
</script>

<style scoped>
.ss-page { display: flex; width: 100%; flex-direction: column; gap: 14px; }
.ss-hero { position: relative; display: flex; flex-direction: column; gap: 14px; overflow: hidden; padding: 20px 22px; }
.ss-hero::before {
  content: ''; position: absolute; inset: 0; pointer-events: none; border-radius: inherit;
  background: radial-gradient(28rem 12rem at 0% 0%, rgb(16 185 129 / 12%), transparent 70%), radial-gradient(22rem 10rem at 100% 0%, rgb(99 102 241 / 12%), transparent 70%);
}
.ss-hero > * { position: relative; }
.ss-hero-main { display: flex; flex-wrap: wrap; align-items: flex-start; justify-content: space-between; gap: 12px 24px; }
.ss-title { display: flex; align-items: center; gap: 10px; font-size: 22px; font-weight: 750; letter-spacing: -.015em; color: var(--surface-text); }
.ss-summary-dot { position: relative; height: 12px; width: 12px; flex: none; border-radius: 9999px; background: var(--surface-muted); }
.ss-summary-dot[data-status='normal'] { background: theme('colors.emerald.500'); box-shadow: 0 0 0 5px rgb(16 185 129 / 18%); }
.ss-summary-dot[data-status='degraded'] { background: theme('colors.amber.500'); box-shadow: 0 0 0 5px rgb(245 158 11 / 20%); }
.ss-summary-dot[data-status='outage'] { background: theme('colors.rose.500'); box-shadow: 0 0 0 5px rgb(244 63 94 / 18%); }
.ss-summary-dot:not([data-status='idle'])::after {
  content: ''; position: absolute; inset: -5px; border-radius: inherit; border: 2px solid currentColor; color: inherit; opacity: 0;
  animation: ss-ping 2.4s var(--motion-ease-fluid) infinite;
}
.ss-summary-dot[data-status='normal']::after { border-color: theme('colors.emerald.400'); }
.ss-summary-dot[data-status='degraded']::after { border-color: theme('colors.amber.400'); }
.ss-summary-dot[data-status='outage']::after { border-color: theme('colors.rose.400'); }
@keyframes ss-ping { 0% { transform: scale(.6); opacity: .8; } 100% { transform: scale(1.6); opacity: 0; } }
.ss-lead { margin-top: 6px; max-width: 46rem; font-size: 13px; line-height: 1.7; color: var(--surface-muted); }
.ss-refresh { display: flex; align-items: center; gap: 8px; }
.ss-updated { font-size: 12px; color: var(--surface-muted); }
.ss-refresh-btn {
  display: grid; height: 34px; width: 34px; place-items: center; border-radius: 10px; color: var(--surface-muted);
  background: var(--surface-soft); transition: background-color var(--motion-hover) var(--motion-ease), color var(--motion-hover) var(--motion-ease);
}
.ss-refresh-btn:hover:not(:disabled) { background: var(--surface-hover); color: var(--surface-text); }
.ss-legend { display: flex; flex-wrap: wrap; gap: 6px 16px; font-size: 12px; color: var(--surface-text); }
.ss-legend-item { display: inline-flex; align-items: center; gap: 6px; }
.ss-legend-item > span { margin-left: 2px; }
.ss-legend-item > i { height: 10px; width: 10px; border-radius: 3px; background: var(--surface-soft); box-shadow: inset 0 0 0 1px var(--surface-edge); }
.ss-legend-item[data-status='normal'] > i { background: theme('colors.emerald.500'); box-shadow: none; }
.ss-legend-item[data-status='degraded'] > i { background: theme('colors.amber.400'); box-shadow: none; }
.ss-legend-item[data-status='outage'] > i { background: theme('colors.rose.500'); box-shadow: none; }
.ss-filters { display: flex; flex-wrap: wrap; gap: 8px; }
.ss-chip {
  display: inline-flex; height: 32px; align-items: center; gap: 6px; border-radius: 9999px; padding-inline: 12px;
  font-size: 13px; font-weight: 600; color: var(--surface-muted); background: var(--surface-solid);
  box-shadow: inset 0 0 0 1px var(--surface-edge);
  transition: color var(--motion-hover) var(--motion-ease), background-color var(--motion-hover) var(--motion-ease), box-shadow var(--motion-hover) var(--motion-ease), transform var(--motion-press) var(--motion-ease);
}
.ss-chip:hover { color: var(--surface-text); box-shadow: inset 0 0 0 1px var(--card-hover-edge); }
.ss-chip:active { transform: scale(var(--press-scale)); }
.ss-chip[aria-pressed='true'] { color: var(--surface-text); background: var(--surface-selected); box-shadow: inset 0 0 0 1px var(--card-hover-edge); }
.ss-chip-count { min-width: 18px; border-radius: 9999px; padding: 0 6px; font-size: 11px; line-height: 18px; text-align: center; background: var(--surface-soft); }
.ss-failed { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 10px; padding: 16px; font-size: 13px; color: var(--surface-muted); }
.ss-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; }
@media (max-width: 1200px) { .ss-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 720px) {
  .ss-grid { grid-template-columns: minmax(0, 1fr); }
  .ss-hero { padding: 16px; }
  .ss-title { font-size: 19px; }
}
@media (prefers-reduced-motion: reduce) {
  .ss-summary-dot::after { animation: none; }
}
</style>

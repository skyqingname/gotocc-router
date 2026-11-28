<template>
  <article class="card gs-card" :data-status="group.status" data-test="status-card">
    <header class="gs-head">
      <div class="gs-name-row">
        <span class="gs-platform" :data-tip="platformLabel(group.platform)"><PlatformIcon :platform="group.platform" size="sm" /></span>
        <h3 class="gs-name">{{ group.name }}</h3>
        <span v-if="group.is_exclusive" class="gs-badge" :data-tip="t('serviceStatus.tips.exclusive')">{{ t('serviceStatus.card.exclusive') }}</span>
      </div>
      <span class="gs-rate gc-num" :data-tip="t('serviceStatus.tips.rate', { rate: rateText })">{{ t('serviceStatus.card.rate', { rate: rateText }) }}</span>
    </header>
    <p v-if="group.description" class="gs-desc">{{ group.description }}</p>

    <div class="gs-state">
      <span class="gs-pill" :data-status="group.status" :data-tip="statusTip">
        <i class="gs-dot" aria-hidden="true"></i>{{ t(`serviceStatus.status.${group.status}`) }}
      </span>
      <span v-if="group.recent.total" class="gs-recent gc-num" :data-tip="statusTip">
        {{ t('serviceStatus.card.recent', { total: group.recent.total, success: group.recent.success }) }}
      </span>
    </div>

    <dl class="gs-metrics">
      <div class="gs-metric" :data-tip="group.day.total ? t('serviceStatus.tips.daySuccess', { total: group.day.total, success: group.day.success }) : t('serviceStatus.tips.dayIdle')">
        <dt>{{ t('serviceStatus.card.daySuccess') }}</dt>
        <dd class="gc-num">{{ group.day.total ? percent(group.day.success / group.day.total) : '—' }}</dd>
        <span v-if="group.day.total" class="gs-sub gc-num">{{ group.day.success }}/{{ group.day.total }}</span>
      </div>
      <div class="gs-metric" :data-tip="group.ttft_p50_ms === null ? t('serviceStatus.tips.ttftNone') : t('serviceStatus.tips.ttft')">
        <dt>{{ t('serviceStatus.card.ttft') }}</dt>
        <dd class="gc-num">{{ group.ttft_p50_ms === null ? '—' : duration(group.ttft_p50_ms) }}</dd>
      </div>
      <div class="gs-metric" :data-tip="group.cache_hit_rate === null ? t('serviceStatus.tips.cacheNone') : t('serviceStatus.tips.cacheHit')">
        <dt>{{ t('serviceStatus.card.cacheHit') }}</dt>
        <dd class="gc-num">{{ group.cache_hit_rate === null ? '—' : percent(group.cache_hit_rate) }}</dd>
      </div>
    </dl>

    <GroupStatusBar :buckets="group.buckets" :timezone="timezone" />

    <footer class="gs-foot">
      <button type="button" class="btn btn-secondary btn-sm" :data-tip="t('serviceStatus.tips.models')" data-test="status-models" @click="emit('models', group)">
        <Icon name="cube" size="sm" />{{ t('serviceStatus.card.models') }}
      </button>
      <button v-support-readonly type="button" class="btn btn-primary btn-sm" :data-tip="t('serviceStatus.tips.use')" data-test="status-use" @click="emit('use', group)">
        {{ t('serviceStatus.card.use') }}<Icon name="arrowRight" size="sm" class="gs-arrow" />
      </button>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GroupStatusBar from '@/components/gotocc/status/GroupStatusBar.vue'
import type { GroupStatus } from '@/api/serviceStatus'
import { platformLabel } from '@/utils/platformColors'
import { supportReadonly as vSupportReadonly } from '@/directives/supportReadonly'
import { getLocale } from '@/i18n'

// 一个分组一张卡：状态（最近 N 次请求）、三项指标、近 24 小时状态条，底部看模型或直接用它建密钥。
const props = defineProps<{ group: GroupStatus; timezone: string }>()
const emit = defineEmits<{ models: [group: GroupStatus]; use: [group: GroupStatus] }>()
const { t } = useI18n()

const rateText = computed(() => new Intl.NumberFormat(getLocale(), { maximumFractionDigits: 4 }).format(props.group.rate_multiplier))
const statusTip = computed(() => props.group.recent.total
  ? t('serviceStatus.tips.statusCounts', { total: props.group.recent.total, success: props.group.recent.success })
  : t('serviceStatus.tips.statusIdle'))

function percent(value: number): string {
  return new Intl.NumberFormat(getLocale(), { style: 'percent', maximumFractionDigits: 1 }).format(value)
}

function duration(ms: number): string {
  return ms < 1000 ? `${Math.round(ms)} ms` : `${new Intl.NumberFormat(getLocale(), { maximumFractionDigits: 2 }).format(ms / 1000)} s`
}
</script>

<style scoped>
.gs-card { position: relative; display: flex; min-width: 0; flex-direction: column; gap: 14px; overflow: hidden; padding: 18px 18px 16px; }
.gs-card::before {
  content: ''; position: absolute; left: 0; right: 0; top: 0; height: 3px; pointer-events: none;
  background: var(--gs-tone); opacity: .9;
}
.gs-card[data-status='normal'] { --gs-tone: linear-gradient(90deg, theme('colors.emerald.400'), theme('colors.teal.400')); }
.gs-card[data-status='degraded'] { --gs-tone: linear-gradient(90deg, theme('colors.amber.300'), theme('colors.amber.500')); }
.gs-card[data-status='outage'] { --gs-tone: linear-gradient(90deg, theme('colors.rose.400'), theme('colors.rose.600')); }
.gs-card[data-status='idle'] { --gs-tone: var(--surface-soft); }
.gs-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; }
.gs-name-row { display: flex; min-width: 0; align-items: center; gap: 8px; }
.gs-platform { display: grid; height: 28px; width: 28px; flex: none; place-items: center; border-radius: 9px; color: var(--surface-text); background: var(--surface-soft); }
.gs-name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 15px; font-weight: 700; color: var(--surface-text); }
.gs-badge { flex: none; border-radius: 9999px; padding: 1px 8px; font-size: 11px; font-weight: 650; color: theme('colors.accent.700'); background: rgb(139 92 246 / 12%); }
.dark .gs-badge { color: theme('colors.accent.300'); }
.gs-rate { flex: none; border-radius: 8px; padding: 3px 8px; font-size: 12px; font-weight: 700; color: var(--surface-text); background: var(--surface-soft); }
.gs-desc { margin-top: -6px; display: -webkit-box; overflow: hidden; -webkit-line-clamp: 2; -webkit-box-orient: vertical; font-size: 12px; line-height: 1.6; color: var(--surface-muted); }
.gs-state { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.gs-pill { display: inline-flex; align-items: center; gap: 7px; border-radius: 9999px; padding: 4px 11px 4px 9px; font-size: 13px; font-weight: 700; }
.gs-dot { position: relative; height: 8px; width: 8px; border-radius: 9999px; background: currentColor; }
.gs-pill[data-status='normal'] { color: theme('colors.emerald.700'); background: rgb(16 185 129 / 12%); }
.gs-pill[data-status='degraded'] { color: theme('colors.amber.700'); background: rgb(245 158 11 / 14%); }
.gs-pill[data-status='outage'] { color: theme('colors.rose.700'); background: rgb(244 63 94 / 12%); }
.gs-pill[data-status='idle'] { color: var(--surface-muted); background: var(--surface-soft); }
.dark .gs-pill[data-status='normal'] { color: theme('colors.emerald.300'); }
.dark .gs-pill[data-status='degraded'] { color: theme('colors.amber.300'); }
.dark .gs-pill[data-status='outage'] { color: theme('colors.rose.300'); }
.gs-pill:not([data-status='idle']) .gs-dot::after {
  content: ''; position: absolute; inset: -3px; border-radius: inherit; border: 2px solid currentColor; opacity: 0;
  animation: gs-ping 2.2s var(--motion-ease-fluid) infinite;
}
@keyframes gs-ping { 0% { transform: scale(.6); opacity: .7; } 100% { transform: scale(1.8); opacity: 0; } }
.gs-recent { font-size: 12px; color: var(--surface-muted); }
.gs-metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
.gs-metric {
  display: flex; min-width: 0; flex-direction: column; gap: 3px; border-radius: 12px; padding: 10px 11px; background: var(--surface-soft);
  transition: background-color var(--motion-hover) var(--motion-ease), transform var(--motion-hover) var(--motion-ease);
}
.gs-metric:hover { background: var(--surface-hover); transform: translateY(var(--hover-lift)); }
.gs-metric dt { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 11px; color: var(--surface-muted); }
.gs-metric dd { font-size: 17px; font-weight: 750; letter-spacing: -.01em; color: var(--surface-text); }
.gs-sub { font-size: 11px; color: var(--surface-muted); }
.gs-foot { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-top: 2px; }
.gs-foot :deep(.btn) { gap: 6px; }
.gs-arrow { transition: transform var(--motion-hover) var(--motion-ease); }
.gs-foot .btn-primary:hover .gs-arrow { transform: translateX(var(--icon-nudge)); }
@media (prefers-reduced-motion: reduce) {
  .gs-pill .gs-dot::after { animation: none; }
}
</style>

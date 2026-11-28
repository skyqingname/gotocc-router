<template>
  <div class="sbar">
    <div class="sbar-cells" role="list" :style="{ gridTemplateColumns: `repeat(${buckets.length}, minmax(0, 1fr))` }">
      <span
        v-for="(bucket, index) in buckets"
        :key="bucket.start"
        role="listitem"
        class="sbar-cell"
        :data-status="bucket.status"
        :data-now="index === buckets.length - 1 || undefined"
        :data-tip="tipOf(bucket, index === buckets.length - 1)"
        :aria-label="tipOf(bucket, index === buckets.length - 1)"
      ></span>
    </div>
    <div class="sbar-axis">
      <span>{{ t('serviceStatus.card.barStart') }}</span>
      <span>{{ t('serviceStatus.card.barEnd') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { StatusBucket } from '@/api/serviceStatus'
import { getLocale } from '@/i18n'

// 近 24 小时的状态条：每小时一格（格数和每格时长来自根目录 service-status.json），按该小时的成功率着色，没请求的格子灰色；最后一格是当前这一小时。
// 时刻按站点时区显示，和格子的整点对齐。
const props = defineProps<{ buckets: StatusBucket[]; timezone: string }>()
const { t } = useI18n()
const clock = computed(() => new Intl.DateTimeFormat(getLocale(), { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: props.timezone }))
const percent = new Intl.NumberFormat(getLocale(), { style: 'percent', maximumFractionDigits: 1 })

function tipOf(bucket: StatusBucket, now: boolean): string {
  const range = `${clock.value.format(new Date(bucket.start))}–${clock.value.format(new Date(bucket.end))}${now ? t('serviceStatus.tips.bucketNow') : ''}`
  if (!bucket.total) return t('serviceStatus.tips.bucketIdle', { range })
  return t('serviceStatus.tips.bucket', { range, total: bucket.total, rate: percent.format(bucket.success / bucket.total) })
}
</script>

<style scoped>
.sbar { display: flex; flex-direction: column; gap: 6px; }
.sbar-cells { display: grid; align-items: end; gap: 3px; height: 30px; }
.sbar-cell {
  height: 100%; border-radius: 3px; background: var(--surface-soft);
  transform-origin: bottom;
  transition: transform var(--motion-hover) var(--motion-ease), filter var(--motion-hover) var(--motion-ease), box-shadow var(--motion-hover) var(--motion-ease);
}
.sbar-cell[data-status='normal'] { background: linear-gradient(180deg, theme('colors.emerald.400'), theme('colors.emerald.500')); }
.sbar-cell[data-status='degraded'] { background: linear-gradient(180deg, theme('colors.amber.300'), theme('colors.amber.500')); }
.sbar-cell[data-status='outage'] { background: linear-gradient(180deg, theme('colors.rose.400'), theme('colors.rose.600')); }
.sbar-cell[data-status='idle'] { background: var(--surface-soft); box-shadow: inset 0 0 0 1px var(--surface-edge); }
.sbar-cell[data-now] { animation: sbar-now 2.4s var(--motion-ease-fluid) infinite; }
.sbar-cells:hover .sbar-cell { filter: saturate(.7) opacity(.75); }
.sbar-cells:hover .sbar-cell:hover { filter: none; transform: scaleY(1.14); box-shadow: 0 6px 12px -6px rgb(0 0 0 / 35%); }
@keyframes sbar-now { 0%, 100% { opacity: 1; } 50% { opacity: .62; } }
.sbar-axis { display: flex; justify-content: space-between; font-size: 11px; color: var(--surface-muted); }
@media (max-width: 640px) {
  .sbar-cells { gap: 2px; height: 26px; }
}
@media (prefers-reduced-motion: reduce) {
  .sbar-cell[data-now] { animation: none; }
}
</style>

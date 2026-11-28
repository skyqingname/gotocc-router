<template>
  <div class="cring" :data-tip="tip">
    <svg class="cring-svg" viewBox="0 0 120 120" aria-hidden="true">
      <defs>
        <linearGradient :id="gradientId" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" class="cring-stop-a" />
          <stop offset="100%" class="cring-stop-b" />
        </linearGradient>
      </defs>
      <circle class="cring-track" cx="60" cy="60" :r="RADIUS" />
      <circle
        class="cring-fill"
        cx="60" cy="60" :r="RADIUS"
        :stroke="`url(#${gradientId})`"
        :stroke-dasharray="CIRCUMFERENCE"
        :stroke-dashoffset="CIRCUMFERENCE * (1 - progress)"
      />
    </svg>
    <div class="cring-center">
      <span class="cring-value gc-num">{{ Math.round(shown) }}</span>
      <span class="cring-unit">{{ unit }}</span>
      <span class="cring-label">{{ label }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useCountUp } from '@/components/gotocc/dashboard/useCountUp'

// 本月进度环：已签天数 ÷ 本月天数，数字跟着滚动。
const props = defineProps<{ days: number; total: number; label: string; unit: string; tip: string }>()
const RADIUS = 52
const CIRCUMFERENCE = 2 * Math.PI * RADIUS
const gradientId = `checkin-ring-${Math.random().toString(36).slice(2, 8)}`
const progress = computed(() => Math.min(props.days / props.total, 1))
const shown = useCountUp(() => props.days, 700)
</script>

<style scoped>
.cring { position: relative; display: grid; height: 132px; width: 132px; flex: none; place-items: center; }
.cring-svg { position: absolute; inset: 0; height: 100%; width: 100%; transform: rotate(-90deg); }
.cring-track { fill: none; stroke: var(--surface-soft); stroke-width: 10; }
.cring-fill {
  fill: none; stroke-width: 10; stroke-linecap: round;
  transition: stroke-dashoffset 900ms var(--motion-ease-fluid);
  filter: drop-shadow(0 4px 10px rgb(99 102 241 / 30%));
}
.cring-stop-a { stop-color: theme('colors.primary.500'); }
.cring-stop-b { stop-color: theme('colors.accent.500'); }
.cring-center { position: relative; display: flex; flex-direction: column; align-items: center; line-height: 1; }
.cring-value { font-size: 34px; font-weight: 750; letter-spacing: -.02em; color: var(--surface-text); }
.cring-unit { margin-top: 2px; font-size: 12px; font-weight: 600; color: var(--surface-muted); }
.cring-label { margin-top: 6px; font-size: 11px; color: var(--surface-muted); }
@media (max-width: 640px) {
  .cring { height: 108px; width: 108px; }
  .cring-value { font-size: 28px; }
}
</style>

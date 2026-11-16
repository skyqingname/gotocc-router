<!-- Card structure adapted from TokenFlux/TokenRouter e74bba115fc2cfd77e97a2a22a15dad0605d64d1 (frontend/src/components/admin/usage/UsageStatsCards.vue). Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation. -->
<template>
  <!-- 指标卡：图标、标签、数值和副文字。可点击时渲染为按钮并带选中态，否则为静态卡片。 -->
  <component
    :is="clickable ? 'button' : 'div'"
    :type="clickable ? 'button' : undefined"
    :aria-pressed="clickable ? active : undefined"
    class="metric-card card flex min-w-0 flex-col gap-3 overflow-hidden p-4 text-left"
    :class="clickable && (active ? '!border-primary-500 ring-1 ring-primary-500/30 dark:!border-primary-400' : 'hover:!border-gray-300 dark:hover:!border-dark-500')"
    @click="clickable && emit('select')"
  >
    <div class="flex min-w-0 items-center gap-2">
      <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md" :class="[METRIC_TONES[tone].tile, METRIC_TONES[tone].icon]">
        <Icon :name="icon" size="sm" />
      </span>
      <span class="truncate text-xs font-medium text-gray-500 dark:text-dark-400">{{ label }}</span>
    </div>
    <Skeleton v-if="loading" :width="96" :height="32" />
    <span v-else class="min-w-0 truncate text-2xl font-semibold tracking-tight tabular-nums text-gray-900 dark:text-white">
      <template v-if="typeof value === 'number'">
        <!-- 滚动中的数字只做视觉展示，读屏读取最终值 -->
        <span aria-hidden="true">{{ format(animated) }}</span>
        <span class="sr-only">{{ format(value) }}</span>
      </template>
      <template v-else>{{ value }}</template>
    </span>
    <Skeleton v-if="loading" :width="120" :height="16" />
    <span v-else-if="hint" class="truncate text-xs text-gray-500 dark:text-dark-400" :title="hint">{{ hint }}</span>
  </component>
</template>

<script setup lang="ts">
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatNumberLocaleString } from '@/utils/format'
import { COUNT_UP_MS } from './motion'
import { METRIC_TONES, type MetricTone } from './tones'
import { useCountUp } from './useCountUp'

type IconName = InstanceType<typeof Icon>['$props']['name']

// 数值为数字时从旧值滚动到新值并用 format 格式化，为文字时原样显示（如「见使用记录」）。
const props = withDefaults(defineProps<{
  icon: IconName
  label: string
  value: number | string
  tone: MetricTone
  hint?: string
  format?: (value: number) => string
  loading?: boolean
  clickable?: boolean
  active?: boolean
}>(), {
  format: (value: number) => formatNumberLocaleString(Math.round(value)),
})

const emit = defineEmits<{ (event: 'select'): void }>()

const animated = useCountUp(() => (typeof props.value === 'number' ? props.value : 0), COUNT_UP_MS)
</script>

<style scoped>
.metric-card {
  transition: border-color var(--motion-fast) var(--motion-ease), box-shadow var(--motion-fast) var(--motion-ease);
}
</style>

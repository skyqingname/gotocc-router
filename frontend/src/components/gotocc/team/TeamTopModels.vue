<template>
  <div class="card flex min-w-0 flex-col p-4" :style="{ '--bar-ms': `${TOP_MODEL_BAR_MS}ms` }">
    <div class="mb-4 flex min-h-7 items-baseline gap-2">
      <h3 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
        <Icon name="chartBar" size="sm" class="text-primary-600 dark:text-primary-400" />
        {{ t('gotocc.dashboard.topModels.title') }}
      </h3>
      <span class="truncate text-xs text-gray-500 dark:text-dark-400">{{ t('gotocc.dashboard.topModels.subtitle', { metric: t(`gotocc.dashboard.usageChart.metrics.${metric}`) }) }}</span>
    </div>
    <div v-if="!loaded" class="space-y-4" aria-hidden="true">
      <div v-for="row in TOP_COUNT" :key="row" class="space-y-2 px-2"><Skeleton width="70%" :height="16" /><Skeleton width="100%" :height="4" /></div>
    </div>
    <div v-else-if="rows.length === 0" class="flex flex-1 flex-col items-center justify-center gap-2 py-8 text-sm text-gray-500 dark:text-dark-400">
      <Icon name="cube" size="lg" class="text-gray-300 dark:text-dark-600" />
      {{ t('gotocc.dashboard.topModels.empty') }}
    </div>
    <ol v-else class="flex flex-1 flex-col gap-1">
      <li v-for="(row, index) in rows" :key="`${row.model}-${version}`" class="px-2 py-2">
        <div class="flex min-w-0 items-center gap-2">
          <span class="w-4 shrink-0 text-center text-xs font-semibold tabular-nums" :class="index === 0 ? 'text-primary-600 dark:text-primary-400' : 'text-gray-400 dark:text-dark-500'">{{ index + 1 }}</span>
          <ModelIcon :model="row.model" size="16px" class="shrink-0" />
          <span class="min-w-0 flex-1 truncate text-sm text-gray-800 dark:text-dark-100">{{ row.model }}</span>
          <span class="shrink-0 text-sm font-medium tabular-nums text-gray-900 dark:text-white">{{ row.value }}</span>
        </div>
        <div class="mt-2 flex items-center gap-2 pl-6">
          <div class="h-1 flex-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
            <div class="share-bar h-full rounded-full" :style="{ '--bar-w': `${row.share}%`, '--bar-delay': `${index * TOP_MODEL_BAR_STEP_MS}ms`, opacity: 1 - index * 0.15 }"></div>
          </div>
          <span class="w-12 shrink-0 text-right text-xs tabular-nums text-gray-500 dark:text-dark-400">{{ formatShare(row.share) }}</span>
        </div>
      </li>
      <li v-if="others" class="mt-auto flex items-center justify-between gap-2 px-2 pt-2 text-xs text-gray-500 dark:text-dark-400">
        <span class="truncate">{{ t('gotocc.dashboard.topModels.others', { count: others.count }) }}</span>
        <span class="shrink-0 tabular-nums">{{ others.value }} · {{ formatShare(others.share) }}</span>
      </li>
    </ol>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import type { TeamUsageModel } from '@/api/team'
import { TOP_MODEL_BAR_MS, TOP_MODEL_BAR_STEP_MS } from '@/components/gotocc/dashboard/motion'
import { formatMetric, modelValue, type TeamMetric } from './teamMetrics'

// 团队常用模型：按当前指标取前 5，其余合并；与仪表盘的常用模型同一表现。
const TOP_COUNT = 5

const props = defineProps<{ models: TeamUsageModel[]; metric: TeamMetric; loaded: boolean }>()
const { t } = useI18n()

// 数据或指标变化时递增，作为行 key 让占比条重新伸展。
const version = ref(0)
watch(() => [props.models, props.metric], () => {
  version.value += 1
})

const ranking = computed(() => {
  const sorted = props.models
    .filter((model) => modelValue(model, props.metric) > 0)
    .sort((a, b) => modelValue(b, props.metric) - modelValue(a, props.metric))
  const total = sorted.reduce((sum, model) => sum + modelValue(model, props.metric), 0)
  const rest = sorted.slice(TOP_COUNT)
  const restAmount = rest.reduce((sum, model) => sum + modelValue(model, props.metric), 0)
  return {
    top: sorted.slice(0, TOP_COUNT).map((model) => ({ model: model.model, value: formatMetric(modelValue(model, props.metric), props.metric), share: (modelValue(model, props.metric) / total) * 100 })),
    others: rest.length > 0 ? { count: rest.length, value: formatMetric(restAmount, props.metric), share: (restAmount / total) * 100 } : null,
  }
})

const rows = computed(() => ranking.value.top)
const others = computed(() => ranking.value.others)
const formatShare = (share: number) => `${share < 10 ? share.toFixed(1) : Math.round(share)}%`
</script>

<style scoped>
.share-bar {
  width: var(--bar-w);
  background-image: linear-gradient(90deg, theme('colors.primary.500'), theme('colors.accent.500'));
  animation: share-bar var(--bar-ms) var(--motion-ease) var(--bar-delay) both;
}

@keyframes share-bar {
  from {
    width: 0;
  }
  to {
    width: var(--bar-w);
  }
}

@media (prefers-reduced-motion: reduce) {
  .share-bar {
    animation: none;
  }
}
</style>

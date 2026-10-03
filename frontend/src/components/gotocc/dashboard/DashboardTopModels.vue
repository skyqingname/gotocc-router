<template>
  <div class="card flex min-w-0 flex-col p-4" :aria-busy="!loaded">
    <div class="mb-4 flex min-h-7 items-center justify-between gap-2">
      <div class="flex min-w-0 items-baseline gap-2">
        <h3 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
          <Icon name="chartBar" size="sm" class="text-primary-600 dark:text-primary-400" />
          {{ t('gotocc.dashboard.topModels.title') }}
        </h3>
        <span class="truncate text-xs text-gray-500 dark:text-dark-400">{{ subtitle }}</span>
      </div>
      <RouterLink to="/usage" class="inline-flex shrink-0 items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300">
        {{ t('gotocc.dashboard.topModels.viewAll') }}
        <Icon name="chevronRight" size="xs" />
      </RouterLink>
    </div>

    <div v-if="!loaded" class="space-y-4" aria-hidden="true">
      <div v-for="row in TOP_COUNT" :key="row" class="space-y-2 px-2">
        <div class="flex items-center gap-2">
          <Skeleton :width="16" :height="16" />
          <Skeleton width="55%" :height="16" />
          <Skeleton :width="48" :height="16" class="ml-auto" />
        </div>
        <Skeleton width="100%" :height="4" />
      </div>
    </div>

    <div v-else-if="rows.length === 0" class="flex flex-1 flex-col items-center justify-center gap-2 py-8 text-sm text-gray-500 dark:text-dark-400">
      <Icon name="cube" size="lg" class="text-gray-300 dark:text-dark-600" />
      {{ t('gotocc.dashboard.topModels.empty') }}
    </div>

    <ol v-else class="flex flex-1 flex-col gap-1">
      <li v-for="(row, index) in rows" :key="`${row.model}-${version}`">
        <!-- 点击按该模型筛选整个用量区，再次点击取消 -->
        <button
          type="button"
          class="group w-full rounded-lg px-2 py-2 text-left transition-colors"
          :class="row.active ? 'bg-primary-500/10 ring-1 ring-primary-500/20' : 'hover:bg-gray-50 dark:hover:bg-dark-700'"
          :aria-pressed="row.active"
          :title="t(row.active ? 'gotocc.dashboard.topModels.clearFilter' : 'gotocc.dashboard.topModels.filterHint')"
          @click="applyModelFilter(row.active ? null : row.model)"
        >
          <div class="flex min-w-0 items-center gap-2">
            <span class="w-4 shrink-0 text-center text-xs font-semibold tabular-nums" :class="index === 0 ? 'text-primary-600 dark:text-primary-400' : 'text-gray-400 dark:text-dark-500'">{{ index + 1 }}</span>
            <ModelIcon :model="row.model" size="16px" class="shrink-0" />
            <span class="min-w-0 flex-1 truncate text-sm" :class="row.active ? 'font-medium text-primary-700 dark:text-primary-300' : 'text-gray-800 dark:text-dark-100'">{{ row.model }}</span>
            <span class="shrink-0 text-sm font-medium tabular-nums text-gray-900 dark:text-white">{{ row.value }}</span>
          </div>
          <div class="mt-2 flex items-center gap-2 pl-6">
            <div class="h-1 flex-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
              <div class="top-model-bar h-full rounded-full" :style="{ '--bar-w': `${row.share}%`, '--bar-delay': `${index * TOP_MODEL_BAR_STEP_MS}ms`, opacity: 1 - index * 0.15 }"></div>
            </div>
            <span class="w-12 shrink-0 text-right text-xs tabular-nums text-gray-500 dark:text-dark-400">{{ row.shareText }}</span>
          </div>
        </button>
      </li>
      <!-- 排行之外的模型合并成一行，只展示合计和占比 -->
      <li v-if="others" class="mt-auto flex items-center justify-between gap-2 px-2 pt-2 text-xs text-gray-500 dark:text-dark-400">
        <span class="truncate">{{ t('gotocc.dashboard.topModels.others', { count: others.count }) }}</span>
        <span class="shrink-0 tabular-nums">{{ others.value }} · {{ others.shareText }}</span>
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
import { formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import type { ModelStat } from '@/types'
import { formatCostAuto } from './format'
import { TOP_MODEL_BAR_STEP_MS } from './motion'
import { cacheHitRateOf } from './usageData'
import { injectUsageState, type UsageMetric } from './usageState'

// 常用模型排行，结构参照 TokenRouter（LGPL-3.0）：按当前指标取前 5，其余合并；占比条依次伸展。
const TOP_COUNT = 5

const { t } = useI18n()
const { metric, models, loaded, filters, applyModelFilter } = injectUsageState()

// 每次取到新数据或切换指标时递增，作为行 key 让占比条重新伸展。
const version = ref(0)
watch([models, metric], () => {
  version.value += 1
})

const subtitle = computed(() => t('gotocc.dashboard.topModels.subtitle', { metric: t(`gotocc.dashboard.usageChart.metrics.${metric.value}`) }))

// 命中率不是可加总的量，排行按 Token 用量排序。
const amountOf = (model: ModelStat, key: UsageMetric): number => {
  if (key === 'requests') return model.requests
  if (key === 'cost') return model.actual_cost
  return model.total_tokens
}

const formatAmount = (amount: number): string => {
  if (metric.value === 'requests') return formatNumber(amount)
  if (metric.value === 'cost') return formatCostAuto(amount)
  return formatTokensK(amount)
}

const formatModelValue = (model: ModelStat): string => {
  if (metric.value !== 'cacheHitRate') return formatAmount(amountOf(model, metric.value))
  const rate = cacheHitRateOf(model.input_tokens, model.cache_creation_tokens, model.cache_read_tokens)
  return rate === null ? '—' : t('gotocc.dashboard.topModels.hitRate', { rate: `${rate.toFixed(1)}%` })
}

const formatShare = (share: number): string => `${share < 10 ? share.toFixed(1) : Math.round(share)}%`

const ranking = computed(() => {
  const sorted = models.value
    .filter((model) => amountOf(model, metric.value) > 0)
    .sort((a, b) => amountOf(b, metric.value) - amountOf(a, metric.value))
  const total = sorted.reduce((sum, model) => sum + amountOf(model, metric.value), 0)
  const shareOf = (amount: number) => (amount / total) * 100
  const rest = sorted.slice(TOP_COUNT)
  const restAmount = rest.reduce((sum, model) => sum + amountOf(model, metric.value), 0)
  return {
    top: sorted.slice(0, TOP_COUNT).map((model) => ({ model, share: shareOf(amountOf(model, metric.value)) })),
    others: rest.length > 0 ? { count: rest.length, amount: restAmount, share: shareOf(restAmount) } : null,
  }
})

const rows = computed(() => ranking.value.top.map(({ model, share }) => ({
  model: model.model,
  value: formatModelValue(model),
  share,
  shareText: formatShare(share),
  active: filters.value.model === model.model,
})))

const others = computed(() => {
  const rest = ranking.value.others
  return rest && { count: rest.count, value: formatAmount(rest.amount), shareText: formatShare(rest.share) }
})
</script>

<style scoped>
/* 占比条从 0 伸展到目标宽度，前一行先动；行 key 变化时重新播放。 */
.top-model-bar {
  width: var(--bar-w);
  background-image: linear-gradient(90deg, theme('colors.primary.500'), theme('colors.accent.500'));
  animation: top-model-bar var(--dash-top-model-bar-ms) var(--motion-ease) var(--bar-delay) both;
}

@keyframes top-model-bar {
  from {
    width: 0;
  }
  to {
    width: var(--bar-w);
  }
}

@media (prefers-reduced-motion: reduce) {
  .top-model-bar {
    animation: none;
  }
}
</style>

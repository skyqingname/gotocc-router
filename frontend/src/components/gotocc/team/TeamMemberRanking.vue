<template>
  <div class="card flex min-w-0 flex-col p-4" :style="{ '--bar-ms': `${TOP_MODEL_BAR_MS}ms` }">
    <div class="mb-4 flex min-h-7 items-baseline gap-2">
      <h3 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
        <Icon name="users" size="sm" class="text-primary-600 dark:text-primary-400" />
        {{ t('team.memberRanking') }}
      </h3>
      <span class="truncate text-xs text-gray-500 dark:text-dark-400">{{ t('gotocc.dashboard.topModels.subtitle', { metric: t(`gotocc.dashboard.usageChart.metrics.${metric}`) }) }}</span>
    </div>
    <div v-if="!loaded" class="space-y-4" aria-hidden="true">
      <div v-for="row in 4" :key="row" class="space-y-2 px-2"><Skeleton width="60%" :height="16" /><Skeleton width="100%" :height="4" /></div>
    </div>
    <p v-else-if="rows.length === 0" class="flex flex-1 items-center justify-center py-8 text-sm text-gray-500 dark:text-dark-400">{{ t('team.noUsage') }}</p>
    <!-- 点击成员打开详情抽屉 -->
    <ol v-else class="flex flex-1 flex-col gap-1">
      <li v-for="(row, index) in rows" :key="`${row.userId}-${version}`">
        <!-- 已离队成员没有成员详情，只展示排行 -->
        <button type="button" class="w-full rounded-lg px-2 py-2 text-left transition-colors enabled:hover:bg-gray-50 dark:enabled:hover:bg-dark-700" :disabled="row.left" :title="row.left ? undefined : t('team.viewUsage')" @click="emit('open', row.userId)">
          <span class="flex min-w-0 items-center gap-2">
            <span class="w-4 shrink-0 text-center text-xs font-semibold tabular-nums" :class="index === 0 ? 'text-primary-600 dark:text-primary-400' : 'text-gray-400 dark:text-dark-500'">{{ index + 1 }}</span>
            <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[10px] font-semibold text-white" :style="{ backgroundColor: row.color }">{{ initialsOf(row.name) }}</span>
            <span class="min-w-0 flex-1 truncate text-sm text-gray-800 dark:text-dark-100">{{ row.name }}</span>
            <span v-if="row.left" class="badge badge-gray shrink-0">{{ t('team.leftMember') }}</span>
            <span class="shrink-0 text-sm font-medium tabular-nums text-gray-900 dark:text-white">{{ row.value }}</span>
          </span>
          <span class="mt-2 flex items-center gap-2 pl-6">
            <span class="h-1 flex-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
              <span class="share-bar block h-full rounded-full" :style="{ '--bar-w': `${row.share}%`, '--bar-delay': `${index * TOP_MODEL_BAR_STEP_MS}ms`, backgroundColor: row.color }"></span>
            </span>
            <span class="w-12 shrink-0 text-right text-xs tabular-nums text-gray-500 dark:text-dark-400">{{ row.share < 10 ? row.share.toFixed(1) : Math.round(row.share) }}%</span>
          </span>
        </button>
      </li>
    </ol>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import type { TeamMemberUsageSeries } from '@/api/team'
import { TOP_MODEL_BAR_MS, TOP_MODEL_BAR_STEP_MS } from '@/components/gotocc/dashboard/motion'
import { initialsOf } from './teamFormat'
import { formatMetric, memberColor, totalValue, type TeamMetric } from './teamMetrics'

// 成员排行：按当前指标排序，占比条用成员色，与成员走势一致。
const props = defineProps<{ series: TeamMemberUsageSeries[]; metric: TeamMetric; loaded: boolean }>()
const emit = defineEmits<{ (event: 'open', userId: number): void }>()
const { t } = useI18n()

const version = ref(0)
watch(() => [props.series, props.metric], () => {
  version.value += 1
})

const rows = computed(() => {
  const items = props.series
    .map((item, index) => ({ item, color: memberColor(index), amount: totalValue(item.summary, props.metric) }))
    .filter((row) => row.amount > 0)
    .sort((a, b) => b.amount - a.amount)
  const total = items.reduce((sum, row) => sum + row.amount, 0)
  return items.map((row) => ({
    userId: row.item.actor_user_id,
    name: row.item.display_name,
    left: row.item.status === 'left',
    color: row.color,
    value: formatMetric(row.amount, props.metric),
    share: (row.amount / total) * 100,
  }))
})
</script>

<style scoped>
.share-bar {
  width: var(--bar-w);
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

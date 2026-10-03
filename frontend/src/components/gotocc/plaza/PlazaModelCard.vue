<template>
  <article class="plaza-card card flex min-w-0 flex-col gap-3 p-4">
    <!-- 名称、计费方式、厂商、用途与生效倍率 -->
    <header class="flex items-start gap-3">
      <ModelIcon :model="model.name" size="22px" class="mt-0.5 shrink-0" />
      <div class="min-w-0 flex-1">
        <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white" :title="model.name">{{ model.name }}</h3>
        <div class="mt-1.5 flex flex-wrap gap-1.5">
          <span class="plaza-tag bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-200">{{ t(billingLabelKey(model)) }}</span>
          <span v-if="info" class="plaza-tag bg-primary-50 text-primary-700 dark:bg-primary-500/15 dark:text-primary-300">{{ vendorLabel(info.vendor) }}</span>
          <span v-for="purpose in info?.purposes ?? []" :key="purpose" class="plaza-tag" :class="PURPOSE_CLASSES[purpose] ?? PURPOSE_CLASSES.language">{{ t(`gotocc.plaza.purposes.${purpose}`) }}</span>
        </div>
      </div>
      <span class="shrink-0 rounded-full bg-primary-600 px-2 py-0.5 text-xs font-semibold tabular-nums text-white" :title="t('modelPlaza.table.rate')">{{ formatRate(rate) }}</span>
    </header>

    <p v-if="info?.description" class="line-clamp-2 text-sm leading-6 text-gray-600 dark:text-dark-300" :title="info.description">{{ info.description }}</p>

    <!-- 近 24 小时状态（全站汇总） -->
    <div class="flex items-center gap-2 rounded-lg bg-gray-50 px-3 py-2 text-xs dark:bg-dark-900/60">
      <span class="h-2 w-2 shrink-0 rounded-full" :class="healthClass"></span>
      <span class="text-gray-500 dark:text-dark-400">{{ t('gotocc.plaza.last24h') }}</span>
      <span v-if="model.stats" class="truncate tabular-nums text-gray-700 dark:text-dark-200">
        {{ t('gotocc.plaza.statsLine', { requests: model.stats.requests.toLocaleString(), rate: successText }) }}<template v-if="firstTokenText"> · {{ t('gotocc.plaza.firstToken', { time: firstTokenText }) }}</template>
      </span>
      <span v-else class="text-gray-400 dark:text-dark-500">{{ t('gotocc.plaza.noRequests') }}</span>
    </div>

    <!-- 实付价格（折后） -->
    <section class="rounded-xl border border-primary-100 bg-primary-50/40 p-3 dark:border-primary-500/20 dark:bg-primary-500/5">
      <p class="mb-2 flex items-baseline gap-2 text-xs">
        <span class="font-medium text-primary-700 dark:text-primary-300">{{ t('modelPlaza.table.paidPrice') }}</span>
        <span class="text-gray-400 dark:text-dark-500">{{ t(unitKey(model)) }}</span>
      </p>
      <p v-if="!model.pricing" class="text-sm text-gray-500 dark:text-dark-400">{{ t('modelPlaza.detail.noPricing') }}</p>
      <div v-else-if="isTokenModel(model)" class="grid grid-cols-3 gap-2">
        <div v-for="column in tokenColumns" :key="column.key" class="min-w-0 rounded-lg bg-white px-2.5 py-2 dark:bg-dark-800">
          <p class="text-[11px] text-gray-500 dark:text-dark-400">{{ column.label }}</p>
          <p v-for="line in column.lines" :key="line.label" class="mt-0.5 flex items-baseline justify-between gap-1 text-xs">
            <span class="shrink-0 text-gray-400 dark:text-dark-500">{{ line.label }}</span>
            <span class="truncate font-semibold tabular-nums text-gray-900 dark:text-white">{{ line.value }}</span>
          </p>
        </div>
      </div>
      <div v-else class="space-y-1 rounded-lg bg-white px-3 py-2 dark:bg-dark-800">
        <p class="flex items-baseline gap-1">
          <span class="text-lg font-semibold tabular-nums text-gray-900 dark:text-white">{{ paidPerUnit(model.pricing.per_request_price, rate) }}</span>
          <span class="text-xs text-gray-500 dark:text-dark-400">{{ t(perUnitKey(model)) }}</span>
        </p>
        <p v-for="interval in requestIntervals(model)" :key="tierLabel(interval)" class="flex justify-between text-xs">
          <span class="text-gray-500 dark:text-dark-400">{{ tierLabel(interval) }}</span>
          <span class="font-medium tabular-nums text-gray-800 dark:text-dark-100">{{ paidPerUnit(interval.per_request_price, rate) }}</span>
        </p>
      </div>
    </section>

    <!-- 官方价格（不乘倍率，取基础档） -->
    <p v-if="officialLine" class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400">
      <Icon name="infoCircle" size="xs" class="shrink-0" />
      <span class="truncate">{{ t('modelPlaza.table.officialPrice') }} · {{ t('modelPlaza.table.unitPerMillion') }} · {{ officialLine }}</span>
    </p>

    <footer class="mt-auto flex items-center justify-between gap-2 pt-1 text-xs">
      <span class="truncate text-gray-500 dark:text-dark-400">{{ factsLine }}</span>
      <button type="button" class="inline-flex shrink-0 items-center gap-1 font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300" @click="emit('detail')">
        {{ t('gotocc.plaza.viewDetail') }}
        <Icon name="chevronRight" size="xs" />
      </button>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import {
  billingLabelKey,
  formatRate,
  formatTokenCount,
  hasCachePricing,
  hasTierCachePricing,
  isTokenModel,
  modelRate,
  officialPerMillion,
  paidPerMillion,
  paidPerUnit,
  perUnitKey,
  requestIntervals,
  tierLabel,
  tokenIntervals,
  unitKey,
} from './plazaPricing'
import { vendorLabel } from './vendors'

// 模型广场卡片：名称与标签、简介、近 24 小时状态、实付分档价、官方价和上下文。
const props = defineProps<{ group: ModelPlazaGroup; model: PlazaModel }>()
const emit = defineEmits<{ (event: 'detail'): void }>()
const { t } = useI18n()

// 用途标签配色。类名写成完整字符串，保证 Tailwind 能扫描到。
const PURPOSE_CLASSES: Record<string, string> = {
  language: 'bg-sky-50 text-sky-700 dark:bg-sky-500/15 dark:text-sky-300',
  image: 'bg-amber-50 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
  video: 'bg-rose-50 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300',
  audio: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300',
}

const info = computed(() => props.model.info)
const rate = computed(() => modelRate(props.group, props.model))

const successText = computed(() => (props.model.stats?.success_rate == null ? '—' : `${props.model.stats.success_rate.toFixed(1)}%`))
const firstTokenText = computed(() => {
  const ms = props.model.stats?.avg_first_token_ms
  if (ms == null) return ''
  return ms < 1000 ? `${Math.round(ms)}ms` : `${(ms / 1000).toFixed(1)}s`
})
// 成功率 ≥ 99% 绿色，≥ 95% 琥珀，其余红色；没有请求时灰色。
const healthClass = computed(() => {
  const value = props.model.stats?.success_rate
  if (value == null) return 'bg-gray-300 dark:bg-dark-500'
  if (value >= 99) return 'bg-emerald-500'
  if (value >= 95) return 'bg-amber-500'
  return 'bg-red-500'
})

// 基础价与各档共有的单价字段。
type PriceSource = { input_price?: number | null; output_price?: number | null; cache_write_price?: number | null; cache_read_price?: number | null }

// token 模型三列：输入、输出、缓存；有分档时输入、输出逐档列出，缓存列取最低档，完整分档见详情。
const tokenColumns = computed(() => {
  const pricing = props.model.pricing!
  const intervals = tokenIntervals(props.model)
  const lines = (pick: (source: PriceSource) => number | null | undefined) => (intervals.length > 0
    ? intervals.map((interval) => ({ label: tierLabel(interval), value: paidPerMillion(pick(interval), rate.value) }))
    : [{ label: '', value: paidPerMillion(pick(pricing), rate.value) }])
  const cacheSource: PriceSource = intervals.length > 0 && hasTierCachePricing(intervals) ? intervals[0] : pricing
  const cacheLines = hasCachePricing(props.model)
    ? [
        { label: t('modelPlaza.table.cacheWriteShort'), value: paidPerMillion(cacheSource.cache_write_price, rate.value) },
        { label: t('modelPlaza.table.cacheReadShort'), value: paidPerMillion(cacheSource.cache_read_price, rate.value) },
      ]
    : [{ label: '', value: '-' }]
  return [
    { key: 'input', label: t('modelPlaza.table.input'), lines: lines((source) => source.input_price) },
    { key: 'output', label: t('modelPlaza.table.output'), lines: lines((source) => source.output_price) },
    { key: 'cache', label: t('modelPlaza.table.cache'), lines: cacheLines },
  ]
})

// 官方价基础档：输入 · 输出。
const officialLine = computed(() => {
  const official = props.model.official_pricing
  if (!official || !isTokenModel(props.model) || official.output_price == null) return ''
  return `${t('modelPlaza.table.input')} ${officialPerMillion(official.input_price)} · ${t('modelPlaza.table.output')} ${officialPerMillion(official.output_price)}`
})

const factsLine = computed(() => {
  const value = info.value
  if (!value?.context_window) return value?.release_date ? t('gotocc.plaza.released', { date: value.release_date }) : ''
  const parts = [t('gotocc.plaza.context', { size: formatTokenCount(value.context_window) })]
  if (value.max_output_tokens) parts.push(t('gotocc.plaza.maxOutput', { size: formatTokenCount(value.max_output_tokens) }))
  return parts.join(' · ')
})
</script>

<style scoped>
.plaza-card {
  transition:
    transform var(--motion-normal) var(--motion-ease),
    box-shadow var(--motion-normal) var(--motion-ease),
    border-color var(--motion-normal) var(--motion-ease);
}

.plaza-card:hover {
  @apply border-primary-200 shadow-card-hover dark:border-primary-500/30;
  transform: translateY(-2px);
}

.plaza-tag {
  @apply inline-flex items-center rounded-md px-1.5 py-0.5 text-[11px] font-medium;
}

@media (prefers-reduced-motion: reduce) {
  .plaza-card:hover {
    transform: none;
  }
}
</style>

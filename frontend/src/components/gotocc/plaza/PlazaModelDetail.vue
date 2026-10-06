<template>
  <BaseDialog :show="Boolean(model && group)" :title="model?.name ?? ''" width="wide" @close="emit('close')">
    <div v-if="model && group" class="space-y-5 text-sm">
      <!-- 名称、厂商、用途与简介 -->
      <div class="flex items-start gap-3">
        <ModelIcon :model="model.name" size="32px" class="shrink-0" />
        <div class="min-w-0 flex-1">
          <p class="flex flex-wrap items-center gap-2">
            <span class="text-base font-semibold text-gray-900 dark:text-white">{{ info?.display_name || model.name }}</span>
            <span v-if="info" class="rounded-md bg-primary-50 px-1.5 py-0.5 text-[11px] font-medium text-primary-700 dark:bg-primary-500/15 dark:text-primary-300">{{ vendorLabel(info.vendor) }}</span>
            <span v-for="purpose in info?.purposes ?? []" :key="purpose" class="rounded-md bg-gray-100 px-1.5 py-0.5 text-[11px] font-medium text-gray-600 dark:bg-dark-700 dark:text-dark-200">{{ t(`gotocc.plaza.purposes.${purpose}`) }}</span>
          </p>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ group.name }} · {{ t('modelPlaza.table.rate') }} {{ formatRate(rate) }}</p>
        </div>
      </div>
      <p v-if="info?.description" class="leading-6 text-gray-700 dark:text-dark-200">{{ info.description }}</p>

      <!-- 视频模型能力与接入示例 -->
      <div v-if="model.video" class="space-y-2 rounded-xl bg-rose-50/60 p-4 dark:bg-rose-500/5">
        <p class="text-xs font-medium text-rose-700 dark:text-rose-300">{{ t('gotocc.plaza.videoCapabilities') }}</p>
        <div class="flex flex-wrap gap-1.5">
          <span v-for="item in videoCapabilityItems(model.video.capabilities)" :key="item" class="rounded-full border border-rose-200 bg-white px-2.5 py-0.5 text-xs text-rose-700 dark:border-rose-500/30 dark:bg-dark-800 dark:text-rose-300">{{ item }}</span>
        </div>
        <RouterLink :to="{ path: '/docs/video', query: { model: model.name } }" class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400">
          {{ t('gotocc.plaza.videoExamples') }}<Icon name="chevronRight" size="xs" />
        </RouterLink>
      </div>

      <!-- 官方信息 -->
      <dl v-if="info?.source" class="grid grid-cols-2 gap-x-4 gap-y-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-900/60 sm:grid-cols-3">
        <div v-for="fact in facts" :key="fact.label">
          <dt class="text-xs text-gray-500 dark:text-dark-400">{{ fact.label }}</dt>
          <dd class="mt-0.5 font-medium text-gray-900 dark:text-white">{{ fact.value }}</dd>
        </div>
      </dl>
      <div v-if="capabilities.length" class="flex flex-wrap gap-1.5">
        <span v-for="capability in capabilities" :key="capability" class="inline-flex items-center gap-1 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-0.5 text-xs text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-300">
          <Icon name="check" size="xs" />{{ capability }}
        </span>
      </div>

      <!-- 近 24 小时 -->
      <div class="flex flex-wrap items-center gap-x-5 gap-y-1 rounded-xl border border-gray-100 px-4 py-3 text-xs dark:border-dark-700">
        <span class="font-medium text-gray-700 dark:text-dark-200">{{ t('gotocc.plaza.last24h') }}</span>
        <template v-if="model.stats">
          <span>{{ t('gotocc.plaza.requests') }} <b class="tabular-nums">{{ model.stats.requests.toLocaleString() }}</b></span>
          <span>{{ t('gotocc.plaza.successRate') }} <b class="tabular-nums">{{ model.stats.success_rate == null ? '—' : `${model.stats.success_rate.toFixed(1)}%` }}</b></span>
          <span v-if="model.stats.avg_first_token_ms != null">{{ t('gotocc.plaza.avgFirstToken') }} <b class="tabular-nums">{{ formatMs(model.stats.avg_first_token_ms) }}</b></span>
        </template>
        <span v-else class="text-gray-400 dark:text-dark-500">{{ t('gotocc.plaza.noRequests') }}</span>
      </div>

      <!-- 实付价格与官方价格，token 模型逐档列出 -->
      <section class="overflow-x-auto rounded-xl border border-gray-100 dark:border-dark-700">
        <table class="min-w-full text-xs">
          <thead class="bg-gray-50 text-left text-gray-500 dark:bg-dark-900/60 dark:text-dark-400">
            <tr>
              <th class="px-3 py-2 font-medium">{{ t('gotocc.plaza.priceKind') }}</th>
              <th v-if="isTokenModel(model)" class="px-3 py-2 font-medium">{{ t('gotocc.plaza.tier') }}</th>
              <template v-if="isTokenModel(model)">
                <th class="px-3 py-2 text-right font-medium">{{ t('modelPlaza.table.input') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('modelPlaza.table.output') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('modelPlaza.table.cacheWrite') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('modelPlaza.table.cacheRead') }}</th>
              </template>
              <th v-else class="px-3 py-2 text-right font-medium">{{ t(unitKey(model)) }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-50 dark:divide-dark-700/60">
            <tr v-for="row in priceRows" :key="row.key" :class="row.paid ? 'text-gray-900 dark:text-white' : 'text-gray-500 dark:text-dark-400'">
              <td class="whitespace-nowrap px-3 py-2">{{ row.kind }}</td>
              <td v-if="isTokenModel(model)" class="whitespace-nowrap px-3 py-2">{{ row.tier }}</td>
              <td v-for="(value, index) in row.values" :key="index" class="whitespace-nowrap px-3 py-2 text-right tabular-nums" :class="{ 'font-semibold': row.paid }">{{ value }}</td>
            </tr>
          </tbody>
        </table>
      </section>
      <p v-if="isTokenModel(model)" class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.unitPerMillion') }} · {{ model.long_context_basis === 'marginal' ? t('modelPlaza.table.tierHintMarginal') : t('modelPlaza.table.tierHint') }}</p>

      <!-- 推理强度倍率与分时倍率 -->
      <div v-if="reasoningMultipliers.length" class="flex flex-wrap gap-1.5">
        <span v-for="[effort, multiplier] in reasoningMultipliers" :key="effort" class="rounded-md bg-violet-50 px-2 py-0.5 text-xs text-violet-700 dark:bg-violet-500/15 dark:text-violet-300" :title="t('modelPlaza.table.reasoningMultiplierHint', { effort, multiplier })">
          {{ t('modelPlaza.table.reasoningMultiplierBadge', { effort, multiplier }) }}
        </span>
      </div>
      <ul v-if="model.time_pricing?.periods.length" class="space-y-1 text-xs text-gray-600 dark:text-dark-300">
        <li v-for="period in model.time_pricing.periods" :key="period.start_time">
          {{ period.start_time.slice(0, 5) }}–{{ period.end_time.slice(0, 5) }}<template v-if="model.time_pricing.weekdays_only"> · {{ t('modelPlaza.table.timePricingWeekdays') }}</template> · ×{{ period.multiplier }}
        </li>
      </ul>

      <p v-if="info?.source" class="text-xs text-gray-400 dark:text-dark-500">{{ t('gotocc.plaza.sourceNote') }}</p>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { REASONING_EFFORT_LEVELS } from '@/constants/channel'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import type { UserPricingInterval } from '@/api/channels'
import {
  formatRate,
  formatTokenCount,
  isTokenModel,
  modelRate,
  officialIntervals,
  officialPerMillion,
  paidPerMillion,
  paidPerUnit,
  requestIntervals,
  tierLabel,
  tokenIntervals,
  unitKey,
} from './plazaPricing'
import { vendorLabel } from './vendors'
import { videoCapabilityItems } from '../video/videoDocs'

// 模型详情：官方信息（models.dev）、能力、近 24 小时状态、实付与官方的完整分档价。
const props = defineProps<{ group: ModelPlazaGroup | null; model: PlazaModel | null }>()
const emit = defineEmits<{ (event: 'close'): void }>()
const { t } = useI18n()

const info = computed(() => props.model?.info)
const rate = computed(() => modelRate(props.group!, props.model!))

const modalityText = (values: string[] | null | undefined) => (values ?? []).map((value) => t(`gotocc.plaza.modalities.${value}`)).join(' / ') || '—'

const facts = computed(() => {
  const value = info.value!
  return [
    { label: t('gotocc.plaza.facts.context'), value: value.context_window ? formatTokenCount(value.context_window) : '—' },
    { label: t('gotocc.plaza.facts.maxOutput'), value: value.max_output_tokens ? formatTokenCount(value.max_output_tokens) : '—' },
    { label: t('gotocc.plaza.facts.input'), value: modalityText(value.input_modalities) },
    { label: t('gotocc.plaza.facts.output'), value: modalityText(value.output_modalities) },
    { label: t('gotocc.plaza.facts.knowledge'), value: value.knowledge || '—' },
    { label: t('gotocc.plaza.facts.released'), value: value.release_date || '—' },
  ]
})

const capabilities = computed(() => {
  const value = info.value
  if (!value?.source) return []
  return [
    value.reasoning && t('gotocc.plaza.capabilities.reasoning'),
    value.tool_call && t('gotocc.plaza.capabilities.toolCall'),
    value.structured_output && t('gotocc.plaza.capabilities.structuredOutput'),
    value.attachment && t('gotocc.plaza.capabilities.attachment'),
    value.open_weights && t('gotocc.plaza.capabilities.openWeights'),
  ].filter((item): item is string => Boolean(item))
})

const formatMs = (ms: number) => (ms < 1000 ? `${Math.round(ms)}ms` : `${(ms / 1000).toFixed(2)}s`)

type PriceSource = { input_price?: number | null; output_price?: number | null; cache_write_price?: number | null; cache_read_price?: number | null }

// 价格表行：实付在前（乘生效倍率），官方在后（不乘倍率）；分档模型逐档一行。
const priceRows = computed(() => {
  const model = props.model!
  const rows: Array<{ key: string; kind: string; tier: string; paid: boolean; values: string[] }> = []
  if (!isTokenModel(model)) {
    rows.push({ key: 'paid', kind: t('modelPlaza.table.paidPrice'), tier: '', paid: true, values: [paidPerUnit(model.pricing?.per_request_price, rate.value)] })
    requestIntervals(model).forEach((interval) => rows.push({ key: `paid-${tierLabel(interval)}`, kind: tierLabel(interval), tier: '', paid: true, values: [paidPerUnit(interval.per_request_price, rate.value)] }))
    return rows
  }
  const tokenValues = (source: PriceSource, format: (value: number | null | undefined) => string) =>
    [format(source.input_price), format(source.output_price), format(source.cache_write_price), format(source.cache_read_price)]
  const addRows = (kind: string, paid: boolean, base: PriceSource | null | undefined, intervals: UserPricingInterval[], format: (value: number | null | undefined) => string) => {
    if (!base) return
    if (intervals.length === 0) {
      rows.push({ key: `${kind}-base`, kind, tier: '—', paid, values: tokenValues(base, format) })
      return
    }
    intervals.forEach((interval) => rows.push({ key: `${kind}-${tierLabel(interval)}`, kind, tier: tierLabel(interval), paid, values: tokenValues(interval, format) }))
  }
  addRows(t('modelPlaza.table.paidPrice'), true, model.pricing, tokenIntervals(model), (value) => paidPerMillion(value, rate.value))
  addRows(t('modelPlaza.table.officialPrice'), false, model.official_pricing, officialIntervals(model), officialPerMillion)
  return rows
})

const reasoningMultipliers = computed(() => {
  const multipliers = props.model?.pricing?.reasoning_effort_multipliers
  return REASONING_EFFORT_LEVELS.flatMap((effort) => {
    const multiplier = multipliers?.[effort]
    return typeof multiplier === 'number' && multiplier > 0 ? [[effort, multiplier] as [string, number]] : []
  })
})
</script>

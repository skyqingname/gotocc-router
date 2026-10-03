import { formatScaled, resolveIntervalPrices } from '@/utils/pricing'
import { BILLING_MODE_IMAGE, BILLING_MODE_PER_REQUEST, BILLING_MODE_TOKEN, BILLING_MODE_VIDEO, type BillingMode } from '@/constants/channel'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import type { UserPricingInterval } from '@/api/channels'

// 模型广场卡片与详情共用的计价规则，与原价目表（components/modelPlaza/PlazaModelPricingTable.vue）一致：
// 实付价 = 渠道单价 × 生效倍率；图片、视频可分别使用独立倍率；官方参考价不乘倍率。

const PER_MILLION = 1_000_000
// 价格保底两位小数，更长的有效小数原样保留。
const MIN_DECIMALS = 2

export const billingMode = (model: PlazaModel): BillingMode => (model.pricing?.billing_mode || BILLING_MODE_TOKEN) as BillingMode

export const isTokenModel = (model: PlazaModel): boolean => billingMode(model) === BILLING_MODE_TOKEN

// 生效倍率 = 用户专属倍率 ?? 分组默认倍率。
export const groupRate = (group: ModelPlazaGroup): number => group.user_rate_multiplier ?? group.rate_multiplier

// 单个模型的生效倍率：图片、视频开启独立倍率时取独立倍率。
export const modelRate = (group: ModelPlazaGroup, model: PlazaModel): number => {
  if (billingMode(model) === BILLING_MODE_VIDEO && group.video_rate_independent) return Math.max(0, group.video_rate_multiplier)
  if (billingMode(model) === BILLING_MODE_IMAGE && group.image_rate_independent) return group.image_rate_multiplier
  return groupRate(group)
}

export const paidPerMillion = (value: number | null | undefined, rate: number): string =>
  value == null ? '-' : formatScaled(value * rate, PER_MILLION, MIN_DECIMALS)

export const paidPerUnit = (value: number | null | undefined, rate: number): string =>
  value == null ? '-' : formatScaled(value * rate, 1, MIN_DECIMALS)

export const officialPerMillion = (value: number | null | undefined): string =>
  value == null ? '-' : formatScaled(value, PER_MILLION, MIN_DECIMALS)

const byContext = (intervals: UserPricingInterval[]) => [...intervals].sort((a, b) => a.min_tokens - b.min_tokens)

// token 模型的实付档位，档位可给绝对价或相对基础价的倍率。
export const tokenIntervals = (model: PlazaModel): UserPricingInterval[] =>
  byContext(model.pricing?.intervals ?? []).map((interval) => resolveIntervalPrices(interval, model.pricing!))

export const officialIntervals = (model: PlazaModel): UserPricingInterval[] => byContext(model.official_pricing?.intervals ?? [])

// 按次 / 按图 / 按秒模型只保留配了单价的档位。
export const requestIntervals = (model: PlazaModel): UserPricingInterval[] =>
  (model.pricing?.intervals ?? []).filter((interval) => interval.per_request_price != null)

const trimZero = (value: number) => String(Math.round(value * 100) / 100)

export const formatTokenCount = (count: number): string => {
  if (count >= 1_000_000) return `${trimZero(count / 1_000_000)}M`
  if (count >= 1_000) return `${trimZero(count / 1_000)}K`
  return String(count)
}

// 档位标签：优先后端给出的 tier_label；否则有上限为「≤上限」，末档为「>下限」。
export const tierLabel = (interval: UserPricingInterval): string => {
  if (interval.tier_label) return interval.tier_label
  return interval.max_tokens == null ? `>${formatTokenCount(interval.min_tokens)}` : `≤${formatTokenCount(interval.max_tokens)}`
}

export const hasCachePricing = (model: PlazaModel): boolean =>
  model.pricing?.cache_write_price != null || model.pricing?.cache_write_1h_price != null || model.pricing?.cache_read_price != null

// 任一档带缓存价时，缓存列按档展示。
export const hasTierCachePricing = (intervals: UserPricingInterval[]): boolean =>
  intervals.some((interval) => interval.cache_write_price != null || interval.cache_write_1h_price != null || interval.cache_read_price != null)

// 价格表头的单位文案键。
export const unitKey = (model: PlazaModel): string => ({
  [BILLING_MODE_TOKEN]: 'modelPlaza.table.unitPerMillion',
  [BILLING_MODE_PER_REQUEST]: 'modelPlaza.table.unitPerRequest',
  [BILLING_MODE_IMAGE]: 'modelPlaza.table.unitPerImage',
  [BILLING_MODE_VIDEO]: 'modelPlaza.table.unitPerSecond',
})[billingMode(model)]

// 非 token 单价的后缀文案键。
export const perUnitKey = (model: PlazaModel): string => ({
  [BILLING_MODE_TOKEN]: 'modelPlaza.table.perUnitRequest',
  [BILLING_MODE_PER_REQUEST]: 'modelPlaza.table.perUnitRequest',
  [BILLING_MODE_IMAGE]: 'modelPlaza.table.perUnitImage',
  [BILLING_MODE_VIDEO]: 'modelPlaza.table.perUnitSecond',
})[billingMode(model)]

// 计费方式标签文案键。
export const billingLabelKey = (model: PlazaModel): string => ({
  [BILLING_MODE_TOKEN]: 'gotocc.plaza.billing.token',
  [BILLING_MODE_PER_REQUEST]: 'modelPlaza.table.perRequest',
  [BILLING_MODE_IMAGE]: 'modelPlaza.table.perImage',
  [BILLING_MODE_VIDEO]: 'modelPlaza.table.perSecondVideo',
})[billingMode(model)]

// 卡片排序：token 模型在前（官方 token 价与按张/按次价量纲不同），组内按官方输出价从高到低，同价按名称降序。
export const sortModels = (models: PlazaModel[]): PlazaModel[] => [...models].sort((a, b) => {
  if (isTokenModel(a) !== isTokenModel(b)) return isTokenModel(a) ? -1 : 1
  const pa = a.official_pricing?.output_price ?? null
  const pb = b.official_pricing?.output_price ?? null
  if (pa != null && pb != null && pa !== pb) return pb - pa
  if (pa != null && pb == null) return -1
  if (pa == null && pb != null) return 1
  return b.name.localeCompare(a.name)
})

// 倍率显示：去掉浮点噪声，如 0.8x。
export const formatRate = (rate: number): string => `${Math.round(rate * 1000) / 1000}x`

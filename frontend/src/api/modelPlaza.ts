/**
 * Model Plaza API（公开端点，可匿名访问）
 * 以分组为中心的模型价目：分组信息 + 模型渠道定价 + LiteLLM 官方参考价。
 * 带 token 请求时后端会额外返回专属分组与用户专属倍率。
 */

import { apiClient } from './client'
import type { UserPricingInterval, UserSupportedModelPricing } from './channels'
import type { VideoCapabilities } from '@/components/admin/channel/video-models'

/** 官方参考价（USD per token，与计费目录同源；字段缺失 = 目录未覆盖）。 */
export interface PlazaOfficialPricing {
  input_price: number | null
  output_price: number | null
  /** 5m 缓存写入（= LiteLLM cache_creation）。 */
  cache_write_price: number | null
  /** 1h 缓存写入（LiteLLM cache_creation_above_1hr），多数模型缺失。 */
  cache_write_1h_price?: number | null
  cache_read_price: number | null
  /** 官方长上下文阶梯（多档模型才有），不受分组开关影响。 */
  intervals?: UserPricingInterval[]
}

/**
 * 多档时的计价基准：
 * - whole_request：整单按所在档单价计价（目录阶梯、渠道区间）；
 * - marginal：仅超出阈值的部分按该档单价计价（平台旧规则）。
 */
export type PlazaLongContextBasis = 'whole_request' | 'marginal'

/** 分时倍率时段：配置时区当天 [start_time, end_time) 内整单实付乘 multiplier。 */
export interface PlazaTimePricingPeriod {
  start_time: string
  end_time: string
  multiplier: number
}

/** 计费会生效的分时倍率（仅倍率 ≠ 1 的时段，已按开始时间升序）。 */
export interface PlazaTimePricing {
  /** IANA 时区名，如 Asia/Shanghai。 */
  timezone: string
  /** true 时时段仅周一至周五生效，周末整天按标准价计费。 */
  weekdays_only?: boolean
  periods: PlazaTimePricingPeriod[]
}

/** GoToCC：模型官方信息（随版 models.dev 快照，后台填写的简介/厂商/用途优先）。 */
export interface PlazaModelInfo {
  vendor: string
  vendor_name: string
  display_name: string
  description: string
  /** 简介来自后台填写。 */
  custom_description: boolean
  purposes: string[]
  context_window: number
  max_output_tokens: number
  input_modalities: string[] | null
  output_modalities: string[] | null
  reasoning: boolean
  tool_call: boolean
  structured_output: boolean
  attachment: boolean
  open_weights: boolean
  knowledge: string
  release_date: string
  /** 'models.dev' 表示官方目录中有该模型。 */
  source: string
}

/** GoToCC：全站近 24 小时的模型请求概况；success_rate 为百分比。 */
export interface PlazaModelStats {
  requests: number
  errors: number
  success_rate: number | null
  avg_first_token_ms: number | null
}

export interface PlazaModel {
  name: string
  platform: string
  /** 实收口径的展示定价：档位可提供绝对单价或相对基础价倍率；均为标准时段价。 */
  pricing: UserSupportedModelPricing | null
  official_pricing: PlazaOfficialPricing | null
  /** 仅多档模型返回。 */
  long_context_basis?: PlazaLongContextBasis
  /** 仅配置了分时倍率的模型返回。 */
  time_pricing?: PlazaTimePricing
  info?: PlazaModelInfo
  /** 近 24 小时没有请求时缺省。 */
  stats?: PlazaModelStats
  /** 仅 Video 分组模型返回：系列与能力，文档页据此生成示例。 */
  video?: PlazaVideoInfo
}

export interface PlazaVideoInfo {
  family: string
  /** 未填写能力时为 null，网关不做限制。 */
  capabilities: VideoCapabilities | null
}

export interface ModelPlazaGroup {
  id: number
  name: string
  description: string
  platform: string
  /** 'standard' | 'subscription' */
  subscription_type: string
  rate_multiplier: number
  /** 登录且管理员为该用户配了专属倍率时返回；生效倍率 = user_rate ?? rate_multiplier。 */
  user_rate_multiplier?: number
  peak_rate_enabled: boolean
  peak_start: string
  peak_end: string
  peak_rate_multiplier: number
  is_exclusive: boolean
  /** 生图独立倍率：true 时图片计费模型的实付倍率取 image_rate_multiplier，不取分组/专属倍率。 */
  image_rate_independent: boolean
  image_rate_multiplier: number
  /** 视频独立倍率开启时，覆盖视频模型的分组/用户专属倍率。 */
  video_rate_independent: boolean
  video_rate_multiplier: number
  /** 分组是否启用长上下文阶梯计费；false 时实付列只展示最低档，官方阶梯仅供参考。 */
  long_context_pricing_enabled: boolean
  models: PlazaModel[]
}

export interface ModelPlazaResponse {
  /** 管理员配置的全局价格说明（Markdown）。 */
  description: string
  groups: ModelPlazaGroup[]
}

/** 获取模型广场数据。开关未启用时后端返回 404。 */
export async function getModelPlaza(options?: { signal?: AbortSignal }): Promise<ModelPlazaResponse> {
  const { data } = await apiClient.get<ModelPlazaResponse>('/model-plaza', {
    signal: options?.signal
  })
  return data
}

export const modelPlazaAPI = { getModelPlaza }

export default modelPlazaAPI

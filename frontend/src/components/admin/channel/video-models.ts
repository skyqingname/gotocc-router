import defaults from '../../../../../video-platform-defaults.json'
import type { BillingMode } from '@/api/admin/channels'
import { createDefaultTimePricingForm, type IntervalFormEntry, type PricingFormEntry } from './types'

export interface VideoParameter {
  name: string
  label?: string
  disabled?: boolean
  type: string
  required: boolean
  values: string[]
  min?: number
  max?: number
  step?: number
}

// 模型对外能力：文档、模型广场展示，网关在扣费前按此校验请求。
export interface VideoCapabilities {
  min_seconds?: number
  max_seconds?: number
  fixed_seconds?: number[]
  resolutions?: string[]
  aspect_ratios?: string[]
  reference_images: boolean
  reference_videos: boolean
  reference_audios: boolean
  max_reference_images?: number
  max_reference_videos?: number
  max_reference_audios?: number
  max_reference_total?: number
  audio_output: boolean
}

export interface VideoModelConfig {
  enabled: boolean
  protocol: 'openai' | 'custom_json' | 'yingce'
  provider_definition?: Record<string, unknown>
  provider_id?: string
  provider_options?: Record<string, Record<string, unknown>>
  upstream_model: string
  create_status: string
  create_path: string
  status_path: string
  content_path: string
  headers: Record<string, string>
  defaults: Record<string, unknown>
  parameters: VideoParameter[]
  request_fields: Record<string, string>
  id_field: string
  status_field: string
  video_url_field: string
  statuses: Record<string, string>
  description?: string
  family?: string
  capabilities?: VideoCapabilities
}

export function newVideoModel(model: string): VideoModelConfig {
  return { ...JSON.parse(JSON.stringify(defaults)), upstream_model: model }
}

export interface VideoProtocolEntry {
 id: string; name: string; vendor: string; version: string; documentation: string; template: VideoModelConfig
 configuration: { fields: {name: string; type: string; label: string; required?: boolean}[] }
 definition: { create: {method:string;path:string;contentType?:string;body?:unknown};poll?:{method:string;path:string};result?:{method:string;path:string};auth:{type:string};parameters:{name:string;type:string;description?:string;mapping?:string;values?:string[]}[] }
 workflows: {id:string;label:string;defaults?:Record<string,unknown>;parameters?:unknown[]}[]
}

export const RESOLUTION_PLACEHOLDER = '{resolution}'
export const COMMON_RESOLUTIONS = ['480p', '720p', '768p', '1080p', '2K', '4K']
export const COMMON_ASPECT_RATIOS = ['16:9', '9:16', '1:1', '4:3', '3:4', '21:9']
export const COMMON_FAMILIES = ['Seedance', 'Wan', 'MiniMax 海螺', 'Grok', 'Kling', 'Veo', 'Vidu', 'Sora', '其他']

// 新模型的起始能力：三类参考素材均可传、不限数量，不生成音频；按上游说明再收紧。
export const newVideoCapabilities = (): VideoCapabilities => ({
  reference_images: true, reference_videos: true, reference_audios: true, audio_output: false,
})

// 尚未填写能力的旧模型：网关不做限制，表单按「全部允许」展示。
export const unrestrictedVideoCapabilities = (): VideoCapabilities => ({
  reference_images: true, reference_videos: true, reference_audios: true, audio_output: true,
})

// 能力一行摘要，例如「4–15 秒 · 720p / 1080p · 图 9 · 视频 3 · 音频 3」。
export function capabilitySummary(caps?: VideoCapabilities): string {
  if (!caps) return '未填写能力（不限制）'
  const parts: string[] = []
  if (caps.fixed_seconds?.length) parts.push(`${caps.fixed_seconds.join(' / ')} 秒`)
  else if (caps.min_seconds || caps.max_seconds) parts.push(`${caps.min_seconds || 1}–${caps.max_seconds || '∞'} 秒`)
  if (caps.resolutions?.length) parts.push(caps.resolutions.join(' / '))
  const references = [
    caps.reference_images && `图 ${caps.max_reference_images || '不限'}`,
    caps.reference_videos && `视频 ${caps.max_reference_videos || '不限'}`,
    caps.reference_audios && `音频 ${caps.max_reference_audios || '不限'}`,
  ].filter(Boolean)
  parts.push(references.length ? references.join(' · ') : '仅文生')
  if (caps.audio_output) parts.push('可出声')
  return parts.join(' · ')
}

// ── 价格：存放在分组逐模型定价（model_pricing），每个视频模型一条规则 ──

export interface VideoModelPrice {
  mode: BillingMode
  // 不区分分辨率时的统一价
  price: number | null
  // 按分辨率的单价，键为分辨率档位
  tiers: Record<string, number | null>
}

const toPrice = (value: number | string | null | undefined): number | null =>
  value === null || value === undefined || value === '' ? null : Number(value)

export function readVideoModelPrice(pricing: PricingFormEntry[], model: string): VideoModelPrice {
  const entry = pricing.find((item) => item.models.includes(model))
  const tiers: Record<string, number | null> = {}
  for (const interval of entry?.intervals ?? []) {
    if (interval.tier_label) tiers[interval.tier_label] = toPrice(interval.per_request_price)
  }
  return { mode: entry?.billing_mode === 'per_request' ? 'per_request' : 'video', price: toPrice(entry?.per_request_price), tiers }
}

// 模型从原有共享规则中拆出，写成独立规则；未填写任何价格时不保留规则。
export function writeVideoModelPrice(pricing: PricingFormEntry[], model: string, price: VideoModelPrice): PricingFormEntry[] {
  const next = pricing
    .map((entry) => (entry.models.includes(model) ? { ...entry, models: entry.models.filter((name) => name !== model) } : entry))
    .filter((entry) => entry.models.length > 0)
  const tiers = Object.entries(price.tiers).filter((tier): tier is [string, number] => tier[1] !== null && !Number.isNaN(tier[1]))
  if (price.price === null && tiers.length === 0) return next
  next.push({
    models: [model],
    billing_mode: price.mode,
    input_price: null, output_price: null, cache_write_price: null, cache_write_1h_price: null, cache_read_price: null,
    reasoning_effort_multipliers: null, image_input_price: null, image_output_price: null,
    per_request_price: tiers.length ? null : price.price,
    intervals: tiers.map(([label, value], index): IntervalFormEntry => ({
      min_tokens: 0, max_tokens: null, tier_label: label,
      input_price: null, output_price: null, cache_write_price: null, cache_write_1h_price: null, cache_read_price: null,
      input_multiplier: null, output_multiplier: null, cache_write_multiplier: null, cache_read_multiplier: null,
      per_request_price: value, sort_order: index,
    })),
    time_pricing: createDefaultTimePricingForm(),
  })
  return next
}

export const renameVideoModelPrice = (pricing: PricingFormEntry[], from: string, to: string): PricingFormEntry[] =>
  pricing.map((entry) => ({ ...entry, models: entry.models.map((name) => (name === from ? to : name)) }))

export const removeVideoModelPrice = (pricing: PricingFormEntry[], model: string): PricingFormEntry[] =>
  pricing.map((entry) => ({ ...entry, models: entry.models.filter((name) => name !== model) })).filter((entry) => entry.models.length > 0)

// Model families follow TokenFlux/TokenRouter edca264117e63404bd77b6f04e2db0473f498d73 modelcatalog/media.go. Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation.
// 创作台按模型家族取像素表与可选参数，预估价沿用模型广场的实付口径。
import { BILLING_MODE_IMAGE, BILLING_MODE_PER_REQUEST } from '@/constants/channel'
import { billingMode, modelRate, requestIntervals } from '@/components/gotocc/plaza/plazaPricing'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import config from '../../../../../creative-studio.json'

export interface MediaFamily {
  model_pattern: string
  dimensions: Record<string, Record<string, string>>
  qualities: string[]
  backgrounds: string[]
  custom_size: boolean
}

// JSON 推断会把各家族的比例键并成可选字段，这里按登记结构读取。
const families = config.media_families as unknown as MediaFamily[]

// 按登记顺序取第一个匹配的家族；末项的空规则匹配其余模型。
export function mediaFamily(model: string): MediaFamily {
  const name = model.trim().toLowerCase()
  return families.find(family => new RegExp(family.model_pattern).test(name))!
}

export function familySizes(family: MediaFamily): string[] {
  return [...Object.values(family.dimensions).flatMap(ratios => Object.values(ratios)), ...(family.custom_size ? ['custom'] : [])]
}

// 与后端 ClassifyImageBillingTier 一致：按最长边落档。
export function billingTier(size: string): string {
  const longEdge = Math.max(...size.split('x').map(Number))
  return config.billing_tiers.find(tier => tier.max_edge === null || longEdge <= tier.max_edge)!.tier
}

// 单张实付价：同档位价优先，其次按张价，再乘模型广场的生效倍率；token 计费或未定价时为 null。
export function imageUnitPrice(groups: ModelPlazaGroup[], groupID: number | null | undefined, model: string, size: string): number | null {
  const group = groups.find(item => item.id === groupID)
  const offer = group?.models.find(item => item.name === model)
  if (!group || !offer?.pricing) return null
  const mode = billingMode(offer)
  if (mode !== BILLING_MODE_IMAGE && mode !== BILLING_MODE_PER_REQUEST) return null
  const tier = billingTier(size)
  const price = requestIntervals(offer).find(interval => interval.tier_label === tier)?.per_request_price ?? offer.pricing.per_request_price
  return price == null ? null : price * modelRate(group, offer)
}

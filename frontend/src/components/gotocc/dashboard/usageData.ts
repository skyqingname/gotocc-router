import type { TrendDataPoint } from '@/types'

// 用量时间窗口与汇总计算，移植自 TokenRouter（LGPL-3.0）。

// 快捷时间范围；custom 表示来自日期选择器的自定义范围。
export type UsageRangePreset = '24h' | '7d' | '30d' | 'custom'

export type UsageGranularity = 'hour' | 'day'

// 一次趋势查询的时间窗口，结束时间不包含在内。
export interface UsageRange {
  startAt: Date
  endAt: Date
  granularity: UsageGranularity
}

// 一段时间内各指标的合计；输入侧没有 token 时命中率记为 null。
export interface UsageTotals {
  requests: number
  tokens: number
  cost: number
  cacheHitRate: number | null
}

const HOUR_MS = 60 * 60 * 1000
// 跨度不超过两天时按小时聚合，避免折线只剩一两个点。
const HOURLY_MAX_SPAN_MS = 2 * 24 * HOUR_MS

const pad = (value: number): string => String(value).padStart(2, '0')

// 与后端按天分组一致的键（YYYY-MM-DD，本地时区）。
export const formatDayKey = (date: Date): string =>
  `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`

// 与后端按小时分组一致的键（YYYY-MM-DD HH:00，本地时区）。
export const formatHourKey = (date: Date): string => `${formatDayKey(date)} ${pad(date.getHours())}:00`

// 趋势接口接受的本地时间参数（YYYY-MM-DDTHH:mm:ss）。
export const formatQueryTime = (date: Date): string =>
  `${formatDayKey(date)}T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`

const startOfDay = (date: Date): Date => new Date(date.getFullYear(), date.getMonth(), date.getDate())

const addDays = (date: Date, days: number): Date => {
  const next = new Date(date)
  next.setDate(next.getDate() + days)
  return next
}

const resolveGranularity = (startAt: Date, endAt: Date): UsageGranularity =>
  endAt.getTime() - startAt.getTime() <= HOURLY_MAX_SPAN_MS ? 'hour' : 'day'

// 快捷范围换算成整点或整天对齐的查询窗口。
export const resolvePresetRange = (preset: Exclude<UsageRangePreset, 'custom'>, now: Date = new Date()): UsageRange => {
  if (preset === '24h') {
    const endAt = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours() + 1)
    return { startAt: new Date(endAt.getTime() - 24 * HOUR_MS), endAt, granularity: 'hour' }
  }
  const days = preset === '7d' ? 7 : 30
  const endAt = addDays(startOfDay(now), 1)
  return { startAt: addDays(endAt, -days), endAt, granularity: 'day' }
}

// 日期选择器给出 YYYY-MM-DD，结束日包含当天。
export const resolveCustomRange = (startValue: string, endValue: string): UsageRange => {
  const startAt = new Date(`${startValue}T00:00:00`)
  const endAt = addDays(new Date(`${endValue}T00:00:00`), 1)
  return { startAt, endAt, granularity: resolveGranularity(startAt, endAt) }
}

// 紧邻当前窗口之前、长度相同的窗口，用于环比。
export const previousRange = (range: UsageRange): UsageRange => {
  if (range.granularity === 'day') {
    // 按日历天回退，跨夏令时的一天也保持整天对齐。
    const days = Math.round((startOfDay(range.endAt).getTime() - startOfDay(range.startAt).getTime()) / (24 * HOUR_MS))
    return { startAt: addDays(range.startAt, -days), endAt: range.startAt, granularity: 'day' }
  }
  const span = range.endAt.getTime() - range.startAt.getTime()
  return { startAt: new Date(range.startAt.getTime() - span), endAt: range.startAt, granularity: 'hour' }
}

// 窗口内全部分组键，截止到当前时刻，未来的时段不画。
export const bucketKeys = (range: UsageRange, now: Date = new Date()): string[] => {
  const keys: string[] = []
  const limit = Math.min(range.endAt.getTime(), now.getTime() + 1)
  if (range.granularity === 'hour') {
    const cursor = new Date(range.startAt)
    cursor.setMinutes(0, 0, 0)
    while (cursor.getTime() < limit) {
      keys.push(formatHourKey(cursor))
      cursor.setHours(cursor.getHours() + 1)
    }
    return keys
  }
  let cursor = startOfDay(range.startAt)
  while (cursor.getTime() < limit) {
    keys.push(formatDayKey(cursor))
    cursor = addDays(cursor, 1)
  }
  return keys
}

const emptyPoint = (date: string): TrendDataPoint => ({
  date,
  requests: 0,
  input_tokens: 0,
  output_tokens: 0,
  cache_creation_tokens: 0,
  cache_read_tokens: 0,
  total_tokens: 0,
  cost: 0,
  actual_cost: 0,
})

// 按分组键补齐后端省略的空时段，保证横轴间隔均匀。
export const fillTrendBuckets = (points: TrendDataPoint[], keys: string[]): TrendDataPoint[] => {
  const byKey = new Map(points.map((point) => [point.date, point]))
  return keys.map((key) => byKey.get(key) ?? emptyPoint(key))
}

// 缓存命中率，分母是输入侧全部 token（后端的 input_tokens 已扣除缓存读取）。
export const cacheHitRateOf = (input: number, cacheCreation: number, cacheRead: number): number | null => {
  const inputSide = input + cacheCreation + cacheRead
  return inputSide > 0 ? (cacheRead / inputSide) * 100 : null
}

export const summarizeTrend = (points: TrendDataPoint[]): UsageTotals => {
  let requests = 0
  let tokens = 0
  let cost = 0
  let input = 0
  let cacheCreation = 0
  let cacheRead = 0
  for (const point of points) {
    requests += point.requests
    tokens += point.total_tokens
    cost += point.actual_cost
    input += point.input_tokens
    cacheCreation += point.cache_creation_tokens
    cacheRead += point.cache_read_tokens
  }
  return { requests, tokens, cost, cacheHitRate: cacheHitRateOf(input, cacheCreation, cacheRead) }
}

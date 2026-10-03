import type { TeamUsageDaily, TeamUsageModel, TeamUsageSummary } from '@/api/team'
import { formatDateLocalInput, formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import { formatTeamCost } from './teamFormat'

// 团队概览的指标：消费、请求、Token 可切换驱动图表和排行；活跃成员只在指标卡展示。
export type TeamMetric = 'cost' | 'requests' | 'tokens'
export type TeamCardMetric = TeamMetric | 'activeMembers'

export const TEAM_METRICS: TeamMetric[] = ['cost', 'requests', 'tokens']

// 成员配色从品牌靛色开始，按成员在序列中的位置取色，同一成员在趋势和排行中同色。
export const MEMBER_COLORS = ['#6366f1', '#0ea5e9', '#10b981', '#f59e0b', '#8b5cf6', '#f43f5e', '#14b8a6', '#ec4899']
export const memberColor = (index: number): string => MEMBER_COLORS[index % MEMBER_COLORS.length]

export const dailyValue = (point: TeamUsageDaily, metric: TeamCardMetric): number => {
  if (metric === 'cost') return point.actual_cost
  if (metric === 'requests') return point.request_count
  if (metric === 'tokens') return point.input_tokens + point.output_tokens
  return point.active_members
}

export const totalValue = (summary: TeamUsageSummary, metric: TeamCardMetric): number => {
  if (metric === 'cost') return summary.actual_cost
  if (metric === 'requests') return summary.request_count
  if (metric === 'tokens') return summary.input_tokens + summary.output_tokens
  return summary.active_members
}

export const modelValue = (model: TeamUsageModel, metric: TeamMetric): number => {
  if (metric === 'cost') return model.actual_cost
  if (metric === 'requests') return model.request_count
  return model.input_tokens + model.output_tokens
}

export const formatMetric = (value: number, metric: TeamCardMetric): string => {
  if (metric === 'cost') return formatTeamCost(value)
  if (metric === 'tokens') return formatTokensK(value)
  return formatNumber(Math.round(value))
}

// 纵轴刻度用短格式：金额最多两位小数且去掉末尾的 0，避免出现 $0.0000。
export const formatAxis = (value: number, metric: TeamMetric): string => {
  if (metric === 'cost') return `$${Number(value.toFixed(2))}`
  if (metric === 'tokens') return formatTokensK(value)
  return formatNumber(value)
}

// from 到 to 的每一天（含两端），接口只返回有用量的日期，按这些日期补零。
export const daysBetween = (from: string, to: string): string[] => {
  const days: string[] = []
  const cursor = new Date(`${from}T00:00:00`)
  const end = new Date(`${to}T00:00:00`)
  while (cursor <= end) {
    days.push(formatDateLocalInput(cursor))
    cursor.setDate(cursor.getDate() + 1)
  }
  return days
}

// 紧邻当前范围之前、天数相同的范围，用于环比和对比线。
export const previousRangeOf = (from: string, to: string): { from: string; to: string } => {
  const length = daysBetween(from, to).length
  const end = new Date(`${from}T00:00:00`)
  end.setDate(end.getDate() - 1)
  const start = new Date(end)
  start.setDate(start.getDate() - (length - 1))
  return { from: formatDateLocalInput(start), to: formatDateLocalInput(end) }
}

// 把每日数据按日期补齐成与 days 对齐的数组。
export const seriesOf = (summary: TeamUsageSummary | null, days: string[], metric: TeamCardMetric): number[] => {
  const byDate = new Map((summary?.daily ?? []).map((point) => [point.date, dailyValue(point, metric)]))
  return days.map((date) => byDate.get(date) ?? 0)
}

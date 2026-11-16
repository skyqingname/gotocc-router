import { i18n } from '@/i18n'
import { formatDateLocalInput, formatDateTime, formatRelativeTime } from '@/utils/format'

// 任务中心的日期与时长换算。日期按用户本地时区的自然日处理：开始日 0 点为 since，结束日次日 0 点为不含的 until。

function localDayStart(date: string, offsetDays: number): number {
  const [year, month, day] = date.split('-').map(Number)
  return Math.floor(new Date(year, month - 1, day + offsetDays).getTime() / 1000)
}

// 清空的日期表示该侧不设界限，对应接口参数 0。
export function dayRangeToUnix(start: string, end: string): { since: number; until: number } {
  return { since: start ? localDayStart(start, 0) : 0, until: end ? localDayStart(end, 1) : 0 }
}

// 默认范围：含今天在内的最近 days 天。
export function defaultDayRange(days: number): { start: string; end: string } {
  const today = new Date()
  const start = new Date(today.getFullYear(), today.getMonth(), today.getDate() - (days - 1))
  return { start: formatDateLocalInput(start), end: formatDateLocalInput(today) }
}

export function relativeTime(unixSeconds: number): string {
  return formatRelativeTime(new Date(unixSeconds * 1000))
}

export function absoluteTime(unixSeconds: number): string {
  return formatDateTime(new Date(unixSeconds * 1000))
}

// 耗时 = 完成时间 - 创建时间，按秒、分、时三档显示。
export function formatElapsed(fromUnix: number, toUnix: number): string {
  const total = toUnix - fromUnix
  const t = i18n.global.t
  if (total < 60) return t('gotocc.taskCenter.duration.seconds', { s: total })
  if (total < 3600) return t('gotocc.taskCenter.duration.minutes', { m: Math.floor(total / 60), s: total % 60 })
  return t('gotocc.taskCenter.duration.hours', { h: Math.floor(total / 3600), m: Math.floor((total % 3600) / 60) })
}

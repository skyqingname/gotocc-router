// 团队页的格式化：金额按美元，相对时间随界面语言。

export const formatTeamCost = (value: number): string =>
  `$${value.toLocaleString('en-US', { minimumFractionDigits: value >= 1 ? 2 : 4, maximumFractionDigits: value >= 1 ? 2 : 4 })}`

const RELATIVE_STEPS: Array<[Intl.RelativeTimeFormatUnit, number]> = [
  ['year', 365 * 24 * 3600],
  ['month', 30 * 24 * 3600],
  ['day', 24 * 3600],
  ['hour', 3600],
  ['minute', 60],
]

// 例如「3 小时前」；不足一分钟显示「现在」。
export const formatRelativeTime = (iso: string, locale: string): string => {
  const seconds = (new Date(iso).getTime() - Date.now()) / 1000
  const formatter = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
  for (const [unit, size] of RELATIVE_STEPS) {
    if (Math.abs(seconds) >= size) return formatter.format(Math.round(seconds / size), unit)
  }
  return formatter.format(0, 'second')
}

// 姓名首两个字符，作为没有头像时的占位。
export const initialsOf = (name: string): string => name.substring(0, 2).toUpperCase()

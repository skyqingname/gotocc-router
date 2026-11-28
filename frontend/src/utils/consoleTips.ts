import consoleGuide from '../../../console-guide.json'
import { withCreditUnit } from '@/utils/credits'
import { formatRelativeWithDateTime } from '@/utils/format'

// 悬停提示里的数值写法：金额给到 amount_digits 位并去掉末尾的 0，时间给「相对时间 · 精确时间」。

export function exactCredit(value: number): string {
  return withCreditUnit(value.toFixed(consoleGuide.tooltip.amount_digits).replace(/\.?0+$/, ''))
}

export function timeTip(value: string | Date): string {
  return formatRelativeWithDateTime(value)
}

// 仪表盘金额统一按美元显示；小额消费保留 4 位小数，避免显示成 0.00。
export const formatUsd = (value: number, fractionDigits: number): string =>
  `$${value.toLocaleString('en-US', { minimumFractionDigits: fractionDigits, maximumFractionDigits: fractionDigits })}`

export const formatCostAuto = (value: number): string => formatUsd(value, value >= 1 ? 2 : 4)

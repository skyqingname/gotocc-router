import config from '../../../balance-cashback.json'

export interface BalanceCashbackSettings {
  enabled: boolean
  minimum_amount: number
  percent: number
}

export function validBalanceCashbackSettings(value: BalanceCashbackSettings): boolean {
  return [value.minimum_amount, value.percent].every(number => Number.isFinite(number) && number >= 0)
}

export function formatCashbackAmount(amount: number, locale: string): string {
  return amount.toLocaleString(locale, {
    minimumFractionDigits: 2,
    maximumFractionDigits: config.amount_decimal_places,
  })
}

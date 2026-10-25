import { i18n } from '@/i18n'

export const creditUnit = (): string => i18n.global.t('common.creditUnit')

// Keep the caller's precision; changing the unit never changes the amount.
export const withCreditUnit = (amount: string | number): string => `${amount} ${creditUnit()}`

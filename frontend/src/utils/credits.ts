import billingCredits from '../../../billing-credits.json'

export const creditUnit = (): string => billingCredits.display_symbol

// Keep the caller's precision; changing the symbol never changes the amount.
export const withCreditUnit = (amount: string | number): string => `${creditUnit()}${amount}`

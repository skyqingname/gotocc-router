export default {
  balanceCashback: {
    title: 'Balance cashback',
    description: 'Main-site users receive extra credit when a balance code redemption or an administrator deposit exceeds the per-credit threshold.',
    enabled: 'Enable balance cashback',
    minimumAmount: 'Single credit greater than ($)',
    minimumHint: 'The original credit must be strictly greater than this amount. Equal amounts do not qualify.',
    percent: 'Cashback rate (%)',
    percentHint: 'Calculated from the original credit and recorded as a separate entry.',
    invalidSettings: 'Enter a non-negative threshold and cashback rate.',
    historyTitle: 'Balance cashback',
    congratulations: 'Congratulations!',
    notice: 'You received an extra ${amount} in credit, recorded as a separate entry.',
  },
}

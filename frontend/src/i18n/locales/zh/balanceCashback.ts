export default {
  balanceCashback: {
    title: '余额返现',
    description: '主站用户兑换余额码或由平台管理员加额时，满足单笔门槛可获额外额度。',
    enabled: '启用余额返现',
    minimumAmount: '单笔额度大于（积分）',
    minimumHint: '严格大于此额度才参与，等于门槛不返现。',
    percent: '返现比例（%）',
    percentHint: '按本笔原始额度计算，返现单独入账。',
    invalidSettings: '请填写不小于 0 的参与门槛和返现比例。',
    historyTitle: '余额返现',
    congratulations: '恭喜您！',
    notice: '本次额外增加了 {amount} 积分 额度，已单独入账。',
  },
}

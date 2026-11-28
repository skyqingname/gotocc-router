import type { RouteLocation, RouteLocationRaw } from 'vue-router'

export type WalletTab = 'recharge' | 'subscription' | 'orders' | 'redeem'

/**
 * 原充值/订阅、我的订阅、我的订单、兑换四页并入 /wallet 后的重定向：查询参数和锚点原样带上。
 * 旧 /purchase 带 tab=subscription，或是订阅订单的支付回跳（order_type=subscription、plan_id）进订阅选项卡，其余进充值。
 */
export function walletRedirect(to: RouteLocation, from: 'purchase' | Exclude<WalletTab, 'recharge'>): RouteLocationRaw {
  const subscription = to.query.tab === 'subscription' || to.query.order_type === 'subscription' || Boolean(to.query.plan_id)
  const tab: WalletTab = from === 'purchase' ? (subscription ? 'subscription' : 'recharge') : from
  return { path: '/wallet', query: { ...to.query, tab }, hash: to.hash }
}

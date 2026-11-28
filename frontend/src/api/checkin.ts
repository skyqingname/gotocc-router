/** 签到中心：某月的签到日历，以及签今天。金额都是余额单位。 */
import { apiClient } from './client'

export interface CheckinDay {
  date: string
  reward: number
  milestone_days: number
  milestone_reward: number
}

export interface CheckinMilestone {
  days: number
  reward: number
  reached: boolean
  remaining: number
  /** false：本月剩下的天数已经不够签满。 */
  reachable: boolean
}

export interface CheckinOverview {
  timezone: string
  today: string
  current_month: string
  month: string
  /** 本自然月是否有过充值；false 时 lock_reason 为 recharge_required。 */
  unlocked: boolean
  lock_reason?: 'recharge_required'
  checked_in_today: boolean
  daily_reward: number
  /** 今天签到能拿到的金额（含今天触发的里程碑）；已签时为 0。 */
  today_reward: number
  today_milestone_days: number
  days: CheckinDay[]
  month_days: number
  month_reward: number
  total_reward: number
  milestones: CheckinMilestone[]
}

export interface CheckinResult {
  date: string
  reward: number
  milestone_days: number
  milestone_reward: number
  total: number
  month_days: number
  balance: number
}

export async function getOverview(month?: string): Promise<CheckinOverview> {
  const { data } = await apiClient.get<CheckinOverview>('/checkin', { params: month ? { month } : undefined })
  return data
}

export async function checkIn(): Promise<CheckinResult> {
  const { data } = await apiClient.post<CheckinResult>('/checkin')
  return data
}

export const checkinAPI = { getOverview, checkIn }

export default checkinAPI

/** 用户端服务状态：按分组的被动检测（只看真实请求），以及分组可用模型。 */
import { apiClient } from './client'
import type { GroupPlatform } from '@/types'

export type GroupHealth = 'normal' | 'degraded' | 'outage' | 'idle'

export interface StatusCounts {
  total: number
  success: number
}

export interface StatusBucket extends StatusCounts {
  start: string
  end: string
  status: GroupHealth
}

export interface GroupStatus {
  id: number
  name: string
  description: string
  platform: GroupPlatform
  subscription_type: string
  is_exclusive: boolean
  /** 当前用户实际生效的倍率；站长客户是站长给他的客户价。 */
  rate_multiplier: number
  status: GroupHealth
  recent: StatusCounts
  day: StatusCounts
  ttft_p50_ms: number | null
  cache_hit_rate: number | null
  buckets: StatusBucket[]
}

export interface ServiceStatusSnapshot {
  generated_at: string
  timezone: string
  recent_requests: number
  bucket_minutes: number
  groups: GroupStatus[]
}

export interface GroupModel {
  name: string
  platform: GroupPlatform
}

export async function getSnapshot(): Promise<ServiceStatusSnapshot> {
  const { data } = await apiClient.get<ServiceStatusSnapshot>('/service-status')
  return data
}

export async function getGroupModels(groupId: number): Promise<GroupModel[]> {
  const { data } = await apiClient.get<GroupModel[]>(`/service-status/groups/${groupId}/models`)
  return data
}

export const serviceStatusAPI = { getSnapshot, getGroupModels }

export default serviceStatusAPI

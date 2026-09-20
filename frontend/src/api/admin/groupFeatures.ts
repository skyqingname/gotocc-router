import { apiClient } from '../client'
import type { RateScheduleConfig } from '@/utils/rate-schedule'

export interface GroupRateScheduleView {
  group_id: number
  version: number
  config: RateScheduleConfig
  server_timezone: string
  source: 'legacy' | 'schedule'
  updated_at: string
}

export async function getGroupRateSchedule(groupID: number): Promise<GroupRateScheduleView> {
  const { data } = await apiClient.get<GroupRateScheduleView>(`/admin/group-features/${groupID}/rate-schedule`)
  return data
}

export async function saveGroupRateSchedule(
  groupID: number,
  expectedVersion: number,
  config: RateScheduleConfig
): Promise<GroupRateScheduleView> {
  const { data } = await apiClient.put<GroupRateScheduleView>(`/admin/group-features/${groupID}/rate-schedule`, {
    expected_version: expectedVersion,
    config
  })
  return data
}

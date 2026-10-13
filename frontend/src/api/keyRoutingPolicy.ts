import { apiClient } from './client'
import type { RoutingPriorityGroup } from './groups'

export interface RoutingPreference {
  default_group_order: number[]
  model_rules: Array<{ model: string; group_ids: number[] }>
}
export interface KeyRoutingPreference {
  allow_user_override: boolean
  preference: RoutingPreference | null
  available_groups: RoutingPriorityGroup[]
}
export async function getKeyRoutingPreference(id: number, signal?: AbortSignal): Promise<KeyRoutingPreference> {
  const { data } = await apiClient.get<KeyRoutingPreference>(`/keys/${id}/routing-policy`, { signal })
  return data
}
export async function updateKeyRoutingPreference(id: number, preference: RoutingPreference | null): Promise<void> {
  await apiClient.put(`/keys/${id}/routing-policy`, preference, { headers: { 'Content-Type': 'application/json' }, transformRequest: [(data) => JSON.stringify(data)] })
}

/**
 * User Groups API endpoints (non-admin)
 * Handles group-related operations for regular users
 */

import { apiClient } from './client'
import type { Group } from '@/types'

export interface RoutingPriorityGroup {
  id: number
  name: string
  platform: string
}

export interface RoutingPriorities {
  default_source: 'administrator' | 'group_sort'
  groups: RoutingPriorityGroup[]
  model_rules: Array<{
    model: string
    matched_rule: string
    groups: RoutingPriorityGroup[]
  }>
}

export async function getRoutingPriorities(
  scope: 'personal' | 'team' = 'personal',
  signal?: AbortSignal,
): Promise<RoutingPriorities> {
  const { data } = await apiClient.get<RoutingPriorities>('/groups/routing-priorities', {
    params: { scope }, signal,
  })
  return data
}

/**
 * Get available groups that the current user can bind to API keys
 * This returns groups based on user's permissions:
 * - Standard groups: public (non-exclusive) or explicitly allowed
 * - Subscription groups: user has active subscription
 * @returns List of available groups
 */
export async function getAvailable(scope: 'personal' | 'team' = 'personal'): Promise<Group[]> {
  const { data } = await apiClient.get<Group[]>('/groups/available', { params: { scope } })
  return data
}

/**
 * Get current user's custom group rate multipliers
 * @returns Map of group_id to custom rate_multiplier
 */
export async function getUserGroupRates(scope: 'personal' | 'team' = 'personal'): Promise<Record<number, number>> {
  const { data } = await apiClient.get<Record<number, number> | null>('/groups/rates', { params: { scope } })
  return data || {}
}

export const userGroupsAPI = {
  getAvailable,
  getRoutingPriorities,
  getUserGroupRates
}

export default userGroupsAPI

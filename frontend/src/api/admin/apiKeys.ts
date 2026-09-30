/**
 * Admin API Keys API endpoints
 * Handles API key management for administrators
 */

import { apiClient } from '../client'
import type { ApiKeyRoutingCapabilities, ApiKeyRoutingMode } from '@/types'
import type { AdminSupportAPIKey } from './supportView'

export interface AutoGroupRoutingPolicy {
  default_group_order: number[]
  model_rules: Array<{ model: string; group_ids: number[] }>
}

export async function getAutoRoutingPolicy(): Promise<AutoGroupRoutingPolicy> {
  const { data } = await apiClient.get<AutoGroupRoutingPolicy>('/admin/api-keys/routing-policy')
  return data
}

export async function updateAutoRoutingPolicy(policy: AutoGroupRoutingPolicy): Promise<void> {
  await apiClient.put('/admin/api-keys/routing-policy', policy)
}

export interface UpdateApiKeyGroupResult {
  api_key: AdminSupportAPIKey
  auto_granted_group_access: boolean
  granted_group_id?: number
  granted_group_name?: string
}

/**
 * Update an API key's group binding
 * @param id - API Key ID
 * @param groupId - Group ID (0 to unbind, positive to bind, null/undefined to skip)
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKeyGroup(id: number, groupId: number | null): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, {
    group_id: groupId === null ? 0 : groupId,
  })
  return data
}

export async function updateApiKeyRouting(
  id: number,
  routingMode: ApiKeyRoutingMode,
  groupId: number | null = null,
): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, {
    routing_mode: routingMode,
    group_id: routingMode === 'auto' ? null : groupId,
  })
  return data
}

export async function getApiKeyRoutingCapabilities(id: number): Promise<ApiKeyRoutingCapabilities> {
  const { data } = await apiClient.get<ApiKeyRoutingCapabilities>(`/admin/api-keys/${id}/routing-capabilities`)
  return data
}

export const apiKeysAPI = {
  updateApiKeyGroup,
  updateApiKeyRouting,
  getApiKeyRoutingCapabilities,
}

export default apiKeysAPI

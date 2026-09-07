import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'

export interface ReusableInvitationCode {
  id: number
  code: string
  status: 'active' | 'disabled'
  max_uses: number
  used_count: number
  expires_at?: string | null
  notes: string
  created_at: string
  updated_at: string
}

export interface ReusableInvitationCodeUse {
  id: number
  code_id: number
  user_id: number
  email: string
  auth_source: string
  used_at: string
}

export interface CreateReusableInvitationCodeRequest {
  code: string
  max_uses?: number
  expires_at?: string | null
  notes?: string
}

export async function list(
  page = 1,
  pageSize = 20,
  filters?: { sort_by?: string; sort_order?: 'asc' | 'desc' },
  options?: { signal?: AbortSignal }
): Promise<BasePaginationResponse<ReusableInvitationCode>> {
  const { data } = await apiClient.get<BasePaginationResponse<ReusableInvitationCode>>(
    '/admin/reusable-invitation-codes',
    {
      params: { page, page_size: pageSize, ...filters },
      signal: options?.signal
    }
  )
  return data
}

export async function create(
  request: CreateReusableInvitationCodeRequest
): Promise<ReusableInvitationCode> {
  const { data } = await apiClient.post<ReusableInvitationCode>(
    '/admin/reusable-invitation-codes',
    request
  )
  return data
}

export async function disable(id: number): Promise<ReusableInvitationCode> {
  const { data } = await apiClient.post<ReusableInvitationCode>(
    `/admin/reusable-invitation-codes/${id}/disable`
  )
  return data
}

export async function listUses(id: number, limit = 50): Promise<ReusableInvitationCodeUse[]> {
  const { data } = await apiClient.get<ReusableInvitationCodeUse[]>(
    `/admin/reusable-invitation-codes/${id}/uses`,
    { params: { limit } }
  )
  return data
}

export default { list, create, disable, listUses }

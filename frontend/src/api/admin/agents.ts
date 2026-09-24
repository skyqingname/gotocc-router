/**
 * LC-024 admin agent API: the application review queue.
 *
 * Nothing about an application is user-supplied beyond the submission itself, so
 * the payload is only the identity of the applicant and the timing.
 */
import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export type AgentStatus = 'pending' | 'approved' | 'rejected'
export type AgentSource = 'applied' | 'grandfathered'

export interface AgentApplication {
  user_id: number
  username: string
  email: string
  /** Account registration time, used to tell new users from grandfathered ones. */
  created_at: string
  status: AgentStatus
  source: AgentSource
  applied_at: string | null
  reviewed_at: string | null
}

export interface ListAgentApplicationsParams {
  page?: number
  page_size?: number
  search?: string
  status?: AgentStatus | ''
}

export async function listApplications(
  params: ListAgentApplicationsParams = {},
): Promise<PaginatedResponse<AgentApplication>> {
  const { data } = await apiClient.get<PaginatedResponse<AgentApplication>>('/admin/agents', {
    params: {
      page: params.page ?? 1,
      page_size: params.page_size ?? 20,
      search: params.search ?? '',
      status: params.status || undefined,
    },
  })
  return data
}

export async function reviewApplication(
  userId: number,
  approve: boolean,
): Promise<{ user_id: number; status: AgentStatus }> {
  const { data } = await apiClient.post<{ user_id: number; status: AgentStatus }>(
    `/admin/agents/${userId}/review`,
    { approve },
  )
  return data
}

export const agentsAPI = { listApplications, reviewApplication }

export default agentsAPI

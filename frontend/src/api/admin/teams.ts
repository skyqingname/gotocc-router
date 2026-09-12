import { apiClient } from '../client'
import type { TeamContext, TeamMembership, TeamStatus, TeamUsageQuery, TeamUsageSummary } from '../team'

export interface AdminTeam {
  id: number
  name: string
  status: TeamStatus
  member_limit: number
  member_count: number
  created_at: string
  updated_at: string
  owner_user_id: number
  owner_email: string
}

const teamsAPI = {
  async list(): Promise<AdminTeam[]> {
    const { data } = await apiClient.get<AdminTeam[]>('/admin/teams')
    return data
  },
  async get(id: number): Promise<TeamContext> {
    const { data } = await apiClient.get<TeamContext>(`/admin/teams/${id}`)
    return data
  },
  async members(id: number): Promise<TeamMembership[]> {
    const { data } = await apiClient.get<TeamMembership[]>(`/admin/teams/${id}/members`)
    return data
  },
  async usage(id: number, query: TeamUsageQuery = {}): Promise<TeamUsageSummary> {
    const { data } = await apiClient.get<TeamUsageSummary>(`/admin/teams/${id}/usage`, { params: query })
    return data
  },
  async create(payload: { owner_user_id: number; name: string; member_limit: number }): Promise<TeamContext> {
    const { data } = await apiClient.post<TeamContext>('/admin/teams', payload)
    return data
  },
  async update(id: number, payload: { name?: string; status?: TeamStatus; member_limit?: number }): Promise<TeamContext> {
    const { data } = await apiClient.patch<TeamContext>(`/admin/teams/${id}`, payload)
    return data
  },
  async forceTransfer(id: number, targetUserID: number): Promise<TeamContext> {
    const { data } = await apiClient.post<TeamContext>(`/admin/teams/${id}/force-transfer`, { target_user_id: targetUserID })
    return data
  },
  async dissolve(id: number): Promise<void> {
    await apiClient.delete(`/admin/teams/${id}`)
  },
}

export default teamsAPI

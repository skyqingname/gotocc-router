import { apiClient } from './client'

export type TeamRole = 'owner' | 'member'
export type TeamStatus = 'active' | 'suspended'

export interface Team {
  id: number
  name: string
  status: TeamStatus
  member_limit: number
  default_daily_limit_usd: number
  default_weekly_limit_usd: number
  default_monthly_limit_usd: number
  member_count: number
  created_at: string
  updated_at: string
}

export interface TeamMembership {
  id: number
  team_id: number
  user_id: number
  email: string
  username: string
  role: TeamRole
  daily_limit_usd: number
  weekly_limit_usd: number
  monthly_limit_usd: number
  daily_usage_usd: number
  weekly_usage_usd: number
  monthly_usage_usd: number
  joined_at: string
  last_active_at: string | null
}

export interface TeamContext {
  team: Team
  membership: TeamMembership
  owner: TeamMembership
}

export interface TeamInvitation {
  id: number
  team_id: number
  inviter_user_id: number
  email: string
  status: string
  expires_at: string
  accepted_at: string | null
  created_at: string
}

export interface TeamInvitationPreview {
  team_name: string
  inviter_name: string
  inviter_email: string
  expires_at: string
}

export interface TeamAPIKey {
  id: number
  user_id: number
  user_email: string
  name: string
  masked_key: string
  status: string
  team_owner_disabled: boolean
  group_id: number | null
  group_name: string
  last_used_at: string | null
  created_at: string
}

export interface TeamUsageDaily {
  date: string
  actual_cost: number
  request_count: number
}

export interface TeamUsageSummary {
  actual_cost: number
  request_count: number
  input_tokens: number
  output_tokens: number
  daily: TeamUsageDaily[]
}

export interface TeamMemberUsageSeries {
  actor_user_id: number
  display_name: string
  status: 'active' | 'left'
  summary: TeamUsageSummary
}

export interface TeamUsageLog {
  id: number
  actor_user_id: number
  actor_email: string
  api_key_id: number
  api_key_name: string
  request_id: string
  model: string
  actual_cost: number
  input_tokens: number
  output_tokens: number
  created_at: string
}

export interface TeamUsagePage {
  items: TeamUsageLog[]
  total: number
  limit: number
  offset: number
}

export interface TeamUsageQuery {
  from?: string
  to?: string
  member_id?: number
  api_key_id?: number
  limit?: number
  offset?: number
}

export const teamAPI = {
  async current(): Promise<TeamContext> {
    const { data } = await apiClient.get<TeamContext>('/team')
    return data
  },
  async create(name: string): Promise<TeamContext> {
    const { data } = await apiClient.post<TeamContext>('/team', { name })
    return data
  },
  async rename(name: string): Promise<TeamContext> {
    const { data } = await apiClient.patch<TeamContext>('/team', { name })
    return data
  },
  async updateDefaultMemberLimits(limits: {
    default_daily_limit_usd: number
    default_weekly_limit_usd: number
    default_monthly_limit_usd: number
  }): Promise<TeamContext> {
    const { data } = await apiClient.patch<TeamContext>('/team/default-member-limits', limits)
    return data
  },
  async setStatus(status: TeamStatus): Promise<TeamContext> {
    const { data } = await apiClient.post<TeamContext>('/team/status', { status })
    return data
  },
  async dissolve(): Promise<void> {
    await apiClient.delete('/team')
  },
  async leave(): Promise<void> {
    await apiClient.post('/team/leave')
  },
  async members(): Promise<TeamMembership[]> {
    const { data } = await apiClient.get<TeamMembership[]>('/team/members')
    return data
  },
  async removeMember(userID: number): Promise<void> {
    await apiClient.delete(`/team/members/${userID}`)
  },
  async updateLimits(userID: number, limits: {
    daily_limit_usd: number
    weekly_limit_usd: number
    monthly_limit_usd: number
  }): Promise<void> {
    await apiClient.patch(`/team/members/${userID}/limits`, limits)
  },
  async resetUsage(userID: number, periods: { daily: boolean; weekly: boolean; monthly: boolean }): Promise<void> {
    await apiClient.post(`/team/members/${userID}/usage/reset`, periods)
  },
  async invitations(): Promise<TeamInvitation[]> {
    const { data } = await apiClient.get<TeamInvitation[]>('/team/invitations')
    return data
  },
  async invite(email: string): Promise<TeamInvitation> {
    const { data } = await apiClient.post<TeamInvitation>('/team/invitations', { email })
    return data
  },
  async previewInvitation(token: string): Promise<TeamInvitationPreview> {
    const { data } = await apiClient.post<TeamInvitationPreview>('/team/invitations/preview', { token })
    return data
  },
  async resolveInvitation(token: string, resolution: 'accepted' | 'declined'): Promise<TeamContext | null> {
    const { data } = await apiClient.post<TeamContext | null>('/team/invitations/resolve', { token, resolution })
    return data
  },
  async reissueInvitation(id: number): Promise<TeamInvitation> {
    const { data } = await apiClient.post<TeamInvitation>(`/team/invitations/${id}/reissue`)
    return data
  },
  async revokeInvitation(id: number): Promise<void> {
    await apiClient.delete(`/team/invitations/${id}`)
  },
  async startTransfer(targetUserID: number): Promise<void> {
    await apiClient.post('/team/ownership-transfer', { target_user_id: targetUserID })
  },
  async resolveTransfer(token: string, resolution: 'accepted' | 'declined'): Promise<TeamContext | null> {
    const { data } = await apiClient.post<TeamContext | null>('/team/ownership-transfer/resolve', { token, resolution })
    return data
  },
  async usage(query: TeamUsageQuery = {}): Promise<TeamUsageSummary> {
    const { data } = await apiClient.get<TeamUsageSummary>('/team/usage', { params: query })
    return data
  },
  async memberUsage(query: Pick<TeamUsageQuery, 'from' | 'to'> = {}): Promise<TeamMemberUsageSeries[]> {
    const { data } = await apiClient.get<TeamMemberUsageSeries[]>('/team/usage/members', { params: query })
    return data
  },
  async usageLogs(query: TeamUsageQuery = {}): Promise<TeamUsagePage> {
    const { data } = await apiClient.get<TeamUsagePage>('/team/usage/logs', { params: query })
    return data
  },
  async keys(): Promise<TeamAPIKey[]> {
    const { data } = await apiClient.get<TeamAPIKey[]>('/team/keys')
    return data
  },
  async disableKey(id: number): Promise<void> {
    await apiClient.post(`/team/keys/${id}/disable`)
  },
  async enableKey(id: number): Promise<void> {
    await apiClient.post(`/team/keys/${id}/enable`)
  },
  async deleteKey(id: number): Promise<void> {
    await apiClient.delete(`/team/keys/${id}`)
  },
}

export default teamAPI

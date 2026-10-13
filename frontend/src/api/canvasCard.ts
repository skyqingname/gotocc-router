import { apiClient } from './client'
import type { Team } from './team'

export type CanvasCardFunding = 'personal' | 'team'
export interface CanvasCardStock { denomination: number; available: number; issued: number }
export interface CanvasCardOverview { stock: CanvasCardStock[]; personal_balance: number; team: Team | null; team_enabled: boolean }
export interface CanvasCardRequest { operation_id: string; denomination: number; funding_source: CanvasCardFunding; team_id: number | null }
export interface CanvasCardOrder extends CanvasCardRequest { id: number; user_id: number; balance_after: number; code: string; created_at: string }
export interface CanvasCardItem { id: number; denomination: number; code: string; created_at: string; user_id: number | null; funding_source: CanvasCardFunding | null; team_id: number | null; issued_at: string | null }
export interface CanvasCardPage<T> { items: T[]; total: number; page: number; page_size: number; pages: number }

export const canvasCardAPI = {
  async overview() { return (await apiClient.get<CanvasCardOverview>('/redeem/canvas-cards')).data },
  async exchange(input: CanvasCardRequest) { return (await apiClient.post<CanvasCardOrder>('/redeem/canvas-cards', input)).data },
  async stock() { return (await apiClient.get<CanvasCardStock[]>('/admin/canvas-cards/stock')).data },
  async inventory(page: number, denomination: number, status: string) { return (await apiClient.get<CanvasCardPage<CanvasCardItem>>('/admin/canvas-cards', { params: { page, denomination, status } })).data },
  async import(denomination: number, codes: string[]) { return (await apiClient.post<{ imported: number }>('/admin/canvas-cards/import', { denomination, codes })).data },
}

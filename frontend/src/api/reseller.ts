import { apiClient } from './client'
export interface ResellerProfile { user_id: number; enabled: boolean; invitation_code: string; default_multiplier: number }
export interface ResellerCustomer { user_id: number; username: string; email: string; status: string; notes: string; created_at: string; charged: number; profit: number; last_usage_at: string | null }
export interface ResellerPrice { customer_id: number | null; group_id: number | null; multiplier: number | null }
export interface ResellerGroup { id: number; name: string; platform: string; base_multiplier: number; image_multiplier: number; image_independent: boolean; video_multiplier: number; video_independent: boolean }
export interface ResellerPrices { prices: ResellerPrice[]; groups: ResellerGroup[]; default_multiplier: number }
export interface ResellerEarning { id: number; customer_id: number; username: string; group_id: number; group_name: string; model: string; charged: number; cost: number; profit: number; multiplier: number; created_at: string }
export interface ResellerPage<T> { items: T[]; total: number; page: number; page_size: number }
// rebate is the invite-commission total this reseller received as a beneficiary.
// It is unrelated to reseller pricing and survives the removal of per-reseller rates.
export interface ResellerOverview { profile: ResellerProfile; summary: { customer_count: number; charged: number; profit: number; rebate: number } }
export const resellerAPI = {
  access: async () => (await apiClient.get<{enabled: boolean}>('/reseller/access')).data,
  overview: async () => (await apiClient.get<ResellerOverview>('/reseller')).data,
  customers: async (page = 1, search = '') => (await apiClient.get<ResellerPage<ResellerCustomer>>('/reseller/customers', { params: { page, search } })).data,
  notes: async (id: number, notes: string) => (await apiClient.put(`/reseller/customers/${id}/notes`, { notes })).data,
  prices: async () => (await apiClient.get<ResellerPrices>('/reseller/prices')).data,
  savePrices: async (customer_id: number | null, multiplier: number | null, groups: ResellerPrice[]) => (await apiClient.put('/reseller/prices', { customer_id, multiplier, groups })).data,
  earnings: async (page = 1) => (await apiClient.get<ResellerPage<ResellerEarning>>('/reseller/earnings', { params: { page } })).data,
  adminGet: async (id: number) => (await apiClient.get<ResellerProfile>(`/admin/users/${id}/reseller`)).data,
  adminSave: async (id: number, enabled: boolean) => (await apiClient.put<ResellerProfile>(`/admin/users/${id}/reseller`, { enabled })).data
}

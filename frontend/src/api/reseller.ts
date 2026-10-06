import { apiClient } from './client'
import type { User, ApiKey, UsageLog, ResellerCustomerAccount, Announcement } from '@/types'
export interface ResellerCommunicationSettings { contact_enabled: boolean; contact_info: string; announcements_enabled: boolean; sync_main_announcements: boolean }
export interface ResellerProfile extends ResellerCommunicationSettings { user_id: number; enabled: boolean; invitation_code: string; default_multiplier: number; initial_credit: number }
export interface ResellerAnnouncement extends Announcement { source_announcement_id: number | null }
export interface ResellerAnnouncementReview extends Announcement { review_status: 'pending' | 'approved' | 'rejected'; reviewed_at: string | null }
export interface ResellerAnnouncementInput { title: string; content: string; status: 'draft' | 'active' | 'archived'; notify_mode: 'silent' | 'popup' }
export interface ResellerCustomer { user_id: number; username: string; email: string; status: string; notes: string; created_at: string; charged: number; profit: number; last_usage_at: string | null; credit_balance: number; frozen_credit: number; cost: number }
export interface ResellerPrice { customer_id: number | null; group_id: number | null; multiplier: number | null }
export interface ResellerGroup { id: number; name: string; platform: string; base_multiplier: number; image_multiplier: number; image_independent: boolean; video_multiplier: number; video_independent: boolean }
export interface ResellerPrices { prices: ResellerPrice[]; groups: ResellerGroup[]; default_multiplier: number }
export interface ResellerEarning { id: number; customer_id: number; username: string; group_id: number; group_name: string; model: string; charged: number; cost: number; profit: number; multiplier: number; created_at: string; settlement_type: string }
export interface ResellerPage<T> { items: T[]; total: number; page: number; page_size: number }
export interface ResellerOverview { profile: ResellerProfile; invitation_url: string; limits: ResellerOwnerLimits; summary: { customer_count: number; charged: number; profit: number; cost: number; credit_balance: number; gifted: number } }
export interface ResellerOwnerLimits { balance: number; concurrency: number; rpm_limit: number }
export interface ResellerCustomerInput { email: string; username: string; password: string; status: 'active' | 'disabled'; concurrency: number; rpm_limit: number; allowed_groups: number[]; notes: string }
export interface ResellerCreditEntry { id: number; customer_id: number; owner_id: number; operation_id: string; kind: string; amount: number; frozen_amount: number; balance_after: number; frozen_after: number; platform_cost: number; model: string; notes: string; created_at: string }
export interface ResellerCreditInput { operation_id: string; kind: 'increase' | 'deduct'; amount: number; deduct_all: boolean; notes: string }
export interface ResellerKeyInput { name: string; group_id: number | null; routing_mode: 'fixed' | 'auto'; quota: number }
export interface MyResellerCredits extends ResellerPage<ResellerCreditEntry> { account: ResellerCustomerAccount }
export const resellerAPI = {
  saveCommunicationSettings: async (input: ResellerCommunicationSettings) => (await apiClient.put<ResellerProfile>('/reseller/communication-settings', input)).data,
  announcements: async () => (await apiClient.get<ResellerAnnouncement[]>('/reseller/announcements')).data,
  createAnnouncement: async (input: ResellerAnnouncementInput) => (await apiClient.post<ResellerAnnouncement>('/reseller/announcements', input)).data,
  updateAnnouncement: async (id: number, input: ResellerAnnouncementInput) => (await apiClient.put<ResellerAnnouncement>(`/reseller/announcements/${id}`, input)).data,
  setAnnouncementStatus: async (id: number, status: ResellerAnnouncementInput['status']) => (await apiClient.put(`/reseller/announcements/${id}/status`, { status })).data,
  mainAnnouncements: async () => (await apiClient.get<ResellerAnnouncementReview[]>('/reseller/main-announcements')).data,
  reviewMainAnnouncement: async (id: number, source_updated_at: string, status: 'approved' | 'rejected') => (await apiClient.post(`/reseller/main-announcements/${id}/review`, { source_updated_at, status })).data,
  saveInitialCredit: async (initial_credit: number) => (await apiClient.put<ResellerProfile>('/reseller/initial-credit', { initial_credit })).data,
  customer: async (id: number) => (await apiClient.get<User>(`/reseller/customers/${id}`)).data,
  createCustomer: async (input: ResellerCustomerInput) => (await apiClient.post<User>('/reseller/customers', input)).data,
  updateCustomer: async (id: number, input: ResellerCustomerInput) => (await apiClient.put<User>(`/reseller/customers/${id}`, input)).data,
  changeCredit: async (id: number, input: ResellerCreditInput) => (await apiClient.post<ResellerCreditEntry>(`/reseller/customers/${id}/credit`, input)).data,
  creditEntries: async (id: number, page = 1) => (await apiClient.get<ResellerPage<ResellerCreditEntry>>(`/reseller/customers/${id}/credit-entries`, { params: { page } })).data,
  myCredits: async (page = 1) => (await apiClient.get<MyResellerCredits>('/reseller/credits', { params: { page } })).data,
  customerKeys: async (id: number, page = 1) => (await apiClient.get<ResellerPage<ApiKey>>(`/reseller/customers/${id}/keys`, { params: { page } })).data,
  createCustomerKey: async (id: number, input: ResellerKeyInput) => (await apiClient.post<ApiKey>(`/reseller/customers/${id}/keys`, input)).data,
  updateCustomerKey: async (id: number, keyID: number, input: { status?: string; quota?: number }) => (await apiClient.put<ApiKey>(`/reseller/customers/${id}/keys/${keyID}`, input)).data,
  deleteCustomerKey: async (id: number, keyID: number) => (await apiClient.delete(`/reseller/customers/${id}/keys/${keyID}`)).data,
  customerUsage: async (id: number, page = 1) => (await apiClient.get<ResellerPage<UsageLog>>(`/reseller/customers/${id}/usage`, { params: { page } })).data,
  access: async () => (await apiClient.get<{enabled: boolean; customer: ResellerCustomerAccount | null}>('/reseller/access')).data,
  overview: async () => (await apiClient.get<ResellerOverview>('/reseller')).data,
  customers: async (page = 1, search = '') => (await apiClient.get<ResellerPage<ResellerCustomer>>('/reseller/customers', { params: { page, search } })).data,
  notes: async (id: number, notes: string) => (await apiClient.put(`/reseller/customers/${id}/notes`, { notes })).data,
  prices: async () => (await apiClient.get<ResellerPrices>('/reseller/prices')).data,
  savePrices: async (customer_id: number | null, multiplier: number | null, groups: ResellerPrice[]) => (await apiClient.put('/reseller/prices', { customer_id, multiplier, groups })).data,
  earnings: async (page = 1) => (await apiClient.get<ResellerPage<ResellerEarning>>('/reseller/earnings', { params: { page } })).data,
  adminGet: async (id: number) => (await apiClient.get<ResellerProfile>(`/admin/users/${id}/reseller`)).data,
  adminSave: async (id: number, enabled: boolean) => (await apiClient.put<ResellerProfile>(`/admin/users/${id}/reseller`, { enabled })).data
}

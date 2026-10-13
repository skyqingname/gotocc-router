/**
 * Redeem code API endpoints
 * Handles redeem code redemption for users
 */

import { apiClient } from './client'
import type { CanvasCardOrder } from './canvasCard'
import type { PaginatedResponse, RedeemCodeRequest } from '@/types'

export interface RedeemHistoryItem {
  canvas_card?: CanvasCardOrder
  cashback_amount?: number
  id: number
  code: string
  type: string
  value: number
  status: string
  used_at: string
  created_at: string
  // Notes from admin for admin_balance/admin_concurrency types
  notes?: string
  // Subscription-specific fields
  group_id?: number
  validity_days?: number
  group?: {
    id: number
    name: string
  }
}

/**
 * Redeem a code
 * @param code - Redeem code string
 * @returns The redeemed code record (type, value, validity days and group)
 */
export async function redeem(code: string): Promise<RedeemHistoryItem> {
  const payload: RedeemCodeRequest = { code }

  const { data } = await apiClient.post<RedeemHistoryItem>('/redeem', payload)

  return data
}

/**
 * Get user's redemption history
 * @returns The requested page of redeemed codes and the total count
 */
export async function getHistory(page = 1, pageSize = 20): Promise<PaginatedResponse<RedeemHistoryItem>> {
  const { data } = await apiClient.get<PaginatedResponse<RedeemHistoryItem>>('/redeem/history', {
    params: { page, page_size: pageSize }
  })
  return data
}

export async function getActivity(page = 1, pageSize = 20): Promise<PaginatedResponse<RedeemHistoryItem>> {
  return (await apiClient.get<PaginatedResponse<RedeemHistoryItem>>('/redeem/activity', { params: { page, page_size: pageSize } })).data
}

export const redeemAPI = {
  getActivity,
  redeem,
  getHistory
}

export default redeemAPI

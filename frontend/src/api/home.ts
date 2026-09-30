import { apiClient } from './client'
import { getModelPlaza, type ModelPlazaResponse } from './modelPlaza'

export interface HomepageStats {
  today_tokens: number
  total_tokens: number
  total_users: number
}

export async function getHomepageStats(): Promise<HomepageStats> {
  const { data } = await apiClient.get<HomepageStats>('/marketplace/stats')
  return data
}

export async function getHomepageModelPlaza(): Promise<ModelPlazaResponse> {
  return getModelPlaza()
}

export const homeAPI = { getHomepageStats, getHomepageModelPlaza }

export default homeAPI

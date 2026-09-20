import { apiClient } from '../client'

export interface VideoParameter {
  type: 'integer' | 'string' | 'boolean'
  required: boolean
  locked: boolean
  default?: string | number | boolean
  minimum?: number
  maximum?: number
  enum?: (string | number | boolean)[]
}
export interface VideoPrice {
  unit: 'per_job' | 'per_second'
  usd: string
  seconds?: number
  resolution?: string
  size?: string
  generate_audio?: boolean
}
export interface VideoBinding {
  id: string
  name: string
  enabled: boolean
  account_id: number
  public_model: string
  upstream_model: string
  protocol: 'openai_video_json' | 'xai_video'
  parameters: Record<string, VideoParameter>
  prices: VideoPrice[]
}
export interface VideoConfigView {
  group_id: number
  version: number
  config: { bindings: VideoBinding[] }
  updated_at: string
  execution_ready: boolean
}
export interface VideoQuote {
  quote: {
    config_hash: string
    binding_id: string
    account_id: number
    public_model: string
    upstream_model: string
    protocol: string
    parameters: Record<string, string | number | boolean>
    base_cost_usd: string
  }
  submitted: false
  charged: false
}
export async function getVideoConfig(groupID: number): Promise<VideoConfigView> {
  const { data } = await apiClient.get<VideoConfigView>(`/admin/group-features/${groupID}/video-config`)
  return data
}
export async function saveVideoConfig(groupID: number, expectedVersion: number, bindings: VideoBinding[]): Promise<VideoConfigView> {
  const { data } = await apiClient.put<VideoConfigView>(`/admin/group-features/${groupID}/video-config`, {
    expected_version: expectedVersion, config: { bindings }
  })
  return data
}
export async function previewVideoBinding(groupID: number, bindingID: string): Promise<VideoQuote> {
  const { data } = await apiClient.post<VideoQuote>(`/admin/group-features/${groupID}/video-preview`, {
    binding_id: bindingID, parameters: {}
  })
  return data
}

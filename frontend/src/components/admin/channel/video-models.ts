import defaults from '../../../../../video-platform-defaults.json'

export interface VideoParameter {
  name: string
  type: string
  required: boolean
  values: string[]
  min?: number
  max?: number
}

export interface VideoModelConfig {
  enabled: boolean
  protocol: 'openai' | 'custom_json'
  upstream_model: string
  create_status: string
  create_path: string
  status_path: string
  content_path: string
  headers: Record<string, string>
  defaults: Record<string, unknown>
  parameters: VideoParameter[]
  request_fields: Record<string, string>
  id_field: string
  status_field: string
  video_url_field: string
  statuses: Record<string, string>
}

export function newVideoModel(model: string): VideoModelConfig {
  return { ...JSON.parse(JSON.stringify(defaults)), upstream_model: model }
}

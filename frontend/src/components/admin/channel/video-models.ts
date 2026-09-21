import defaults from '../../../../../video-platform-defaults.json'

export interface VideoParameter {
  name: string
  label?: string
  disabled?: boolean
  type: string
  required: boolean
  values: string[]
  min?: number
  max?: number
  step?: number
}

export interface VideoModelConfig {
  enabled: boolean
  protocol: 'openai' | 'custom_json' | 'yingce'
  provider_definition?: Record<string, unknown>
  provider_id?: string
  provider_options?: Record<string, Record<string, unknown>>
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

export interface VideoProtocolEntry {
 id: string; name: string; vendor: string; version: string; documentation: string; template: VideoModelConfig
 configuration: { fields: {name: string; type: string; label: string; required?: boolean}[] }
 definition: { create: {method:string;path:string;contentType?:string;body?:unknown};poll?:{method:string;path:string};result?:{method:string;path:string};auth:{type:string};parameters:{name:string;type:string;description?:string;mapping?:string;values?:string[]}[] }
 workflows: {id:string;label:string;defaults?:Record<string,unknown>;parameters?:unknown[]}[]
}

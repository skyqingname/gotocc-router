import { apiClient } from './client'

// 任务中心（GoToCC）：登录用户自己的视频任务与图片任务统计。图片任务列表、筛选项和下载沿用 api/imageTasks.ts。

export type TaskCenterStatus = 'processing' | 'completed' | 'failed'

// since/until 为 Unix 秒，until 不含；model 为空时不筛选。
export interface TaskCenterRange {
  since: number
  until: number
  model: string
}

export interface VideoTask {
  // 上游受理前为 null
  id: string | null
  model: string
  status: TaskCenterStatus
  seconds: number
  resolution: string
  billing_mode: 'per_request' | 'video'
  // 已完成为实付，结算完成前为 null；其余为 0
  cost: number | null
  created_at: number
  finished_at: number | null
  error: { code: string | null; message: string | null } | null
  content_available: boolean
}

export interface VideoTaskListResponse {
  object: 'list'
  data: VideoTask[]
  has_more: boolean
}

export interface TaskSummary {
  total: number
  completed: number
  processing: number
  failed: number
  cost: number
}

// 图片任务无法与用量记录逐条精确对应时 cost_available 为 false，费用以使用记录为准。
export interface ImageTaskSummary extends TaskSummary {
  cost_available: boolean
}

function rangeParams(range: TaskCenterRange): Record<string, string | number> {
  const params: Record<string, string | number> = { since: range.since, until: range.until }
  if (range.model) params.model = range.model
  return params
}

export async function listVideoTasks(range: TaskCenterRange, limit: number, offset: number): Promise<VideoTaskListResponse> {
  const { data } = await apiClient.get<VideoTaskListResponse>('/video-tasks', { params: { ...rangeParams(range), limit, offset } })
  return data
}

export async function getVideoTaskModels(): Promise<string[]> {
  const { data } = await apiClient.get<{ models: string[] }>('/video-tasks/filters')
  return data.models
}

export async function getVideoTaskSummary(range: TaskCenterRange): Promise<TaskSummary> {
  const { data } = await apiClient.get<TaskSummary>('/video-tasks/summary', { params: rangeParams(range) })
  return data
}

export async function getImageTaskSummary(range: TaskCenterRange): Promise<ImageTaskSummary> {
  const { data } = await apiClient.get<ImageTaskSummary>('/image-tasks/summary', { params: rangeParams(range) })
  return data
}

// 视频内容带登录态取成 Blob，由调用方生成对象地址并在用完后释放。视频体积大，不套用默认的 30 秒超时。
export async function fetchVideoTaskContent(taskId: string): Promise<Blob> {
  const { data } = await apiClient.get<Blob>(`/video-tasks/${encodeURIComponent(taskId)}/content`, { responseType: 'blob', timeout: 0 })
  return data
}

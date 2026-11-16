import { i18n } from '@/i18n'
import { fetchVideoTaskContent } from '@/api/taskCenter'
import { saveAsyncImageBlob } from '@/api/asyncImage'

// 内容接口按 Blob 读取，出错时响应体也是 Blob，因此按状态码给出说明：404 不存在或不属于本人，409 未完成，其余为上游取不到。
export function videoContentErrorMessage(error: unknown): string {
  const status = (error as { status?: number }).status
  const t = i18n.global.t
  if (status === 404) return t('gotocc.taskCenter.video.contentErrors.notFound')
  if (status === 409) return t('gotocc.taskCenter.video.contentErrors.notCompleted')
  return t('gotocc.taskCenter.video.contentErrors.unavailable')
}

// 下载存为 mp4，对象地址在保存后释放。
export async function downloadVideoTask(taskId: string): Promise<void> {
  const blob = await fetchVideoTaskContent(taskId)
  saveAsyncImageBlob(blob, `${taskId}.mp4`)
}

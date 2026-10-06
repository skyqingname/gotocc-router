import type { VideoCapabilities } from '@/components/admin/channel/video-models'

// 视频接口文档与模型广场共用：能力说明、按能力生成的统一 /v1/videos 示例。

export type VideoExampleKind = 'text' | 'image' | 'image-base64' | 'video' | 'audio' | 'all'

export const VIDEO_EXAMPLE_KINDS: { key: VideoExampleKind; label: string }[] = [
  { key: 'text', label: '文生视频' },
  { key: 'image', label: '图生视频' },
  { key: 'image-base64', label: '图生视频（base64）' },
  { key: 'video', label: '参考视频' },
  { key: 'audio', label: '参考音频' },
  { key: 'all', label: '全能参考' },
]

// 尚未填写能力的模型不做限制，示例全部可用。
const unrestricted: VideoCapabilities = { reference_images: true, reference_videos: true, reference_audios: true, audio_output: false }
const capsOf = (caps?: VideoCapabilities | null) => caps ?? unrestricted

// 能力条目，例如「参考图最多 9 张」「不支持参考视频」「时长：5–15 秒」。
export function videoCapabilityItems(caps?: VideoCapabilities | null): string[] {
  if (!caps) return ['参考素材与时长以模型说明为准']
  const items: string[] = []
  if (caps.reference_images) items.push(caps.max_reference_images ? `参考图最多 ${caps.max_reference_images} 张` : '支持参考图')
  if (caps.reference_videos) items.push(caps.max_reference_videos ? `参考视频最多 ${caps.max_reference_videos} 个` : '支持参考视频')
  if (caps.reference_audios) items.push(caps.max_reference_audios ? `参考音频最多 ${caps.max_reference_audios} 段` : '支持参考音频')
  if (caps.max_reference_total) items.push(`参考素材合计最多 ${caps.max_reference_total} 个`)
  if (!caps.reference_images) items.push('不支持参考图')
  if (!caps.reference_videos) items.push('不支持参考视频')
  if (!caps.reference_audios) items.push('不支持参考音频')
  if (caps.audio_output) items.push('可生成音频：传 "generate_audio": true')
  if (caps.resolutions?.length) items.push(`分辨率：${caps.resolutions.join(' / ')}`)
  if (caps.aspect_ratios?.length) items.push(`画面比例：${caps.aspect_ratios.join(' / ')}`)
  if (caps.fixed_seconds?.length) items.push(`固定时长：${caps.fixed_seconds.join(' / ')} 秒`)
  else if (caps.min_seconds || caps.max_seconds) items.push(`时长：${caps.min_seconds || 1}–${caps.max_seconds || '不限'} 秒`)
  return items
}

// 卡片底部的简短规格，例如「4–15 秒 · 720p / 1080p」。
export function videoSpecLine(caps?: VideoCapabilities | null): string {
  if (!caps) return ''
  const parts: string[] = []
  if (caps.fixed_seconds?.length) parts.push(`${caps.fixed_seconds.join(' / ')} 秒`)
  else if (caps.min_seconds || caps.max_seconds) parts.push(`${caps.min_seconds || 1}–${caps.max_seconds || '∞'} 秒`)
  if (caps.resolutions?.length) parts.push(caps.resolutions.join(' / '))
  return parts.join(' · ')
}

export function videoExampleSupported(caps: VideoCapabilities | null | undefined, kind: VideoExampleKind): boolean {
  const c = capsOf(caps)
  return {
    text: true,
    image: c.reference_images,
    'image-base64': c.reference_images,
    video: c.reference_videos,
    audio: c.reference_audios,
    all: c.reference_images && c.reference_videos && c.reference_audios,
  }[kind]
}

// 与网关一致：固定时长取第一档，否则 5 秒落在范围内。
function exampleSeconds(caps: VideoCapabilities): number {
  if (caps.fixed_seconds?.length) return caps.fixed_seconds[0]
  return Math.min(Math.max(5, caps.min_seconds || 0), caps.max_seconds || Infinity)
}

// 64x64 占位 PNG，示例可直接运行；实际使用替换为自己的图片。
const PLACEHOLDER_IMAGE = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAIAAAAlC+aJAAAATklEQVR42u3PQQkAAAgEsOsq2N8GRvAtDFZgqZ7XIiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICBwWV7OoZaC+o1lAAAAAElFTkSuQmCC'

export interface VideoExample {
  payload: Record<string, unknown>
  curl: string
  javascript: string
  python: string
}

export function buildVideoExample(model: string, caps: VideoCapabilities | null | undefined, kind: VideoExampleKind, baseURL: string): VideoExample {
  const c = capsOf(caps)
  const payload: Record<string, unknown> = {
    model,
    prompt: kind === 'text' ? '海边日落，镜头缓慢推进，电影级画质' : '保持参考主体外观一致，自然运动',
    duration: exampleSeconds(c),
  }
  if (c.resolutions?.length) payload.resolution = c.resolutions[0]
  payload.aspect_ratio = c.aspect_ratios?.[0] ?? '16:9'
  if (kind === 'image' || kind === 'all') payload.reference_images = ['https://example.com/reference-image.jpg']
  if (kind === 'image-base64') payload.reference_images = [PLACEHOLDER_IMAGE]
  if (kind === 'video' || kind === 'all') payload.reference_videos = ['https://example.com/reference-video.mp4']
  if (kind === 'audio' || kind === 'all') payload.reference_audios = ['https://example.com/reference-audio.mp3']
  if (c.audio_output) payload.generate_audio = true

  const url = `${baseURL}/videos`
  const pretty = JSON.stringify(payload, null, 2)
  let curl = `curl ${url} \\\n  -H "Authorization: Bearer sk-你的API密钥" \\\n  -H "Content-Type: application/json" \\\n  -d '${JSON.stringify(payload)}'`
  let javascript = `const response = await fetch('${url}', {\n  method: 'POST',\n  headers: { Authorization: 'Bearer sk-你的API密钥', 'Content-Type': 'application/json' },\n  body: JSON.stringify(${pretty})\n})\nconst task = await response.json()\nconsole.log(task.id)`
  let python = `import requests\n\nresponse = requests.post('${url}', headers={\n    'Authorization': 'Bearer sk-你的API密钥',\n    'Content-Type': 'application/json',\n}, json=${pretty})\nprint(response.json()['id'])`
  if (kind === 'image-base64') {
    const body = JSON.stringify({ ...payload, reference_images: ['<reference.jpg 的 data URL>'] }, null, 2)
    curl = `# 参考图以 base64 data URL 直传（示例是一张 64x64 占位图，替换成你自己的即可）\n${curl}`
    javascript = `import { readFileSync } from 'node:fs'\n\nconst dataURL = 'data:image/jpeg;base64,' + readFileSync('reference.jpg').toString('base64')\nconst payload = ${body}\npayload.reference_images = [dataURL]\n\nconst response = await fetch('${url}', {\n  method: 'POST',\n  headers: { Authorization: 'Bearer sk-你的API密钥', 'Content-Type': 'application/json' },\n  body: JSON.stringify(payload)\n})`
    python = `import base64, requests\n\nwith open('reference.jpg', 'rb') as file:\n    data_url = 'data:image/jpeg;base64,' + base64.b64encode(file.read()).decode()\n\npayload = ${body}\npayload['reference_images'] = [data_url]\n\nresponse = requests.post('${url}', headers={\n    'Authorization': 'Bearer sk-你的API密钥',\n    'Content-Type': 'application/json',\n}, json=payload)`
  }
  return { payload, curl, javascript, python }
}

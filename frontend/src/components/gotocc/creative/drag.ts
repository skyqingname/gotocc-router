// Derived from TokenFlux/TokenRouter f383062dd5fd534f462887acb70ea241ce0d8278. Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation.
// 创作台拖放协议：历史输出通过自定义 MIME 携带本地素材索引，图片本体仍留在 IndexedDB。
export const CREATIVE_OUTPUT_DRAG_MIME = 'application/x-gotocc-creative-output'

export interface CreativeOutputDragPayload {
  runId: string
  outputIndex: number
}

const CREATIVE_OUTPUT_RUN_ID_RE = /^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$/
const SUPPORTED_IMAGE_MIMES = new Set(['image/png', 'image/jpeg', 'image/webp'])
const SUPPORTED_IMAGE_EXTENSIONS = new Set(['png', 'jpg', 'jpeg', 'webp'])

// 校验外部文件类型；部分操作系统拖放不会填写 MIME，因此空 MIME 时回退文件扩展名。
export function isSupportedCreativeImageFile(file: File): boolean {
  const mime = file.type.trim().toLowerCase()
  if (mime) return SUPPORTED_IMAGE_MIMES.has(mime)
  const extension = file.name.split('.').pop()?.trim().toLowerCase() ?? ''
  return SUPPORTED_IMAGE_EXTENSIONS.has(extension)
}

// 序列化历史输出拖放载荷，调用方传入的索引必须已经过校验。
export function serializeCreativeOutputDrag(payload: CreativeOutputDragPayload): string {
  return JSON.stringify(payload)
}

// 解析并校验历史输出拖放载荷，任何不可信或不完整的数据都按无效处理。
export function parseCreativeOutputDrag(value: string | null | undefined): CreativeOutputDragPayload | null {
  if (!value) return null
  try {
    const parsed: unknown = JSON.parse(value)
    if (!parsed || typeof parsed !== 'object') return null
    const candidate = parsed as { runId?: unknown; outputIndex?: unknown }
    if (typeof candidate.runId !== 'string') return null
    const runId = candidate.runId.trim()
    if (!CREATIVE_OUTPUT_RUN_ID_RE.test(runId)) return null
    if (typeof candidate.outputIndex !== 'number' || !Number.isInteger(candidate.outputIndex) || candidate.outputIndex < 0) {
      return null
    }
    return { runId, outputIndex: candidate.outputIndex }
  } catch {
    return null
  }
}

import { buildApiUrl } from '@/api/client'
import parameters from './parameters.gen.json'

const nonTextModel = new RegExp(parameters.non_text_model_pattern.replace('(?i)', ''), 'i')
export const isFingerprintTextModel = (id: string) => !nonTextModel.test(id)
export const supportsFingerprintPlatform = (platform: string) => !parameters.unsupported_platforms.includes(platform)

interface ProbeEvent { type: string; text?: string; model?: string; success?: boolean; error?: string }

export async function sampleAccountFingerprint(
  accountID: number,
  modelID: string,
  prompt: string,
  signal: AbortSignal,
  onText: (text: string) => void,
  onModel: (model: string) => void
): Promise<string> {
  const response = await fetch(buildApiUrl(`/admin/accounts/${accountID}/model-fingerprint`), {
    method: 'POST',
    headers: { Authorization: `Bearer ${localStorage.getItem('auth_token')}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ model_id: modelID, prompt }),
    signal
  })
  if (!response.ok) throw new Error(`HTTP ${response.status}: ${await response.text()}`)
  if (!response.body) throw new Error('The account returned no response stream')
  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = '', text = '', completed = false
  const consume = (line: string) => {
    if (!line.startsWith('data:')) return
    const raw = line.slice(5).trim()
    if (!raw) return
    const event = JSON.parse(raw) as ProbeEvent
    if (event.type === 'error') throw new Error(event.error || 'Account request failed')
    if (event.type === 'test_start' && event.model) onModel(event.model)
    if (event.type === 'content' && event.text) {
      text += event.text
      onText(text)
    }
    if (event.type === 'test_complete') {
      if (!event.success) throw new Error(event.error || 'Account request failed')
      completed = true
    }
  }
  try {
    while (true) {
      const { done, value } = await reader.read()
      buffer += done ? decoder.decode() : decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop()!
      for (const line of lines) consume(line.trimEnd())
      if (done) {
        if (buffer) consume(buffer.trimEnd())
        break
      }
    }
    if (!completed) throw new Error('The response stream ended before completion')
    return text
  } finally {
    await reader.cancel()
    reader.releaseLock()
  }
}

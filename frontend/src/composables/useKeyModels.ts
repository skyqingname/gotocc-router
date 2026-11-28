import { ref } from 'vue'
import { buildGatewayUrl } from '@/api/client'

// Models one API key can call: the site's own /v1/models with that key, the same
// list clients see. Loading state drives the "fetching / failed + retry / none /
// available + refresh" line in key dialogs.

export type KeyModelsState = 'idle' | 'loading' | 'ready' | 'error'

export function useKeyModels() {
  const models = ref<string[]>([])
  const state = ref<KeyModelsState>('idle')
  let request = 0

  async function load(apiKey: string) {
    const current = ++request
    state.value = 'loading'
    try {
      const response = await fetch(buildGatewayUrl('/v1/models'), {
        headers: { Authorization: `Bearer ${apiKey}` },
        credentials: 'omit',
        cache: 'no-store'
      })
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      const body = (await response.json()) as { data?: Array<{ id?: string }> }
      if (current !== request) return
      models.value = [...new Set((body.data ?? []).map(entry => entry.id?.trim() ?? '').filter(Boolean))].sort()
      state.value = 'ready'
    } catch {
      if (current === request) state.value = 'error'
    }
  }

  function reset() {
    request++
    models.value = []
    state.value = 'idle'
  }

  return { models, state, load, reset }
}

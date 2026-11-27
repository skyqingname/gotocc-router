import { computed, reactive, ref, type Ref } from 'vue'
import type { PublicSettings } from '@/types'
import keyConsole from '../../../key-console.json'

// The request entry a user copies into clients. Latency depends on the user's own
// network, so the choice lives in this browser and is shared by every view that
// shows or copies a Base URL.

export interface ApiEndpointOption {
  url: string
  name: string
  description: string
  probeUrl: string
  primary: boolean
}

export type EndpointProbe =
  | { state: 'probing' }
  | { state: 'ok'; ms: number }
  | { state: 'failed' }

const probeConfig = keyConsole.endpoint_probe
const selectedUrl = ref(localStorage.getItem(probeConfig.storage_key) ?? '')
const probes = reactive<Record<string, EndpointProbe>>({})
const probing = ref(false)

async function timeProbe(url: string): Promise<number> {
  const init: RequestInit = { mode: 'no-cors', cache: 'no-store', credentials: 'omit' }
  // The first request pays DNS, TCP and TLS setup; the second measures the round trip.
  await fetch(url, { ...init, signal: AbortSignal.timeout(probeConfig.timeout_ms) })
  const started = performance.now()
  await fetch(url, { ...init, signal: AbortSignal.timeout(probeConfig.timeout_ms) })
  return Math.round(performance.now() - started)
}

export function useApiEndpoint(settings: Ref<PublicSettings | null>) {
  const endpoints = computed<ApiEndpointOption[]>(() => {
    const primaryUrl = settings.value?.api_base_url || window.location.origin
    const primary: ApiEndpointOption = {
      url: primaryUrl,
      name: '',
      description: '',
      probeUrl: new URL(probeConfig.primary_probe_path, primaryUrl).toString(),
      primary: true
    }
    return [
      primary,
      ...(settings.value?.custom_endpoints ?? []).map(endpoint => ({
        url: endpoint.endpoint,
        name: endpoint.name,
        description: endpoint.description,
        probeUrl: endpoint.endpoint,
        primary: false
      }))
    ]
  })

  const current = computed(() => endpoints.value.find(endpoint => endpoint.url === selectedUrl.value) ?? endpoints.value[0])
  const baseUrl = computed(() => current.value.url)

  function select(url: string) {
    selectedUrl.value = url
    localStorage.setItem(probeConfig.storage_key, url)
  }

  /** Probes every entry in parallel and selects the fastest; returns it, or null when all fail. */
  async function probeAll(): Promise<{ endpoint: ApiEndpointOption; ms: number } | null> {
    probing.value = true
    const list = endpoints.value
    for (const endpoint of list) probes[endpoint.url] = { state: 'probing' }
    const results = await Promise.allSettled(list.map(endpoint => timeProbe(endpoint.probeUrl)))
    let best: { endpoint: ApiEndpointOption; ms: number } | null = null
    for (const [index, result] of results.entries()) {
      const endpoint = list[index]
      if (result.status === 'fulfilled') {
        probes[endpoint.url] = { state: 'ok', ms: result.value }
        if (!best || result.value < best.ms) best = { endpoint, ms: result.value }
      } else {
        probes[endpoint.url] = { state: 'failed' }
      }
    }
    probing.value = false
    if (best) select(best.endpoint.url)
    return best
  }

  return { endpoints, current, baseUrl, select, probes, probing, probeAll }
}

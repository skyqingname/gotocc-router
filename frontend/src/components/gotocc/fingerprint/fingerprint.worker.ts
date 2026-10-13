import detectorURL from './vendor/detector.json.gz?url'
import bank from './vendor/bank.json'
import { analyzeSharedOutputs, type SharedDetector } from './vendor/shared-detector'
import type { Output } from './vendor/types'

let detector: SharedDetector

self.onmessage = async (event: MessageEvent<{ type: 'load' | 'analyze'; outputs?: Output[] }>) => {
  try {
    if (event.data.type === 'load') {
      const response = await fetch(detectorURL)
      if (!response.ok || !response.body) throw new Error(`Reference data: HTTP ${response.status}`)
      const decoded = response.body.pipeThrough(new DecompressionStream('gzip'))
      detector = await new Response(decoded).json() as SharedDetector
      self.postMessage({ type: 'ready' })
      return
    }
    const analysis = analyzeSharedOutputs(event.data.outputs!, bank, detector, { allowPartial: true })
    self.postMessage({ type: 'result', analysis })
  } catch (error) {
    self.postMessage({ type: 'error', message: error instanceof Error ? error.message : String(error) })
  }
}

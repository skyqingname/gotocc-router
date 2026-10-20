// Adapted type-only surface from Ikaleio/lm-detector (MIT; see LICENSE.lm-detector).
export interface Challenge { id: string; expected_count: number; prompt: string }
export interface Output { text: string; expected_count: number }
export interface Bank {
  schema: string
  built_at: string
  models: { id: string; display_name: string; family: string; family_name: string; response_count: number }[]
}
export interface Analysis {
  prediction: string
  prediction_name: string
  probability: number | null
  used_outputs: number
  results: { model: string; display_name: string; family: string; family_name: string; score: number; probability: number | null }[]
  diagnostics: { index: number; parsed_numbers: number; minimum_numbers: number; accepted: boolean }[]
  evidence: { insufficient: boolean; label: string; reason: string; threshold: number | null; method: string }
  decision?: string
  sequence?: { autocorrelation: number; small_step_share: number; model_like: boolean } | null
  [key: string]: unknown
}

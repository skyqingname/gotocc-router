// 模型广场的厂商与用途展示名。厂商 ID 来自 models.dev 或后台填写；未收录的厂商按原值展示。
export const VENDOR_LABELS: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  google: 'Google',
  gemini: 'Google',
  xai: 'xAI',
  grok: 'xAI',
  deepseek: 'DeepSeek',
  moonshotai: 'Moonshot',
  zhipuai: '智谱',
  zai: '智谱',
  minimax: 'MiniMax',
  alibaba: '阿里云',
  volcengine: '火山引擎',
  bytedance: '字节跳动',
  mistral: 'Mistral',
  meta: 'Meta',
  cohere: 'Cohere',
  antigravity: 'Antigravity',
}

// 同一厂商的不同 ID（如 google 与 gemini 平台）归到同一个筛选项。
export const VENDOR_CANONICAL: Record<string, string> = { gemini: 'google', grok: 'xai', zai: 'zhipuai' }

export const canonicalVendor = (vendor: string): string => VENDOR_CANONICAL[vendor] ?? vendor

export const vendorLabel = (vendor: string): string => VENDOR_LABELS[vendor] ?? vendor

export const PURPOSES = ['language', 'image', 'video', 'audio'] as const
export type Purpose = (typeof PURPOSES)[number]

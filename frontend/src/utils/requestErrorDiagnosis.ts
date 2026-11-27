// 失败请求的归因判断，管理端运维错误与用户端「错误」标签页共用。
// 只回答「是谁的问题、属于哪一类」；各端按自己的读者拼文案：
// 管理端写明上游账号、分组等内部细节（views/admin/ops/utils/opsErrorDiagnosis.ts），
// 用户端只给白话原因和下一步建议（utils/userErrorAdvice.ts）。

export type ErrorBlame = 'user' | 'upstream' | 'gateway'

export type ErrorKind =
  | 'client_closed'
  | 'model_unsupported'
  | 'upstream_no_model'
  | 'upstream_auth'
  | 'upstream_rate_limit'
  | 'upstream_stream_broken'
  | 'upstream_timeout'
  | 'upstream_bad_request'
  | 'upstream_other'
  | 'ciphertext'
  | 'access_quota'
  | 'invalid_request'
  | 'session_route'
  | 'pool_exhausted'
  | 'gateway_internal'

export interface ErrorSignals {
  /** 返回给客户端的状态码（网关状态码优先）。 */
  primaryStatus: number
  gatewayStatus: number | null
  upstreamStatus: number | null
  phase: string
  type: string
  owner: string
  /** 去掉 JSON 包装后的错误消息。 */
  message: string
  upstreamMessage: string
  businessLimited: boolean
  routingCapacityLimited: boolean
  /** 选号时全部候选都只因「不支持该模型」被筛掉。 */
  unsupportedModelOnly: boolean
}

export interface ErrorClassification {
  blame: ErrorBlame
  kind: ErrorKind
}

/** 去掉 `{"error":{"message":…}}` 之类的包装，只留可读的消息。 */
export function extractCleanText(raw?: string | null): string {
  const trimmed = String(raw || '').trim()
  if (!trimmed || trimmed === '{}' || trimmed === '[]' || trimmed.toLowerCase() === 'null') return ''
  if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
    try {
      const parsed = JSON.parse(trimmed)
      if (parsed?.error?.message) return String(parsed.error.message).trim()
      if (parsed?.response?.error?.message) return String(parsed.response.error.message).trim()
      if (parsed?.message) return String(parsed.message).trim()
      if (parsed?.detail) return String(parsed.detail).trim()
    } catch {
      // keep raw snippet
    }
  }
  return trimmed
}

export function looksLikeUpstreamStreamBreak(signals: ErrorSignals): boolean {
  const msgLower = `${signals.message} ${signals.upstreamMessage}`.toLowerCase()
  return (
    msgLower.includes('upstream stream') ||
    msgLower.includes('upstream http/2') ||
    msgLower.includes('stream interrupted') ||
    msgLower.includes('unexpected eof')
  )
}

export function classifyRequestError(signals: ErrorSignals): ErrorClassification {
  const { primaryStatus, gatewayStatus, upstreamStatus, phase, type, owner } = signals
  const msgLower = `${signals.message} ${signals.upstreamMessage}`.toLowerCase()

  if (primaryStatus === 499 || gatewayStatus === 499 || type === 'client_disconnect') {
    return { blame: 'user', kind: 'client_closed' }
  }

  if (
    type === 'model_not_found' ||
    type === 'auto_route_model_not_found' ||
    msgLower.includes('is not available for this key and endpoint') ||
    msgLower.includes('is not supported by any configured account') ||
    signals.unsupportedModelOnly
  ) {
    return { blame: 'user', kind: 'model_unsupported' }
  }

  const streamBreak = looksLikeUpstreamStreamBreak(signals)
  if (
    owner === 'provider' ||
    phase === 'upstream' ||
    phase === 'account_auth' ||
    (upstreamStatus !== null && upstreamStatus >= 400 && owner !== 'client') ||
    streamBreak
  ) {
    if (upstreamStatus === 404 || msgLower.includes('model_not_found') || (msgLower.includes('model') && msgLower.includes('not supported'))) {
      return { blame: 'upstream', kind: 'upstream_no_model' }
    }
    if (upstreamStatus === 401 || upstreamStatus === 403 || phase === 'account_auth' || msgLower.includes('forbidden') || msgLower.includes('cloudflare')) {
      return { blame: 'upstream', kind: 'upstream_auth' }
    }
    if (upstreamStatus === 429 || type === 'rate_limit_error' || msgLower.includes('rate limit') || msgLower.includes('quota')) {
      return { blame: 'upstream', kind: 'upstream_rate_limit' }
    }
    if (streamBreak) return { blame: 'upstream', kind: 'upstream_stream_broken' }
    if (primaryStatus === 504 || upstreamStatus === 504 || msgLower.includes('timeout') || msgLower.includes('deadline exceeded')) {
      return { blame: 'upstream', kind: 'upstream_timeout' }
    }
    if (upstreamStatus === 400 || primaryStatus === 400) return { blame: 'upstream', kind: 'upstream_bad_request' }
    return { blame: 'upstream', kind: 'upstream_other' }
  }

  const teamOrAuth =
    phase === 'auth' ||
    type === 'authentication_error' ||
    type === 'permission_error' ||
    type === 'billing_error' ||
    type === 'subscription_error' ||
    msgLower.includes('团队') ||
    msgLower.includes('api key') ||
    msgLower.includes('balance') ||
    msgLower.includes('quota') ||
    msgLower.includes('limit exceeded')
  if (owner === 'client' || phase === 'request' || teamOrAuth || signals.businessLimited) {
    if (msgLower.includes('encrypted') && msgLower.includes('decrypt')) return { blame: 'user', kind: 'ciphertext' }
    if (teamOrAuth || primaryStatus === 401 || primaryStatus === 403 || primaryStatus === 429) {
      return { blame: 'user', kind: 'access_quota' }
    }
    return { blame: 'user', kind: 'invalid_request' }
  }

  if (msgLower.includes('session routing is temporarily unavailable') || msgLower.includes('takeover')) {
    return { blame: 'gateway', kind: 'session_route' }
  }
  if (signals.routingCapacityLimited || phase === 'routing' || msgLower.includes('no available') || msgLower.includes('temporarily unavailable')) {
    return { blame: 'gateway', kind: 'pool_exhausted' }
  }
  return { blame: 'gateway', kind: 'gateway_internal' }
}

import type { UserErrorRequest, UserErrorRequestDetail } from '@/types'
import { classifyRequestError, extractCleanText, type ErrorBlame, type ErrorKind } from '@/utils/requestErrorDiagnosis'

// 用户端失败请求的白话原因与下一步建议。归因用与管理端相同的判断
// （utils/requestErrorDiagnosis.ts），输入只用用户本来就能看到的字段；
// 文案只说用户能做什么，不出现上游账号、分组内部状态等细节。

/** 用户侧分类码（后端 MapUserErrorCategory）还原为判断所需的阶段与错误类型。 */
const CATEGORY_SIGNALS: Record<string, { phase: string; type: string }> = {
  auth: { phase: 'auth', type: '' },
  service_unavailable: { phase: 'routing', type: '' },
  upstream: { phase: 'upstream', type: '' },
  internal: { phase: 'internal', type: '' },
  security_audit: { phase: 'request', type: 'content_policy_violation' },
  rate_limit: { phase: 'request', type: 'rate_limit_error' },
  quota: { phase: 'request', type: 'billing_error' },
  invalid_request: { phase: 'request', type: 'invalid_request_error' },
  cyber: { phase: 'request', type: 'cyber_policy' },
  other: { phase: '', type: '' }
}

/** 同一类原因在用户端按分类再细分时用的文案键。 */
const REFINED_BY_CATEGORY: Partial<Record<ErrorKind, Record<string, string>>> = {
  access_quota: { quota: 'quota', rate_limit: 'rate_limit', auth: 'auth' },
  invalid_request: { security_audit: 'content_policy', cyber: 'content_policy' }
}

export interface UserErrorAdvice {
  blame: ErrorBlame
  /** 文案键：consoleGuide.errorAdvice.reason.<key> 与 consoleGuide.errorAdvice.next.<key>。 */
  key: string
}

export function adviseUserError(row: UserErrorRequest | UserErrorRequestDetail): UserErrorAdvice {
  const signals = CATEGORY_SIGNALS[row.category]
  const upstreamStatus = 'upstream_status_code' in row && typeof row.upstream_status_code === 'number' && row.upstream_status_code > 0
    ? row.upstream_status_code
    : null
  const { blame, kind } = classifyRequestError({
    primaryStatus: row.status_code,
    gatewayStatus: row.status_code > 0 ? row.status_code : null,
    upstreamStatus,
    phase: signals.phase,
    type: signals.type,
    owner: '',
    message: extractCleanText(row.message),
    upstreamMessage: '',
    businessLimited: false,
    routingCapacityLimited: false,
    unsupportedModelOnly: false
  })
  const refined = REFINED_BY_CATEGORY[kind]?.[row.category]
  return { blame, key: refined ? `${kind}_${refined}` : kind }
}

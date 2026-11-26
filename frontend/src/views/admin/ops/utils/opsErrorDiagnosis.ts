import type { OpsErrorLog, OpsRoutingDiagnostics } from '@/api/admin/ops'

export type OpsBlameKind = 'user' | 'upstream' | 'gateway'

export interface OpsReadableDiagnosis {
  blame: OpsBlameKind
  ownerLabel: string
  subLabel: string
  badgeClass: string
  headline: string
  detail: string
  filterHint: string
  gatewayStatus: number | null
  upstreamStatus: number | null
}

const FILTER_REASON_ZH: Record<string, string> = {
  model_not_supported: '不支持该模型',
  capability_mismatch: '能力不匹配',
  excluded: '本轮已报错排除',
  rate_limited: '限流冷却中',
  model_rate_limited: '模型限流中',
  concurrency_full: '并发槽已满',
  temp_unschedulable: '临时不可调度',
  quota_auto_pause: '额度自动暂停',
  runtime_blocked: '运行态阻断',
  proxy_stream_quarantined: '代理线路熔断',
  channel_upstream_restricted: '渠道模型受限',
  oauth_session_group_denied: 'OAuth 分组受限',
  shadow_parent_unhealthy: '母账号异常'
}

export function formatRoutingDiagnosticsSummary(
  diag: OpsRoutingDiagnostics | undefined | null,
  isZh = true
): string {
  if (!diag) return ''
  const parts: string[] = []
  if (typeof diag.candidate_pool === 'number') {
    parts.push(isZh ? `候选池 ${diag.candidate_pool} 个` : `Pool: ${diag.candidate_pool}`)
  }
  if (diag.filtered_candidates && typeof diag.filtered_candidates === 'object') {
    const entries = Object.entries(diag.filtered_candidates).filter(([, count]) => Number(count) > 0)
    if (entries.length > 0) {
      const items = entries.map(([reason, count]) => {
        const label = isZh ? (FILTER_REASON_ZH[reason] || reason) : reason
        return `${label} x${count}`
      })
      parts.push((isZh ? '筛除原因：' : 'Filtered: ') + items.join(' / '))
    }
  }
  if (diag.timeout_phase) {
    parts.push((isZh ? '超时阶段：' : 'Timeout phase: ') + diag.timeout_phase)
  }
  if (diag.transport_failure) {
    parts.push((isZh ? '传输异常：' : 'Transport: ') + diag.transport_failure)
  }
  return parts.join('；')
}

function extractCleanText(raw?: string | null): string {
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

function formatDurationSec(ms?: number | null): string {
  if (typeof ms !== 'number' || !Number.isFinite(ms) || ms <= 0) return ''
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function formatAccountTag(log: OpsErrorLog, isZh: boolean): string {
  if (log.account_name && log.account_id) return `[${log.account_name} #${log.account_id}]`
  if (log.account_name) return `[${log.account_name}]`
  if (log.account_id) return `[#${log.account_id}]`
  return isZh ? '上游账号' : 'upstream account'
}

function formatGroupTag(log: OpsErrorLog, isZh: boolean): string {
  if (log.group_name) return `[${log.group_name}]`
  if (log.group_id) return `[#${log.group_id}]`
  return isZh ? '当前分组' : 'selected group'
}

function formatModelTag(log: OpsErrorLog): string {
  const req = String(log.requested_model || log.model || '').trim()
  const up = String(log.upstream_model || '').trim()
  if (req && up && req !== up) return `${req} -> ${up}`
  return req || up || 'unknown'
}

function isPureUnsupportedModelFilter(diag?: OpsRoutingDiagnostics | null): boolean {
  if (!diag?.filtered_candidates) return false
  const entries = Object.entries(diag.filtered_candidates).filter(([, c]) => Number(c) > 0)
  if (entries.length === 0) return false
  return entries.every(([k]) => k === 'model_not_supported' || k === 'capability_mismatch' || k === 'channel_upstream_restricted')
}

export function diagnoseOpsError(log: OpsErrorLog, isZh = true): OpsReadableDiagnosis {
  const gwStatus = typeof log.gateway_status_code === 'number' && log.gateway_status_code > 0
    ? log.gateway_status_code
    : (typeof log.status_code === 'number' ? log.status_code : null)
  const upStatus = typeof log.upstream_status_code === 'number' && log.upstream_status_code > 0
    ? log.upstream_status_code
    : null
  const primaryStatus = gwStatus ?? log.status_code ?? 0

  const phase = String(log.phase || '').toLowerCase()
  const errType = String(log.type || '').toLowerCase()
  const ownerRaw = String(log.error_owner || '').toLowerCase()

  const cleanMsg = extractCleanText(log.message)
  const cleanUpMsg = extractCleanText(log.upstream_error_message)
  const msgLower = `${cleanMsg} ${cleanUpMsg}`.toLowerCase()
  const durationText = formatDurationSec(log.duration_ms)
  const filterHint = formatRoutingDiagnosticsSummary(log.routing_diagnostics, isZh)

  const bestRawDetail = (() => {
    const generic = cleanMsg.toLowerCase()
    if (
      cleanUpMsg &&
      (!cleanMsg ||
        generic === 'upstream request failed' ||
        generic === 'service temporarily unavailable' ||
        generic === 'an error occurred while processing your request.')
    ) {
      return cleanUpMsg
    }
    if (cleanMsg && cleanUpMsg && cleanUpMsg !== cleanMsg && !cleanMsg.includes(cleanUpMsg)) {
      return `${cleanMsg} | ${cleanUpMsg}`
    }
    return cleanMsg || cleanUpMsg
  })()

  if (primaryStatus === 499 || gwStatus === 499 || errType === 'client_disconnect') {
    const hasAccount = Boolean(log.account_id || log.account_name)
    const acct = formatAccountTag(log, isZh)
    let headline = ''
    if (cleanMsg.startsWith('用户问题：')) {
      headline = cleanMsg
    } else if (hasAccount) {
      headline = isZh
        ? `用户问题：客户端主动断开连接（已转发至上游 ${acct}${durationText ? `，耗时 ${durationText}` : ''}）`
        : `User issue: client closed connection while waiting on ${acct}${durationText ? ` (${durationText})` : ''}`
    } else {
      headline = isZh
        ? `用户问题：客户端在网关选号或排队阶段主动断开连接${durationText ? `（耗时 ${durationText}）` : ''}`
        : `User issue: client disconnected before upstream dispatch${durationText ? ` (${durationText})` : ''}`
    }
    return {
      blame: 'user',
      ownerLabel: isZh ? '用户问题' : 'User',
      subLabel: isZh ? '客户端断连' : 'Client Closed',
      badgeClass: 'bg-amber-100/[0.18] text-amber-800 dark:bg-amber-900/30 dark:text-amber-200',
      headline,
      detail: bestRawDetail === headline ? '' : bestRawDetail,
      filterHint,
      gatewayStatus: gwStatus || 499,
      upstreamStatus: upStatus
    }
  }

  if (
    errType === 'model_not_found' ||
    errType === 'auto_route_model_not_found' ||
    msgLower.includes('is not available for this key and endpoint') ||
    msgLower.includes('is not supported by any configured account') ||
    isPureUnsupportedModelFilter(log.routing_diagnostics)
  ) {
    const modelName = formatModelTag(log)
    const grp = formatGroupTag(log, isZh)
    const isAuto = msgLower.includes('for this key and endpoint') || !log.group_id
    const headline = isZh
      ? (isAuto
          ? `用户调用问题：该 Key 可用分组均未提供模型 [${modelName}]（端点 ${log.inbound_endpoint || log.request_path || '-'}）`
          : `用户或分组配置问题：分组 ${grp} 内没有支持模型 [${modelName}] 的可用账号`)
      : `User/config issue: model [${modelName}] is not supported in ${grp}`
    return {
      blame: 'user',
      ownerLabel: isZh ? '用户问题' : 'User',
      subLabel: isZh ? '模型未开通' : 'Model Unsupported',
      badgeClass: 'bg-amber-100/[0.18] text-amber-800 dark:bg-amber-900/30 dark:text-amber-200',
      headline,
      detail: bestRawDetail,
      filterHint,
      gatewayStatus: gwStatus,
      upstreamStatus: upStatus
    }
  }

  const looksLikeUpstreamStreamBreak =
    msgLower.includes('upstream stream') ||
    msgLower.includes('upstream http/2') ||
    msgLower.includes('stream interrupted') ||
    msgLower.includes('unexpected eof')
  if (
    ownerRaw === 'provider' ||
    phase === 'upstream' ||
    phase === 'account_auth' ||
    (upStatus !== null && upStatus >= 400 && ownerRaw !== 'client') ||
    looksLikeUpstreamStreamBreak
  ) {
    const acct = formatAccountTag(log, isZh)
    const modelName = formatModelTag(log)
    const codeHint = upStatus ? `HTTP ${upStatus}` : `HTTP ${primaryStatus}`
    let subLabel = isZh ? '上游故障' : 'Upstream Error'
    let headline = ''

    if (upStatus === 404 || msgLower.includes('model_not_found') || (msgLower.includes('model') && msgLower.includes('not supported'))) {
      subLabel = isZh ? '上游缺此模型' : 'Upstream No Model'
      headline = isZh
        ? `上游问题：${acct} 调用不到模型 [${modelName}]（上游返回 ${codeHint}）`
        : `Upstream issue: ${acct} does not serve model [${modelName}] (${codeHint})`
    } else if (upStatus === 401 || upStatus === 403 || phase === 'account_auth' || msgLower.includes('forbidden') || msgLower.includes('cloudflare')) {
      subLabel = isZh ? '上游 401/403 拒访' : 'Upstream Auth/403'
      headline = isZh
        ? `上游问题：${acct} 凭据失效或被上游防火墙拦截（上游返回 ${codeHint}）`
        : `Upstream issue: ${acct} rejected with ${codeHint} (auth or WAF block)`
    } else if (upStatus === 429 || errType === 'rate_limit_error' || msgLower.includes('rate limit') || msgLower.includes('quota')) {
      subLabel = isZh ? '上游限流/无额度' : 'Upstream 429'
      headline = isZh
        ? `上游问题：${acct} 触发上游速率限制或额度耗尽（上游返回 ${codeHint}）`
        : `Upstream issue: ${acct} hit rate/quota limit (${codeHint})`
    } else if (looksLikeUpstreamStreamBreak) {
      subLabel = isZh ? '上游流中途断链' : 'Stream Broken'
      headline = isZh
        ? `上游问题：${acct} 在流式返回中途断开连接${durationText ? `（耗时 ${durationText}）` : ''}`
        : `Upstream issue: ${acct} dropped stream mid-response`
    } else if (primaryStatus === 504 || upStatus === 504 || msgLower.includes('timeout') || msgLower.includes('deadline exceeded')) {
      subLabel = isZh ? '上游响应超时' : 'Upstream Timeout'
      headline = isZh
        ? `上游问题：${acct} 响应超时未按时返回${durationText ? `（等待 ${durationText}）` : ''}`
        : `Upstream issue: ${acct} timed out`
    } else if (upStatus === 400 || primaryStatus === 400) {
      subLabel = isZh ? '上游拒收请求 400' : 'Upstream 400'
      headline = isZh
        ? `上游问题：${acct} 拒绝请求参数或上下文（上游返回 ${codeHint}）`
        : `Upstream issue: ${acct} rejected request (${codeHint})`
    } else {
      headline = isZh
        ? `上游问题：${acct} 服务返回异常（${codeHint}，模型 [${modelName}]）`
        : `Upstream issue: ${acct} failed (${codeHint})`
    }

    return {
      blame: 'upstream',
      ownerLabel: isZh ? '上游问题' : 'Upstream',
      subLabel,
      badgeClass: 'bg-red-100/[0.18] text-red-800 dark:bg-red-900/30 dark:text-red-200',
      headline,
      detail: bestRawDetail,
      filterHint,
      gatewayStatus: gwStatus,
      upstreamStatus: upStatus
    }
  }

  const isTeamOrAuth =
    phase === 'auth' ||
    errType === 'authentication_error' ||
    errType === 'permission_error' ||
    errType === 'billing_error' ||
    errType === 'subscription_error' ||
    msgLower.includes('团队') ||
    msgLower.includes('api key') ||
    msgLower.includes('balance') ||
    msgLower.includes('quota') ||
    msgLower.includes('limit exceeded')
  if (ownerRaw === 'client' || phase === 'request' || isTeamOrAuth || log.is_business_limited) {
    let subLabel = isZh ? '调用参数异常' : 'Invalid Request'
    let headline = ''
    if (msgLower.includes('encrypted') && msgLower.includes('decrypt')) {
      subLabel = isZh ? '历史密文失效' : 'Invalid Ciphertext'
      headline = isZh
        ? `用户会话问题：请求携带的历史加密上下文无法在当前上游解密（跨渠道切换后旧密文残留）`
        : `User session issue: encrypted history item could not be decrypted by current upstream`
    } else if (isTeamOrAuth || primaryStatus === 401 || primaryStatus === 403 || primaryStatus === 429) {
      subLabel = isZh ? '权限/额度受限' : 'Auth / Quota'
      headline = isZh
        ? `用户调用受限：${cleanMsg || 'API Key 权限、团队成员状态或账户额度不满足要求'}`
        : `User access/quota rejected: ${cleanMsg || 'insufficient permission or quota'}`
    } else {
      headline = isZh
        ? `用户调用失败：请求格式、参数或协议不符合要求（HTTP ${primaryStatus}）`
        : `User request rejected (HTTP ${primaryStatus})`
    }
    return {
      blame: 'user',
      ownerLabel: isZh ? '用户问题' : 'User',
      subLabel,
      badgeClass: 'bg-blue-100/[0.18] text-blue-800 dark:bg-blue-900/30 dark:text-blue-200',
      headline,
      detail: bestRawDetail === headline || (Boolean(cleanMsg) && headline.includes(cleanMsg)) ? '' : bestRawDetail,
      filterHint,
      gatewayStatus: gwStatus,
      upstreamStatus: upStatus
    }
  }

  const grp = formatGroupTag(log, isZh)
  const modelName = formatModelTag(log)
  let subLabel = isZh ? '网关内部异常' : 'Gateway Internal'
  let headline = ''
  if (msgLower.includes('session routing is temporarily unavailable') || msgLower.includes('takeover')) {
    subLabel = isZh ? '会话迁移异常' : 'Session Route'
    headline = isZh
      ? `本网关问题：原粘性账号繁忙，迁移至候选账号时未通过响应校验或等待超时`
      : `Gateway issue: session route migration failed or timed out`
  } else if (log.is_routing_capacity_limited || phase === 'routing' || msgLower.includes('no available') || msgLower.includes('temporarily unavailable')) {
    subLabel = isZh ? '渠道池无可用号' : 'Pool Exhausted'
    headline = isZh
      ? `本网关调度受限：分组 ${grp} 当前没有可调度账号处理模型 [${modelName}]（可能全部限流、满载或临时熔断）`
      : `Gateway capacity exhausted in ${grp} for model [${modelName}]`
  } else {
    headline = isZh
      ? `本网关问题：网关内部处理异常（阶段：${phase || 'internal'}，状态码 ${primaryStatus}）`
      : `Gateway internal error (phase: ${phase || 'internal'}, HTTP ${primaryStatus})`
  }

  return {
    blame: 'gateway',
    ownerLabel: isZh ? '本网关问题' : 'Gateway',
    subLabel,
    badgeClass: 'bg-purple-100/[0.18] text-purple-800 dark:bg-purple-900/30 dark:text-purple-200',
    headline,
    detail: bestRawDetail,
    filterHint,
    gatewayStatus: gwStatus,
    upstreamStatus: upStatus
  }
}

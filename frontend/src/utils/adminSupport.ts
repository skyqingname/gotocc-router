export type AdminSupportResource =
  | 'overview'
  | 'api-keys'
  | 'usage'
  | 'channel-status'
  | 'wallet'
  | 'profile'
  | 'affiliate'
  | 'custom'

// 站长中心在协助模式下只读目标用户的「邀请返利」部分。
export const selfPaths: Record<AdminSupportResource, string> = {
  overview: '/dashboard',
  'api-keys': '/keys',
  usage: '/usage',
  'channel-status': '/monitor',
  wallet: '/wallet',
  profile: '/profile',
  affiliate: '/reseller',
  custom: '/custom/:id'
}

export function parseAdminSupportTargetId(value: unknown): number | null {
  const raw = Array.isArray(value) ? value[0] : value
  if (typeof raw !== 'string' && typeof raw !== 'number') return null
  const normalized = String(raw).trim()
  if (!/^\d+$/.test(normalized)) return null
  const userId = Number(normalized)
  return Number.isSafeInteger(userId) && userId > 0 ? userId : null
}

export function adminSupportPath(userId: number, resource: AdminSupportResource): string {
  return `/admin/support/users/${userId}/${resource}`
}

export function selfPathForSupportResource(resource: AdminSupportResource): string {
  return selfPaths[resource]
}

export function supportResourceFromPath(path: string): AdminSupportResource | null {
  const match = path.match(/^\/admin\/support\/users\/\d+\/(overview|api-keys|usage|channel-status|wallet|profile|affiliate|custom)(?:\/|$)/)
  return (match?.[1] as AdminSupportResource | undefined) ?? null
}

export function supportResourceForPersonalPath(path: string): AdminSupportResource {
  if (path === '/wallet' || path.startsWith('/wallet/')) return 'wallet'
  if (path === '/reseller' || path.startsWith('/reseller/')) return 'affiliate'
  if (path.startsWith('/custom/')) return 'custom'
  if (path === '/usage' || path.startsWith('/usage/')) return 'usage'
  if (path === '/monitor' || path.startsWith('/monitor/') || path === '/available-channels') return 'channel-status'
  if (path === '/profile' || path.startsWith('/profile/')) return 'profile'
  if (path === '/keys' || path.startsWith('/keys/')) return 'api-keys'
  return supportResourceFromPath(path) ?? 'overview'
}

export function accountSelectionDestination(
  currentPath: string,
  actorUserId: number,
  targetUserId: number
): string | null {
  const currentSupportResource = supportResourceFromPath(currentPath)
  if (targetUserId === actorUserId) {
    return currentSupportResource ? currentSupportResource === 'custom' ? '/custom/' + currentPath.split('/custom/')[1] : selfPathForSupportResource(currentSupportResource) : null
  }
  const resource = supportResourceForPersonalPath(currentPath)
  return adminSupportPath(targetUserId, resource) + (resource === 'custom' ? '/' + currentPath.split('/custom/')[1] : '')
}

export function supportPathForPersonalPath(userId: number, path: string): string | null {
  const entry = Object.entries(selfPaths).find(([, self]) => self === path || (self === '/custom/:id' && path.startsWith('/custom/')))
  if (!entry) return null
  const resource = entry[0] as AdminSupportResource
  return adminSupportPath(userId, resource) + (resource === 'custom' ? '/' + path.slice('/custom/'.length) : '')
}

// An invalid target must never fall back to the caller's writable personal
// pages. Send the administrator to a management page instead.
export const ADMIN_SUPPORT_SAFE_LANDING = '/admin/users'

export type AdminSupportRouteDecision =
  | { kind: 'safe-landing'; path: string }
  | { kind: 'self'; path: string }
  | { kind: 'support'; userId: number }

export function resolveAdminSupportRouteDecision(input: {
  resource: AdminSupportResource
  targetUserId: number | null
  actorUserId: number
  customId?: unknown
}): AdminSupportRouteDecision {
  if (input.targetUserId === null) {
    return { kind: 'safe-landing', path: ADMIN_SUPPORT_SAFE_LANDING }
  }
  if (input.targetUserId !== input.actorUserId) {
    return { kind: 'support', userId: input.targetUserId }
  }
  if (input.resource === 'custom') {
    const customId = typeof input.customId === 'string' ? input.customId : ''
    if (!customId) return { kind: 'safe-landing', path: ADMIN_SUPPORT_SAFE_LANDING }
    return { kind: 'self', path: `/custom/${customId}` }
  }
  return { kind: 'self', path: selfPathForSupportResource(input.resource) }
}

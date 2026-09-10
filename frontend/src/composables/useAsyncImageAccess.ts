import { computed, ref } from 'vue'
import { keysAPI } from '@/api/keys'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey, ApiKeyRoutingCapabilities } from '@/types'

const loaded = ref(false)
const loading = ref(false)
const hasManageableAsyncImageKey = ref(false)
let pendingLoad: Promise<boolean> | null = null
const pageSize = 100
const autoRoutingCapabilities = new Map<number, ApiKeyRoutingCapabilities>()
const pendingAutoRoutingCapabilities = new Map<number, Promise<ApiKeyRoutingCapabilities | null>>()
let autoRoutingCapabilitiesGeneration = 0

export function keyCanManageAsyncImage(key: ApiKey): boolean {
  return key.status === 'active' || key.status === 'quota_exhausted' || key.status === 'expired'
}

export function isAutoRoutingKey(key: Pick<ApiKey, 'routing_mode'>): boolean {
  return key.routing_mode === 'auto'
}

export function keyAllowsAsyncImage(
  key: ApiKey,
  capabilities?: Pick<ApiKeyRoutingCapabilities, 'async_image_submit'>,
): boolean {
  if (isAutoRoutingKey(key)) {
    return key.status === 'active' && capabilities?.async_image_submit === true
  }
  return key.status === 'active'
    && (key.group?.platform === 'openai' || key.group?.platform === 'grok')
    && key.group?.allow_image_generation === true
}

// Auto capabilities are loaded only for a selected/inspected key. The shared
// promise cache also lets the batch-image entry reuse the same response.
export async function getAutoRoutingCapabilities(key: ApiKey): Promise<ApiKeyRoutingCapabilities | null> {
  if (!isAutoRoutingKey(key)) return null
  const cached = autoRoutingCapabilities.get(key.id)
  if (cached) return cached
  const pending = pendingAutoRoutingCapabilities.get(key.id)
  if (pending) return pending

  const generation = autoRoutingCapabilitiesGeneration
  const request: Promise<ApiKeyRoutingCapabilities | null> = keysAPI.getRoutingCapabilities(key.id)
    .then((capabilities) => {
      if (generation !== autoRoutingCapabilitiesGeneration) return null
      if (capabilities.routing_mode !== 'auto') return null
      autoRoutingCapabilities.set(key.id, capabilities)
      return capabilities
    })
    .catch(() => null)
    .finally(() => {
      if (pendingAutoRoutingCapabilities.get(key.id) === request) {
        pendingAutoRoutingCapabilities.delete(key.id)
      }
    })
  pendingAutoRoutingCapabilities.set(key.id, request)
  return request
}

export function clearAutoRoutingCapabilities() {
  autoRoutingCapabilitiesGeneration += 1
  // Capabilities reflect the owner's current group permissions, so changing
  // any one key can invalidate the shared owner-scoped view.
  autoRoutingCapabilities.clear()
  pendingAutoRoutingCapabilities.clear()
}

// Administrators need the entry even when they have not created a personal API
// key, so they can verify the feature and manage the group-level permission.
// Any non-disabled user key may own history after its status, group, or platform
// changes, so management access cannot be gated on current submission support.
export function canViewAsyncImage(isAdmin: boolean, hasManagementKey: boolean): boolean {
  return isAdmin || hasManagementKey
}

async function loadAsyncImageAccess(force = false): Promise<boolean> {
  const authStore = useAuthStore()
  if (!authStore.isAuthenticated) {
    loaded.value = true
    hasManageableAsyncImageKey.value = false
    return false
  }
  if (loaded.value && !force) return hasManageableAsyncImageKey.value
  if (pendingLoad && !force) return pendingLoad

  loading.value = true
  pendingLoad = (async () => {
    let page = 1
    while (true) {
      const response = await keysAPI.list(page, pageSize, { sort_by: 'created_at', sort_order: 'desc' })
      if ((response.items || []).some(keyCanManageAsyncImage)) {
        hasManageableAsyncImageKey.value = true
        loaded.value = true
        return true
      }
      const pages = Number.isFinite(response.pages) && response.pages > 0 ? response.pages : 1
      if (page >= pages || (response.items || []).length === 0) {
        hasManageableAsyncImageKey.value = false
        loaded.value = true
        return false
      }
      page += 1
    }
  })()
    .catch(() => {
      hasManageableAsyncImageKey.value = false
      loaded.value = true
      return false
    })
    .finally(() => {
      loading.value = false
      pendingLoad = null
    })
  return pendingLoad
}

export function useAsyncImageAccess() {
  const authStore = useAuthStore()

  return {
    canUseAsyncImage: computed(() => canViewAsyncImage(authStore.isAdmin, hasManageableAsyncImageKey.value)),
    asyncImageAccessLoaded: computed(() => loaded.value),
    asyncImageAccessLoading: computed(() => loading.value),
    refreshAsyncImageAccess: loadAsyncImageAccess,
  }
}

import { computed, ref, watch } from 'vue'
import { keysAPI } from '@/api/keys'
import { useUserView as useAuthStore } from '@/composables/useUserView'
import { adminSupportContext, supportRequestGeneration } from '@/utils/adminSupportContext'
import type { ApiKey, ApiKeyRoutingCapabilities } from '@/types'
import { getAutoRoutingCapabilities, isAutoRoutingKey } from './useAsyncImageAccess'

const loaded = ref(false)
const loading = ref(false)
const hasAllowedBatchImageKey = ref(false)
let pendingLoad: Promise<boolean> | null = null
const pageSize = 100

export function keyAllowsBatchImage(
  key: ApiKey,
  capabilities?: Pick<ApiKeyRoutingCapabilities, 'batch_image_submit'>,
): boolean {
  if (isAutoRoutingKey(key)) {
    return key.status === 'active' && capabilities?.batch_image_submit === true
  }
  return (
    key.status === 'active' &&
    key.group?.platform === 'gemini' &&
    key.group?.allow_batch_image_generation === true
  )
}

async function loadBatchImageAccess(force = false): Promise<boolean> {
  const authStore = useAuthStore()
  if (!authStore.isAuthenticated) {
    loaded.value = true
    hasAllowedBatchImageKey.value = false
    return false
  }

  if (loaded.value && !force) {
    return hasAllowedBatchImageKey.value
  }

  if (pendingLoad && !force) {
    return pendingLoad
  }

  const scope = supportRequestGeneration()
  loading.value = true
  pendingLoad = (async () => {
    let page = 1
    while (true) {
      const response = await keysAPI.list(page, pageSize, {
        status: 'active',
        sort_by: 'created_at',
        sort_order: 'desc'
      })

      if (scope !== supportRequestGeneration()) return false
      for (const key of response.items || []) {
        if (!isAutoRoutingKey(key) && keyAllowsBatchImage(key)) {
          hasAllowedBatchImageKey.value = true
          loaded.value = true
          return true
        }
        if (isAutoRoutingKey(key)) {
          const capabilities = await getAutoRoutingCapabilities(key)
          if (keyAllowsBatchImage(key, capabilities || undefined)) {
            hasAllowedBatchImageKey.value = true
            loaded.value = true
            return true
          }
        }
      }

      if (page >= response.pages || (response.items || []).length === 0) {
        hasAllowedBatchImageKey.value = false
        loaded.value = true
        return false
      }

      page += 1
    }
  })()
    .catch(() => {
      if (scope !== supportRequestGeneration()) return false
      hasAllowedBatchImageKey.value = false
      loaded.value = true
      return false
    })
    .finally(() => {
      if (scope !== supportRequestGeneration()) return
      loading.value = false
      pendingLoad = null
    })

  return pendingLoad
}

export function useBatchImageAccess() {
  const canUseBatchImage = computed(() => hasAllowedBatchImageKey.value)

  return {
    canUseBatchImage,
    batchImageAccessLoaded: computed(() => loaded.value),
    batchImageAccessLoading: computed(() => loading.value),
    refreshBatchImageAccess: loadBatchImageAccess,
  }
}

watch(adminSupportContext, () => {
  loaded.value = false
  loading.value = false
  pendingLoad = null
  hasAllowedBatchImageKey.value = false
}, { flush: 'sync' })

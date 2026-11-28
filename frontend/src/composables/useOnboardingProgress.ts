import { computed, ref } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { usageAPI } from '@/api/usage'
import { useAuthStore } from '@/stores/auth'
import consoleGuide from '../../../console-guide.json'

// 「开始使用」三步的完成情况，全部来自真实数据：
// 有密钥算第 1 步；在 CC Switch 导入弹窗里点过「打开 CC Switch」算第 2 步（记在本浏览器）；
// 有用量记录算第 3 步。仪表盘的设置引导卡片收起后也记在本浏览器。

export type OnboardingStepId = 'key' | 'ccswitch' | 'request'

const prefix = consoleGuide.onboarding.storage_prefix
const counts = ref<{ keys: number; requests: number } | null>(null)
const ccswitchDone = ref(false)
const dismissed = ref(false)

const storageKey = (name: string) => `${prefix}${name}_${useAuthStore().user!.id}`

function readFlags() {
  ccswitchDone.value = localStorage.getItem(storageKey('ccswitch')) === '1'
  dismissed.value = localStorage.getItem(storageKey('dismissed')) === '1'
}

/** 每一步的去处：密钥页带上要直接打开的弹窗，使用记录直接进明细。 */
export const ONBOARDING_STEP_ROUTES: Record<OnboardingStepId, RouteLocationRaw> = {
  key: { path: '/keys', query: { onboarding: 'create' } },
  ccswitch: { path: '/keys', query: { onboarding: 'ccswitch' } },
  request: { path: '/usage', query: { view: 'records' } }
}

/** 第 2 步：用户在导入弹窗里点了「打开 CC Switch」。 */
export function markCcSwitchImported() {
  localStorage.setItem(storageKey('ccswitch'), '1')
  ccswitchDone.value = true
}

export function useOnboardingProgress() {
  async function refresh() {
    readFlags()
    const stats = await usageAPI.getDashboardStats()
    counts.value = { keys: stats.total_api_keys, requests: stats.total_requests }
  }

  const steps = computed(() => counts.value
    ? [
        { id: 'key' as OnboardingStepId, done: counts.value.keys > 0 },
        // A request already on record means a client is connected, whichever way it was set up.
        { id: 'ccswitch' as OnboardingStepId, done: ccswitchDone.value || counts.value.requests > 0 },
        { id: 'request' as OnboardingStepId, done: counts.value.requests > 0 }
      ]
    : [])
  const loaded = computed(() => counts.value !== null)
  const doneCount = computed(() => steps.value.filter(step => step.done).length)
  const allDone = computed(() => loaded.value && doneCount.value === steps.value.length)
  const hasKey = computed(() => counts.value !== null && counts.value.keys > 0)

  /** 仪表盘卡片收起。 */
  function dismiss() {
    localStorage.setItem(storageKey('dismissed'), '1')
    dismissed.value = true
  }

  /** 从头像菜单或 ⌘K 重新打开设置引导。 */
  function reopen() {
    localStorage.removeItem(storageKey('dismissed'))
    dismissed.value = false
  }

  return { refresh, steps, loaded, doneCount, allDone, hasKey, dismissed, dismiss, reopen }
}

import { ref } from 'vue'
import { apiClient } from '@/api/client'
import type { AutoGroupRoutingPolicy } from '@/api/admin/apiKeys'
import type { RoutingPreference } from '@/api/keyRoutingPolicy'

export const revision = ref(0)
export const demoGroups = [
  { id: 4, name: 'GPT PRO分组', platform: 'openai' },
  { id: 15, name: 'GPT PLUS分组', platform: 'openai' },
  { id: 1, name: 'cc kiro free', platform: 'anthropic' },
  { id: 8, name: 'GPT稳定官key', platform: 'openai' },
  { id: 12, name: 'CC kiro 企业级', platform: 'anthropic' },
  { id: 21, name: 'CC Max满血官方', platform: 'anthropic' },
]
export const adminPolicy = ref<AutoGroupRoutingPolicy>({ allow_user_override: true, default_group_order: [4,15,1,8,12,21], model_rules: [] })
export const preference = ref<RoutingPreference | null>(null)
const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value))

// Local demonstration data only. Requests cannot reach a live backend from this
// entry point, and this adapter is not imported by the production application.
export function installPreviewAdapter() {
  apiClient.defaults.adapter = async (config) => {
    const url = config.url || ''
    const payload = typeof config.data === 'string' && config.data ? JSON.parse(config.data) : config.data
    let data: unknown
    if (url === '/admin/api-keys/routing-policy') {
      if (config.method === 'put') { adminPolicy.value = clone(payload); revision.value++ }
      data = clone(adminPolicy.value)
    } else if (url.startsWith('/admin/groups')) {
      data = demoGroups.map(group => ({ ...group, status: 'active', sort_order: group.id }))
    } else if (url === '/groups/routing-priorities') {
      data = {
        allow_user_override: adminPolicy.value.allow_user_override,
        default_source: 'administrator',
        groups: [...demoGroups].sort((a,b) => adminPolicy.value.default_group_order.indexOf(a.id) - adminPolicy.value.default_group_order.indexOf(b.id)),
        model_rules: [],
      }
    } else if (url === '/keys/101/routing-policy') {
      if (config.method === 'put') {
        if (!adminPolicy.value.allow_user_override) throw new Error('管理员已关闭自定义权限')
        preference.value = clone(payload)
      }
      data = { allow_user_override: adminPolicy.value.allow_user_override, preference: clone(preference.value), available_groups: demoGroups }
    } else { throw new Error(`Preview fixture missing: ${url}`) }
    return { data: { code: 0, message: 'ok', data }, status: 200, statusText: 'OK', headers: {}, config }
  }
}

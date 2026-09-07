import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const {
  getAllGroups,
  getUserApiKeys,
  showError,
  showSuccess,
  updateApiKeyRouting,
} = vi.hoisted(() => ({
  getAllGroups: vi.fn(),
  getUserApiKeys: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  updateApiKeyRouting: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    apiKeys: { updateApiKeyRouting },
    groups: { getAll: getAllGroups },
    users: { getUserApiKeys },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => ({
        'admin.users.none': 'None',
        'admin.users.routingAuto': 'Smart routing',
        'admin.users.routingChangedSuccess': 'Routing mode updated',
      })[key] || key,
    }),
  }
})

import UserApiKeysModal from '../UserApiKeysModal.vue'

const fixedKey = {
  id: 17,
  user_id: 9,
  name: 'fixed-key',
  group_id: null,
  routing_mode: 'fixed',
  status: 'active',
  has_ip_whitelist: false,
  ip_whitelist_size: 0,
  has_ip_blacklist: false,
  ip_blacklist_size: 0,
  last_used_at: null,
  last_used_ip: null,
  quota: 0,
  quota_used: 0,
  expires_at: null,
  created_at: '2026-09-05T00:00:00Z',
  updated_at: '2026-09-05T00:00:00Z',
  current_concurrency: 0,
  rate_limit_5h: 0,
  rate_limit_1d: 0,
  rate_limit_7d: 0,
  usage_5h: 0,
  usage_1d: 0,
  usage_7d: 0,
  window_5h_start: null,
  window_1d_start: null,
  window_7d_start: null,
  reset_5h_at: null,
  reset_1d_at: null,
  reset_7d_at: null,
}

describe('UserApiKeysModal smart routing', () => {
  beforeEach(() => {
    getAllGroups.mockReset()
    getUserApiKeys.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    updateApiKeyRouting.mockReset()
    getAllGroups.mockResolvedValue([])
    getUserApiKeys.mockResolvedValue({ items: [fixedKey] })
    updateApiKeyRouting.mockResolvedValue({
      api_key: { ...fixedKey, group_id: null, routing_mode: 'auto' },
      auto_granted_group_access: false,
    })
  })

  it('switches a fixed key to smart routing without fabricating a group ID', async () => {
    const wrapper = mount(UserApiKeysModal, {
      props: {
        show: false,
        user: { id: 9, email: 'user@example.test', username: 'user' },
      } as any,
      global: {
        stubs: {
          BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /></div>' },
          GroupBadge: true,
          GroupOptionItem: true,
          Teleport: true,
        },
      },
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    const groupButton = wrapper.findAll('button').find((button) => button.text().includes('None'))
    expect(groupButton).toBeDefined()
    await groupButton!.trigger('click')
    const autoOption = wrapper.findAll('button').find((button) => button.text().includes('Smart routing'))
    expect(autoOption).toBeDefined()
    await autoOption!.trigger('click')
    await flushPromises()

    expect(updateApiKeyRouting).toHaveBeenCalledWith(17, 'auto', null)
    expect(wrapper.text()).toContain('Smart routing')
  })
})

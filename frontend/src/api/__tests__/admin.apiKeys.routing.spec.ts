import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, put } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, put },
}))

import { getApiKeyRoutingCapabilities, updateApiKeyRouting } from '@/api/admin/apiKeys'

describe('administrator API-key routing API', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
    get.mockResolvedValue({ data: { routing_mode: 'auto', protocols: [], async_image_submit: false, batch_image_submit: false } })
    put.mockResolvedValue({ data: { api_key: {}, auto_granted_group_access: false } })
  })

  it('makes automatic routing explicit and clears the fixed group binding', async () => {
    await updateApiKeyRouting(42, 'auto', 8)

    expect(put).toHaveBeenCalledWith('/admin/api-keys/42', {
      routing_mode: 'auto',
      group_id: null,
    })
  })

  it('uses the administrator capability endpoint by key ID', async () => {
    await expect(getApiKeyRoutingCapabilities(42)).resolves.toMatchObject({ routing_mode: 'auto' })
    expect(get).toHaveBeenCalledWith('/admin/api-keys/42/routing-capabilities')
  })
})

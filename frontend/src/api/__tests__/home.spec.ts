import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get },
}))

import { getHomepageModelPlaza, getHomepageStats } from '@/api/home'

describe('home API', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('loads only the public lightweight homepage statistics', async () => {
    const stats = { today_tokens: 12, total_tokens: 34, total_users: 5 }
    get.mockResolvedValueOnce({ data: stats })

    await expect(getHomepageStats()).resolves.toEqual(stats)
    expect(get).toHaveBeenCalledWith('/marketplace/stats')
  })

  it('uses the Plus Model Plaza endpoint for the homepage model summary', async () => {
    const response = { description: '', groups: [] }
    get.mockResolvedValueOnce({ data: response })

    await expect(getHomepageModelPlaza()).resolves.toEqual(response)
    expect(get).toHaveBeenCalledWith('/model-plaza', { signal: undefined })
  })
})

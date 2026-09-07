import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post },
}))

import { getAvailable, getUserGroupRates } from '@/api/groups'
import { create, list } from '@/api/keys'

describe('team scope API contracts', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    get.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20, pages: 0 } })
    post.mockResolvedValue({ data: {} })
  })

  it('sends scope=team when listing team keys', async () => {
    await list(2, 50, { status: 'active', scope: 'team' })

    expect(get).toHaveBeenCalledWith('/keys', {
      params: { page: 2, page_size: 50, status: 'active', scope: 'team' },
      signal: undefined,
    })
  })

  it('keeps scope=team in the API key creation payload', async () => {
    await create('team-key', 7, undefined, undefined, undefined, undefined, undefined, undefined, 'team')

    expect(post).toHaveBeenCalledWith('/keys', {
      name: 'team-key',
      group_id: 7,
      scope: 'team',
    })
  })

  it('requests team-specific groups and rates', async () => {
    get.mockResolvedValue({ data: [] })
    await getAvailable('team')
    get.mockResolvedValue({ data: { 7: 0.8 } })
    await getUserGroupRates('team')

    expect(get).toHaveBeenNthCalledWith(1, '/groups/available', { params: { scope: 'team' } })
    expect(get).toHaveBeenNthCalledWith(2, '/groups/rates', { params: { scope: 'team' } })
  })
})

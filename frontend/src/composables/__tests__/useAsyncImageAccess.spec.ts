import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ApiKey } from '@/types'

const getRoutingCapabilities = vi.hoisted(() => vi.fn())

vi.mock('@/api/keys', () => ({
  keysAPI: { getRoutingCapabilities },
}))

import { canViewAsyncImage, clearAutoRoutingCapabilities, getAutoRoutingCapabilities, keyAllowsAsyncImage, keyCanManageAsyncImage } from '../useAsyncImageAccess'

function apiKey(overrides: Partial<ApiKey>): ApiKey {
  return {
    id: 1,
    user_id: 1,
    key: 'sk-test',
    name: 'test',
    group_id: 1,
    status: 'active',
    ip_whitelist: [],
    ip_blacklist: [],
    last_used_at: null,
    last_used_ip: null,
    quota: 0,
    quota_used: 0,
    expires_at: null,
    created_at: '',
    updated_at: '',
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
    ...overrides,
  }
}

describe('async image access', () => {
  beforeEach(() => {
    clearAutoRoutingCapabilities()
    getRoutingCapabilities.mockReset()
  })

  it('always lets administrators view the feature', () => {
    expect(canViewAsyncImage(true, false)).toBe(true)
  })

  it('requires an eligible API key for regular users', () => {
    expect(canViewAsyncImage(false, false)).toBe(false)
    expect(canViewAsyncImage(false, true)).toBe(true)
  })

  it('accepts only active OpenAI or Grok keys from image-enabled groups', () => {
    expect(keyAllowsAsyncImage(apiKey({ group: { platform: 'openai', allow_image_generation: true } } as Partial<ApiKey>))).toBe(true)
    expect(keyAllowsAsyncImage(apiKey({ group: { platform: 'grok', allow_image_generation: true } } as Partial<ApiKey>))).toBe(true)
    expect(keyAllowsAsyncImage(apiKey({ group: { platform: 'openai', allow_image_generation: false } } as Partial<ApiKey>))).toBe(false)
    expect(keyAllowsAsyncImage(apiKey({ status: 'inactive', group: { platform: 'openai', allow_image_generation: true } } as Partial<ApiKey>))).toBe(false)
  })

  it('uses the server capability for an automatic-routing key instead of a fabricated group platform', () => {
    const autoKey = {
      ...apiKey({ group_id: null, group: undefined }),
      routing_mode: 'auto',
    } as ApiKey
    const withCapabilities = keyAllowsAsyncImage as unknown as (
      key: ApiKey,
      capabilities: { async_image_submit: boolean },
    ) => boolean

    expect(withCapabilities(autoKey, { async_image_submit: true })).toBe(true)
    expect(withCapabilities(autoKey, { async_image_submit: false })).toBe(false)
  })

  it('shares one capability request for repeated checks of the same automatic key', async () => {
    const autoKey = { ...apiKey({}), routing_mode: 'auto' } as ApiKey
    getRoutingCapabilities.mockResolvedValue({
      routing_mode: 'auto',
      protocols: ['openai'],
      async_image_submit: true,
      batch_image_submit: false,
    })

    const [first, second] = await Promise.all([
      getAutoRoutingCapabilities(autoKey),
      getAutoRoutingCapabilities(autoKey),
    ])

    expect(first?.async_image_submit).toBe(true)
    expect(second?.async_image_submit).toBe(true)
    expect(getRoutingCapabilities).toHaveBeenCalledTimes(1)
    expect(getRoutingCapabilities).toHaveBeenCalledWith(1)
  })

  it('keeps a newer pending capability request after the cache is invalidated', async () => {
    const autoKey = { ...apiKey({}), routing_mode: 'auto' } as ApiKey
    let resolveFirst!: (value: { routing_mode: 'auto'; protocols: string[]; async_image_submit: boolean; batch_image_submit: boolean }) => void
    let resolveSecond!: (value: { routing_mode: 'auto'; protocols: string[]; async_image_submit: boolean; batch_image_submit: boolean }) => void
    const firstResponse = new Promise<{ routing_mode: 'auto'; protocols: string[]; async_image_submit: boolean; batch_image_submit: boolean }>((resolve) => { resolveFirst = resolve })
    const secondResponse = new Promise<{ routing_mode: 'auto'; protocols: string[]; async_image_submit: boolean; batch_image_submit: boolean }>((resolve) => { resolveSecond = resolve })
    getRoutingCapabilities
      .mockReturnValueOnce(firstResponse)
      .mockReturnValueOnce(secondResponse)
      .mockResolvedValue({ routing_mode: 'auto', protocols: [], async_image_submit: true, batch_image_submit: false })

    const first = getAutoRoutingCapabilities(autoKey)
    clearAutoRoutingCapabilities()
    const second = getAutoRoutingCapabilities(autoKey)
    resolveFirst({ routing_mode: 'auto', protocols: [], async_image_submit: false, batch_image_submit: false })
    await first

    const repeated = getAutoRoutingCapabilities(autoKey)
    expect(getRoutingCapabilities).toHaveBeenCalledTimes(2)

    resolveSecond({ routing_mode: 'auto', protocols: [], async_image_submit: true, batch_image_submit: false })
    await expect(second).resolves.toMatchObject({ async_image_submit: true })
    await expect(repeated).resolves.toMatchObject({ async_image_submit: true })
  })

  it('keeps every non-disabled key available for owner-scoped task management', () => {
    expect(keyCanManageAsyncImage(apiKey({ status: 'active', group: { platform: 'openai', allow_image_generation: false } } as Partial<ApiKey>))).toBe(true)
    expect(keyCanManageAsyncImage(apiKey({ status: 'quota_exhausted', group: { platform: 'openai', allow_image_generation: false } } as Partial<ApiKey>))).toBe(true)
    expect(keyCanManageAsyncImage(apiKey({ status: 'expired', group: { platform: 'grok', allow_image_generation: false } } as Partial<ApiKey>))).toBe(true)
    expect(keyCanManageAsyncImage(apiKey({ status: 'active', group: { platform: 'anthropic', allow_image_generation: false } } as Partial<ApiKey>))).toBe(true)
    expect(keyCanManageAsyncImage(apiKey({ status: 'active', group_id: null, group: undefined }))).toBe(true)
    expect(keyCanManageAsyncImage(apiKey({ status: 'inactive', group: { platform: 'openai', allow_image_generation: true } } as Partial<ApiKey>))).toBe(false)
  })
})

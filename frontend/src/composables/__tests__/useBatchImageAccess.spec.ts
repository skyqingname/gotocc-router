import { describe, expect, it } from 'vitest'
import type { ApiKey } from '@/types'
import { keyAllowsBatchImage } from '../useBatchImageAccess'

function apiKey(overrides: Partial<ApiKey>): ApiKey {
  return {
    id: 1,
    user_id: 1,
    key: 'sk-test',
    name: 'test',
    group_id: null,
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

describe('batch image access', () => {
  it('uses server capability instead of inventing a Gemini group for automatic routing', () => {
    const autoKey = { ...apiKey({}), routing_mode: 'auto' } as ApiKey

    expect(keyAllowsBatchImage(autoKey, { batch_image_submit: true })).toBe(true)
    expect(keyAllowsBatchImage(autoKey, { batch_image_submit: false })).toBe(false)
  })
})

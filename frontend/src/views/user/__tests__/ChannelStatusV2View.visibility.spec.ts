import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'

const mocks = vi.hoisted(() => ({
  route: { query: {} as Record<string, unknown> },
  router: { replace: vi.fn() },
  authStore: { isAdmin: false },
  appStore: { showError: vi.fn(), cachedPublicSettings: {} },
  getDimensions: vi.fn(),
  getSnapshot: vi.fn(),
  getMatrix: vi.fn(),
  getModels: vi.fn(),
  getErrors: vi.fn(),
  getUsers: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
  useRouter: () => mocks.router,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      te: () => false,
      locale: { value: 'en' },
    }),
  }
})

vi.mock('@/stores/auth', () => ({ useAuthStore: () => mocks.authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => mocks.appStore }))
vi.mock('@/utils/featureFlags', () => ({ isChannelMonitorThroughputHidden: () => false }))
vi.mock('@/api/channelMonitorV2', () => ({
  getDimensions: mocks.getDimensions,
  getSnapshot: mocks.getSnapshot,
  getMatrix: mocks.getMatrix,
  getModels: mocks.getModels,
  getErrors: mocks.getErrors,
  getUsers: mocks.getUsers,
}))

import ChannelStatusV2View from '../ChannelStatusV2View.vue'

const latency = {
  sample_count: 0,
  p50_ms: null,
  p90_ms: null,
  p95_ms: null,
  avg_ms: null,
}
const metrics = {
  success_requests: 0,
  error_requests: 0,
  request_count: 0,
  token_count: 0,
  rpm: 0,
  tpm: 0,
  error_rate: 0,
  cache_rate: 0,
  cache_rate_numerator: 0,
  cache_rate_denominator: 0,
  ttft: latency,
  duration: latency,
}
const coverage = {
  requested_start: '2026-08-15T00:00:00Z',
  coverage_start: '2026-08-15T00:00:00Z',
  data_through: '2026-08-15T01:00:00Z',
  computed_at: '2026-08-15T01:00:00Z',
  aggregation_lag_seconds: 0,
  coverage_complete: true,
  bucket_seconds: 300,
}
const health = {
  overall: 'healthy',
  error_rate: 'healthy',
  ttft: 'healthy',
  cache: 'healthy',
  minimum_sample: 1,
}

function mountView() {
  return shallowMount(ChannelStatusV2View, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
      },
    },
  })
}

describe('ChannelStatusV2View user-ranking visibility', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.route.query = { tab: 'users' }
    mocks.authStore.isAdmin = false
    mocks.getDimensions.mockResolvedValue({ platforms: [], groups: [], models: [] })
    mocks.getSnapshot.mockResolvedValue({
      coverage,
      metrics,
      health,
      trend: [],
      config: { refresh_interval_seconds: 300 },
    })
    mocks.getMatrix.mockResolvedValue({ coverage, group_by: 'platform_group', items: [] })
    mocks.getModels.mockResolvedValue({ coverage, items: [] })
    mocks.getErrors.mockResolvedValue({ coverage, items: [] })
    mocks.getUsers.mockResolvedValue({ coverage, items: [] })
  })

  it('hides ranking and normalizes a legacy users link for regular users', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('channelMonitorV2.tabs.models')
    expect(wrapper.text()).toContain('channelMonitorV2.tabs.errors')
    expect(wrapper.text()).not.toContain('channelMonitorV2.tabs.users')
    expect(mocks.getModels).toHaveBeenCalledOnce()
    expect(mocks.getUsers).not.toHaveBeenCalled()
    expect(mocks.router.replace).toHaveBeenCalledWith(
      expect.objectContaining({ query: expect.objectContaining({ tab: 'models' }) }),
    )

    wrapper.unmount()
  })

  it('keeps ranking available for administrators', async () => {
    mocks.authStore.isAdmin = true
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('channelMonitorV2.tabs.users')
    expect(mocks.getUsers).toHaveBeenCalledOnce()
    expect(mocks.getModels).not.toHaveBeenCalled()

    wrapper.unmount()
  })
})

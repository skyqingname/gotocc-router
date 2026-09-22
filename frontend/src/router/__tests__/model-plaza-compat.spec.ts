import { describe, expect, it, vi } from 'vitest'

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  isAuthenticated: false,
  isAdmin: false,
  isSimpleMode: false,
}))

const appStore = vi.hoisted(() => ({
  siteName: 'GotoCC',
  backendModeEnabled: false,
  cachedPublicSettings: null as null | Record<string, unknown>,
}))

vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
}))
vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))
vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

describe('Model Plaza compatibility route', () => {
  it('redirects the retired /models page to the Plus Model Plaza', async () => {
    const { default: router } = await import('@/router')
    const route = router.getRoutes().find((record) => record.name === 'LegacyModelsRedirect')

    expect(route?.path).toBe('/models')
    expect(route?.redirect).toBe('/model-plaza')
    expect(route?.meta.requiresAuth).toBe(false)
  })

  it('keeps Plus Model Plaza as the only marketplace component', async () => {
    const { default: router } = await import('@/router')
    const plaza = router.getRoutes().find((record) => record.name === 'ModelPlaza')

    expect(plaza?.path).toBe('/model-plaza')
    expect(router.getRoutes().some((record) => record.name === 'ModelMarketplace')).toBe(false)
  })
})

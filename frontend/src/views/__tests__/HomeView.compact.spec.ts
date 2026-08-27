import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'

import HomeView from '../HomeView.vue'

const { appStore, authStore, getHomepageModelPlaza, getHomepageStats } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {} as Record<string, unknown>,
    siteName: 'Fallback site',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
  },
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    user: null as { email?: string } | null,
    checkAuth: vi.fn(),
  },
  getHomepageModelPlaza: vi.fn(),
  getHomepageStats: vi.fn(),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/api/home', () => ({
  getHomepageModelPlaza,
  getHomepageStats,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key, locale: { value: 'zh' } }),
  }
})

function mountHome(settings: Record<string, unknown> = {}) {
  appStore.cachedPublicSettings = {
    site_name: 'Test site',
    site_subtitle: 'Test subtitle',
    ...settings,
  }

  return mount(HomeView, {
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        LocaleSwitcher: { template: '<div data-testid="locale-switcher" />' },
        Icon: { template: '<span data-testid="icon" />' },
        ProviderIcon: { template: '<span data-testid="provider-icon" />' },
        GitHubMark: { template: '<span data-testid="github-mark" />' },
      },
    },
  })
}

function modelPlazaDestination(wrapper: ReturnType<typeof mountHome>) {
  return wrapper
    .findAllComponents(RouterLinkStub)
    .find((link) => link.props('to') === '/model-plaza')
    ?.props('to')
}

describe('HomeView production homepage', () => {
  beforeEach(() => {
    authStore.isAuthenticated = false
    authStore.isAdmin = false
    authStore.user = null
    authStore.checkAuth.mockClear()
    appStore.fetchPublicSettings.mockClear()
    appStore.siteName = 'Fallback site'
    getHomepageStats.mockReset()
    getHomepageModelPlaza.mockReset()
    getHomepageStats.mockResolvedValue({ today_tokens: 1, total_tokens: 2, total_users: 3 })
    getHomepageModelPlaza.mockResolvedValue({ description: '', groups: [] })
    localStorage.clear()
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
  })

  it('renders custom HTML ahead of the built-in homepage', () => {
    const wrapper = mountHome({
      home_content: '<section id="custom-home">Custom home</section>',
    })

    expect(wrapper.get('#custom-home').text()).toBe('Custom home')
    expect(wrapper.find('[data-testid="gotocc-home"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="compact-home"]').exists()).toBe(false)
  })

  it('renders custom URL content ahead of the built-in homepage', () => {
    const wrapper = mountHome({
      home_content: ' https://example.com/home ',
    })

    expect(wrapper.get('iframe').attributes('src')).toBe('https://example.com/home')
    expect(wrapper.find('[data-testid="gotocc-home"]').exists()).toBe(false)
  })

  it('does not use the Plus compact homepage', async () => {
    const wrapper = mountHome({ compact_home_enabled: true })
    await flushPromises()

    expect(wrapper.find('[data-testid="compact-home"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="gotocc-home"]').exists()).toBe(true)
    expect(wrapper.find('.ba-theme-shell').exists()).toBe(true)
  })

  it('uses GotoCC when public settings still expose the Plus default brand', async () => {
    appStore.siteName = 'Sub2API Plus'
    const wrapper = mountHome({ site_name: '' })
    await flushPromises()

    expect(wrapper.get('[data-testid="gotocc-home"]').text()).toContain('GotoCC')
  })

  it('keeps model links on Plus Model Plaza', async () => {
    const wrapper = mountHome({ model_plaza_enabled: true })
    await flushPromises()

    const hrefs = wrapper.findAllComponents(RouterLinkStub).map((link) => link.props('to'))
    expect(hrefs).toContain('/model-plaza')
    expect(hrefs).not.toContain('/models')
  })

  it('links unauthenticated visitors to login', async () => {
    const wrapper = mountHome()
    await flushPromises()
    const hrefs = wrapper.findAllComponents(RouterLinkStub).map((link) => link.props('to'))
    expect(hrefs).toContain('/login')
  })

  it('links authenticated users to their dashboard', async () => {
    authStore.isAuthenticated = true
    const wrapper = mountHome()
    await flushPromises()
    const hrefs = wrapper.findAllComponents(RouterLinkStub).map((link) => link.props('to'))
    expect(hrefs).toContain('/dashboard')
  })

  it('shows the model plaza link to anonymous visitors when public access is enabled', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      model_plaza_enabled: true,
      model_plaza_require_auth: false,
    })

    expect(modelPlazaDestination(wrapper)).toBe('/model-plaza')
  })

  it('hides the model plaza link from anonymous visitors when sign-in is required', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      model_plaza_enabled: true,
      model_plaza_require_auth: true,
    })

    expect(modelPlazaDestination(wrapper)).toBeUndefined()
  })

  it('shows the model plaza link to authenticated visitors when sign-in is required', () => {
    authStore.isAuthenticated = true

    const wrapper = mountHome({
      compact_home_enabled: true,
      model_plaza_enabled: true,
      model_plaza_require_auth: true,
    })

    expect(modelPlazaDestination(wrapper)).toBe('/model-plaza')
  })

  it('shows the model plaza link in the default home header', () => {
    const wrapper = mountHome({
      model_plaza_enabled: true,
      model_plaza_require_auth: false,
    })

    expect(modelPlazaDestination(wrapper)).toBe('/model-plaza')
  })

  it('hides the model plaza link when the feature is disabled', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      model_plaza_enabled: false,
      model_plaza_require_auth: false,
    })

    expect(modelPlazaDestination(wrapper)).toBeUndefined()
  })
})

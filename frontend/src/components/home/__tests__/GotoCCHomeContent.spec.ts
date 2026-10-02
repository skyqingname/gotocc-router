import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'

import GotoCCHomeContent from '../GotoCCHomeContent.vue'

const { getHomepageModelPlaza, getHomepageStats } = vi.hoisted(() => ({
  getHomepageModelPlaza: vi.fn(),
  getHomepageStats: vi.fn(),
}))

vi.mock('@/api/home', () => ({
  getHomepageModelPlaza,
  getHomepageStats,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const defaultProps = {
  siteName: 'GotoCC',
  siteLogo: '',
  siteSubtitle: 'Native AI APIs',
  docUrl: 'https://docs.example.com',
  isAuthenticated: false,
  dashboardPath: '/dashboard',
  isDark: false,
}

function mountHome() {
  return mount(GotoCCHomeContent, {
    props: defaultProps,
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        LocaleSwitcher: { template: '<div data-testid="locale-switcher" />' },
        Icon: { template: '<span data-testid="icon" />' },
        PlatformIcon: { template: '<span data-testid="platform-icon" />' },
      },
    },
  })
}

describe('GotoCCHomeContent', () => {
  beforeEach(() => {
    getHomepageStats.mockReset()
    getHomepageModelPlaza.mockReset()
    getHomepageStats.mockResolvedValue({
      today_tokens: 321,
      total_tokens: 654,
      total_users: 12,
    })
    getHomepageModelPlaza.mockResolvedValue({
      description: '',
      groups: [
        {
          id: 1,
          name: 'OpenAI Standard',
          description: '',
          platform: 'openai',
          subscription_type: 'standard',
          rate_multiplier: 1,
          peak_rate_enabled: false,
          peak_start: '',
          peak_end: '',
          peak_rate_multiplier: 1,
          is_exclusive: false,
          image_rate_independent: false,
          image_rate_multiplier: 1,
          models: [
            { name: 'gpt-5.5', platform: 'openai', pricing: null, official_pricing: null },
          ],
        },
      ],
    })
  })

  it('renders the GotoCC markers, public stats, and Plus model summary', async () => {
    const wrapper = mountHome()
    await flushPromises()

    expect(wrapper.get('[data-testid="gotocc-home"]').text()).toContain('GotoCC')
    expect(wrapper.text()).toContain('321')
    expect(wrapper.text()).toContain('654')
    expect(wrapper.text()).toContain('gpt-5.5')
    expect(wrapper.text()).toContain('GOTOCC_API_KEY')
    expect(getHomepageStats).toHaveBeenCalledOnce()
    expect(getHomepageModelPlaza).toHaveBeenCalledOnce()
  })

  it('keeps the default homepage usable when Model Plaza is unavailable', async () => {
    getHomepageModelPlaza.mockRejectedValueOnce(new Error('model plaza unavailable'))

    const wrapper = mountHome()
    await flushPromises()

    expect(wrapper.get('[data-testid="gotocc-home"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('home.providers.unavailable')
    expect(wrapper.findAllComponents(RouterLinkStub).some((link) => link.props('to') === '/login')).toBe(true)
    expect(wrapper.text()).toContain('GOTOCC_API_KEY')
  })
})

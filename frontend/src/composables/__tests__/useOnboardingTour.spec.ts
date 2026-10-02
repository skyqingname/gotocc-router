import { defineComponent, nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useOnboardingTour } from '../useOnboardingTour'

const { driverFactory, routerPush, currentRoute, onboardingStore } = vi.hoisted(() => ({
  driverFactory: vi.fn(),
  routerPush: vi.fn(),
  currentRoute: { value: { fullPath: '/team' } },
  onboardingStore: {
    getDriverInstance: vi.fn(() => null),
    setDriverInstance: vi.fn(),
    isDriverActive: vi.fn(() => false),
    setControlMethods: vi.fn(),
    clearControlMethods: vi.fn(),
  },
}))

vi.mock('driver.js', () => ({ driver: driverFactory }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: { id: 7, role: 'user' }, isSimpleMode: false }),
}))
vi.mock('@/stores/onboarding', () => ({ useOnboardingStore: () => onboardingStore }))
vi.mock('vue-router', () => ({
  useRouter: () => ({
    currentRoute,
    resolve: (route: { path: string, query?: Record<string, string> }) => ({
      fullPath: route.query?.scope ? `${route.path}?scope=${route.query.scope}` : route.path,
    }),
    push: routerPush,
  }),
}))

describe('useOnboardingTour team flow', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    currentRoute.value = { fullPath: '/team' }
    routerPush.mockImplementation(async (route: { path: string, query?: Record<string, string> }) => {
      currentRoute.value = {
        fullPath: route.query?.scope ? `${route.path}?scope=${route.query.scope}` : route.path,
      }
      await nextTick()
    })
  })

  it('routes between the team and team-key steps while retaining Plus driver controls', async () => {
    let activeIndex = 0
    let active = true
    let config: any
    const driverInstance = {
      drive: vi.fn((index = 0) => { activeIndex = index }),
      moveTo: vi.fn((index: number) => { activeIndex = index }),
      moveNext: vi.fn(() => { activeIndex += 1 }),
      getActiveIndex: vi.fn(() => activeIndex),
      getActiveElement: vi.fn(() => null),
      isActive: vi.fn(() => active),
      destroy: vi.fn(() => {
        active = false
        config?.onDestroyed?.()
      }),
    }
    driverFactory.mockImplementation((nextConfig) => {
      config = nextConfig
      return driverInstance
    })

    const target = document.createElement('div')
    vi.spyOn(target, 'getBoundingClientRect').mockReturnValue({ height: 40 } as DOMRect)
    const querySelector = vi.spyOn(document, 'querySelector').mockReturnValue(target)

    const Harness = defineComponent({
      setup() {
        return useOnboardingTour({ autoStart: false })
      },
      template: '<div />',
    })
    const wrapper = mount(Harness)
    await flushPromises()

    const startTeamTour = (wrapper.vm as any).startTeamTour as (options: { isOwner: boolean, hasTeam: boolean }) => void
    startTeamTour({ isOwner: true, hasTeam: true })
    await flushPromises()

    expect(config.showProgress).toBe(true)
    expect(config.onPopoverRender).toEqual(expect.any(Function))
    for (let index = 0; index < 5; index += 1) {
      await config.onNextClick(null, config.steps[index], {
        config,
        state: { activeIndex: index },
      })
    }

    expect(routerPush).toHaveBeenCalledWith({ path: '/keys', query: { scope: 'team' } })
    expect(driverInstance.moveTo).toHaveBeenLastCalledWith(5)

    await config.onPrevClick(null, config.steps[5], { state: { activeIndex: 5 } })
    expect(routerPush).toHaveBeenLastCalledWith({ path: '/team' })
    expect(driverInstance.moveTo).toHaveBeenLastCalledWith(4)

    config.onCloseClick()
    wrapper.unmount()
    querySelector.mockRestore()
  })
})

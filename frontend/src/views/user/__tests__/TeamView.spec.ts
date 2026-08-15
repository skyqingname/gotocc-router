import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { TeamContext } from '@/api/team'
import TeamView from '../TeamView.vue'

const {
  current,
  keys,
  usage,
  usageLogs,
  memberUsage,
  members,
  invitations,
  showError,
  showSuccess,
  startTeamGuide,
} = vi.hoisted(() => ({
  current: vi.fn(),
  keys: vi.fn(),
  usage: vi.fn(),
  usageLogs: vi.fn(),
  memberUsage: vi.fn(),
  members: vi.fn(),
  invitations: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  startTeamGuide: vi.fn(),
}))

vi.mock('@/api/team', () => ({
  teamAPI: {
    current,
    keys,
    usage,
    usageLogs,
    memberUsage,
    members,
    invitations,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: { team_self_service_enabled: true },
    showError,
    showSuccess,
  }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({ startTeamGuide }),
}))

vi.mock('@/composables/useStepUp', () => ({
  useStepUp: () => ({ run: (action: () => Promise<unknown>) => action() }),
  isStepUpCancelled: () => false,
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ replace: vi.fn() }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => params?.count ? `${key}:${params.count}` : key,
    }),
  }
})

const ownerContext = (status: 'active' | 'suspended' = 'active'): TeamContext => ({
  team: {
    id: 12,
    name: 'GotoCC Core',
    status,
    member_limit: 10,
    default_daily_limit_usd: 5,
    default_weekly_limit_usd: 20,
    default_monthly_limit_usd: 50,
    member_count: 1,
    created_at: '2026-08-01T00:00:00Z',
    updated_at: '2026-08-01T00:00:00Z',
  },
  membership: {
    id: 1,
    team_id: 12,
    user_id: 3,
    email: 'owner@example.com',
    username: 'owner',
    role: 'owner',
    daily_limit_usd: 0,
    weekly_limit_usd: 0,
    monthly_limit_usd: 0,
    daily_usage_usd: 0,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    joined_at: '2026-08-01T00:00:00Z',
    last_active_at: null,
  },
  owner: {
    id: 1,
    team_id: 12,
    user_id: 3,
    email: 'owner@example.com',
    username: 'owner',
    role: 'owner',
    daily_limit_usd: 0,
    weekly_limit_usd: 0,
    monthly_limit_usd: 0,
    daily_usage_usd: 0,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    joined_at: '2026-08-01T00:00:00Z',
    last_active_at: null,
  },
})

const mountView = async () => {
  const wrapper = mount(TeamView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        BaseDialog: true,
        ConfirmDialog: true,
        LoadingSpinner: true,
        TeamInvitationDialog: true,
        TeamMemberUsageCharts: true,
        TotpStepUpDialog: true,
        Icon: { props: ['name'], template: '<span>{{ name }}</span>' },
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('user TeamView', () => {
  beforeEach(() => {
    current.mockReset()
    keys.mockReset()
    usage.mockReset()
    usageLogs.mockReset()
    memberUsage.mockReset()
    members.mockReset()
    invitations.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    startTeamGuide.mockReset()

    keys.mockResolvedValue([])
    usage.mockResolvedValue({ actual_cost: 0, request_count: 0, input_tokens: 0, output_tokens: 0, daily: [] })
    usageLogs.mockResolvedValue({ items: [], total: 0, limit: 10, offset: 0 })
    memberUsage.mockResolvedValue([])
    members.mockResolvedValue([])
    invitations.mockResolvedValue([])
  })

  it('renders the self-service state when the user has no team', async () => {
    current.mockRejectedValue({ reason: 'TEAM_NOT_FOUND' })

    const wrapper = await mountView()

    expect(wrapper.text()).toContain('team.noTeam')
    expect(wrapper.text()).toContain('team.create')
    expect(keys).not.toHaveBeenCalled()
    expect(showError).not.toHaveBeenCalled()
    expect(wrapper.find('[data-tour="team-create-form"]').exists()).toBe(true)

    await wrapper.get('button.btn-secondary').trigger('click')
    expect(startTeamGuide).toHaveBeenCalledWith({ isOwner: false, hasTeam: false })
  })

  it('loads only keys for a suspended team and keeps the resume setting available', async () => {
    current.mockResolvedValue(ownerContext('suspended'))

    const wrapper = await mountView()

    expect(keys).toHaveBeenCalledOnce()
    expect(usage).not.toHaveBeenCalled()
    expect(usageLogs).not.toHaveBeenCalled()
    expect(memberUsage).not.toHaveBeenCalled()
    expect(members).not.toHaveBeenCalled()
    expect(invitations).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('team.resume')
    expect(wrapper.text()).not.toContain('team.overview')
  })

  it('shows only the masked representation of team keys', async () => {
    current.mockResolvedValue(ownerContext())
    keys.mockResolvedValue([{
      id: 9,
      user_id: 4,
      user_email: 'member@example.com',
      name: 'member-key',
      masked_key: 'sk-team...safe',
      key: 'sk-team-secret-must-not-render',
      status: 'active',
      team_owner_disabled: false,
      group_id: 2,
      group_name: 'OpenAI',
      last_used_at: null,
      created_at: '2026-08-01T00:00:00Z',
    }])

    const wrapper = await mountView()
    const keysTab = wrapper.findAll('button').find((button) => button.text().includes('team.keys'))
    await keysTab?.trigger('click')

    expect(wrapper.text()).toContain('sk-team...safe')
    expect(wrapper.text()).not.toContain('sk-team-secret-must-not-render')
  })

  it('exposes owner tour entry points and starts the routed guide', async () => {
    current.mockResolvedValue(ownerContext())

    const wrapper = await mountView()
    expect(wrapper.find('[data-tour="team-members"]').exists()).toBe(true)
    expect(wrapper.find('[data-tour="team-invitations"]').exists()).toBe(true)
    expect(wrapper.find('[data-tour="team-settings-tab"]').exists()).toBe(true)
    expect(wrapper.find('[data-tour="team-member-usage-charts"]').exists()).toBe(true)
    expect(wrapper.find('[data-tour="team-usage-records"]').exists()).toBe(true)

    const guideButton = wrapper.findAll('button').find((button) => button.text().includes('team.guideButton'))
    await guideButton?.trigger('click')
    expect(startTeamGuide).toHaveBeenCalledWith({ isOwner: true, hasTeam: true })
  })
})

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import TeamsView from '../TeamsView.vue'

const { list, showError, showSuccess } = vi.hoisted(() => ({
  list: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    teams: { list },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('@/composables/useStepUp', () => ({
  useStepUp: () => ({ run: (action: () => Promise<unknown>) => action() }),
  isStepUpCancelled: () => false,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const DataTableStub = {
  props: ['data'],
  template: `
    <div>
      <div v-for="row in data" :key="row.id">
        <slot name="cell-name" :value="row.name" :row="row" />
        <slot name="cell-owner_email" :value="row.owner_email" :row="row" />
        <slot name="cell-member_count" :value="row.member_count" :row="row" />
        <slot name="cell-status" :value="row.status" :row="row" />
      </div>
      <slot v-if="data.length === 0" name="empty" />
    </div>
  `,
}

const mountView = async () => {
  const wrapper = mount(TeamsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>' },
        DataTable: DataTableStub,
        BaseDialog: true,
        ConfirmDialog: true,
        EmptyState: true,
        LoadingSpinner: true,
        Pagination: true,
        Select: true,
        TeamMemberUsageCharts: true,
        TotpStepUpDialog: true,
        Icon: { props: ['name'], template: '<span>{{ name }}</span>' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('admin TeamsView', () => {
  beforeEach(() => {
    list.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('loads and renders the administrator team list', async () => {
    list.mockResolvedValue([{
      id: 12,
      name: 'GotoCC Core',
      status: 'active',
      member_limit: 10,
      member_count: 3,
      created_at: '2026-08-01T00:00:00Z',
      updated_at: '2026-08-01T00:00:00Z',
      owner_user_id: 3,
      owner_email: 'owner@example.com',
    }])

    const wrapper = await mountView()

    expect(list).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('GotoCC Core')
    expect(wrapper.text()).toContain('owner@example.com')
    expect(wrapper.text()).toContain('3 / 10')
    expect(wrapper.text()).toContain('team.statusActive')
    expect(showError).not.toHaveBeenCalled()
  })

  it('reports list failures without rendering stale teams', async () => {
    list.mockRejectedValue(new Error('list failed'))

    const wrapper = await mountView()

    expect(showError).toHaveBeenCalledWith('list failed')
    expect(wrapper.text()).not.toContain('GotoCC Core')
  })
})

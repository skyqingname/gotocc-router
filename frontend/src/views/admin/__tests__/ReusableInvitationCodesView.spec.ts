import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ReusableInvitationCodesView from '../ReusableInvitationCodesView.vue'

const { listCodes, createCode, disableCode, listUses, showSuccess, showError } = vi.hoisted(() => ({
  listCodes: vi.fn(),
  createCode: vi.fn(),
  disableCode: vi.fn(),
  listUses: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    reusableInvitationCodes: {
      list: listCodes,
      create: createCode,
      disable: disableCode,
      listUses
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess, showError })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: vi.fn() })
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <table>
      <tbody>
        <tr v-for="row in data" :key="row.id">
          <td v-for="column in columns" :key="column.key">
            <slot :name="'cell-' + column.key" :row="row" :value="row[column.key]">
              {{ row[column.key] }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>
  `
}

describe('ReusableInvitationCodesView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listCodes.mockResolvedValue({
      items: [{
        id: 1,
        code: 'GOTOCC-PERMANENT',
        status: 'active',
        max_uses: 0,
        used_count: 0,
        expires_at: null,
        notes: 'Permanent code',
        created_at: '2026-06-30T08:13:14Z',
        updated_at: '2026-06-30T08:13:14Z'
      }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    createCode.mockResolvedValue({ id: 2, code: 'NEW-PERM' })
    disableCode.mockResolvedValue({ id: 1, status: 'disabled' })
    listUses.mockResolvedValue([])
  })

  const mountView = () => mount(ReusableInvitationCodesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: DataTableStub,
        Pagination: true,
        BaseDialog: {
          props: ['show'],
          emits: ['close'],
          template: '<div v-if="show"><slot /></div>'
        },
        ConfirmDialog: {
          props: ['show'],
          emits: ['confirm', 'cancel'],
          template: '<button v-if="show" data-test="confirm-disable-reusable-code" @click="$emit(\'confirm\')">confirm</button>'
        },
        Icon: true
      }
    }
  })

  it('loads codes on mount using server-side sorting', async () => {
    mountView()
    await flushPromises()

    expect(listCodes).toHaveBeenCalledWith(1, 20, { sort_by: 'id', sort_order: 'desc' })
  })

  it('creates an unlimited non-expiring code by default', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="open-create-reusable-code"]').trigger('click')
    await wrapper.get('[data-test="reusable-code-input"]').setValue('NEW-PERM')
    await wrapper.get('[data-test="create-reusable-code-form"]').trigger('submit')
    await flushPromises()

    expect(createCode).toHaveBeenCalledWith({
      code: 'NEW-PERM',
      max_uses: 0,
      expires_at: null,
      notes: ''
    })
    expect(showSuccess).toHaveBeenCalledWith('admin.reusableInvitationCodes.createSuccess')
  })

  it('disables the selected active code after confirmation', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="disable-reusable-code"]').trigger('click')
    await wrapper.get('[data-test="confirm-disable-reusable-code"]').trigger('click')
    await flushPromises()

    expect(disableCode).toHaveBeenCalledWith(1)
    expect(showSuccess).toHaveBeenCalledWith('admin.reusableInvitationCodes.disableSuccess')
  })
})

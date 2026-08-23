import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AsyncImageView from '../AsyncImageView.vue'

const {
  keysList,
  deleteAsyncImageTask,
  getAsyncImageObjectURL,
  listAsyncImageTasks,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  keysList: vi.fn(),
  deleteAsyncImageTask: vi.fn(),
  getAsyncImageObjectURL: vi.fn(),
  listAsyncImageTasks: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api', () => ({
  keysAPI: { list: keysList },
}))

vi.mock('@/api/asyncImage', () => ({
  AsyncImageDownloadValidationError: class AsyncImageDownloadValidationError extends Error {},
  deleteAsyncImageTask,
  downloadAsyncImageZip: vi.fn(),
  getAsyncImageObjectURL,
  getAsyncImageTask: vi.fn(),
  listAsyncImageModels: vi.fn().mockResolvedValue([]),
  listAsyncImageTasks,
  preferredAsyncImageModel: vi.fn().mockReturnValue(''),
  saveAsyncImageBlob: vi.fn(),
  submitAsyncImageEdit: vi.fn(),
  submitAsyncImageGeneration: vi.fn(),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (params?.taskId) return `${key}:${params.taskId}`
        if (params?.page) return `${key}:${params.page}`
        return key
      },
    }),
  }
})

const pageLayoutStub = {
  template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>',
}
const appLayoutStub = { template: '<main><slot /></main>' }
const baseDialogStub = {
  props: ['show'],
  template: '<section v-if="show" class="base-dialog"><slot /><slot name="footer" /></section>',
}
const dataTableStub = {
  props: ['data'],
  template: `
    <div class="data-table">
      <div v-for="row in data" :key="row.task_id" class="data-row" :data-task-id="row.task_id">
        <slot name="cell-result" :row="row" />
        <slot name="cell-actions" :row="row" />
      </div>
      <slot v-if="data.length === 0" name="empty" />
    </div>
  `,
}
const iconStub = {
  props: ['name'],
  template: '<span class="icon" :data-icon="name" />',
}

const failedTask = {
  id: 'imgtask_failed',
  task_id: 'imgtask_failed',
  object: 'image.generation.task',
  status: 'failed',
  created_at: 1,
  expires_at: 2,
  error: { message: 'upstream failed' },
}

const completedTask = {
  ...failedTask,
  id: 'imgtask_completed',
  task_id: 'imgtask_completed',
  status: 'completed',
  error: undefined,
}

const otherKeyTask = {
  ...completedTask,
  id: 'imgtask_other_key',
  task_id: 'imgtask_other_key',
}

function mountView() {
  return mount(AsyncImageView, {
    attachTo: document.body,
    global: {
      stubs: {
        AppLayout: appLayoutStub,
        TablePageLayout: pageLayoutStub,
        DataTable: dataTableStub,
        BaseDialog: baseDialogStub,
        Icon: iconStub,
      },
    },
  })
}

function findButtonByText(text: string) {
  return Array.from(document.body.querySelectorAll('button')).find(button => button.textContent?.trim() === text)
}

describe('AsyncImageView task management', () => {
  beforeEach(() => {
    keysList.mockReset()
    deleteAsyncImageTask.mockReset()
    getAsyncImageObjectURL.mockReset()
    listAsyncImageTasks.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    keysList.mockResolvedValue({
      items: [{
        id: 9,
        user_id: 7,
        key: 'sk-selected-key',
        name: 'A very long selected API key name that must truncate',
        status: 'active',
        group: { platform: 'openai', allow_image_generation: true },
      }],
    })
    deleteAsyncImageTask.mockResolvedValue(undefined)
    getAsyncImageObjectURL.mockImplementation(async (_apiKey: string, objectID: string) => ({
      id: objectID,
      object: 'image.object',
      url: `https://fresh.test/${objectID}`,
      url_expires_at: Math.floor(Date.now() / 1000) + 3600,
    }))
    listAsyncImageTasks.mockResolvedValue({ object: 'list', data: [failedTask, completedTask], has_more: false })
  })

  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  it('uses matching shared Select triggers for both top filters', async () => {
    const wrapper = mountView()
    await flushPromises()

    const apiKeyFilter = wrapper.get('[data-testid="async-image-api-key-filter"]')
    const statusFilter = wrapper.get('[data-testid="async-image-status-filter"]')
    const apiKeyTrigger = apiKeyFilter.get('.select-trigger')
    const statusTrigger = statusFilter.get('.select-trigger')

    expect(apiKeyTrigger.element.tagName).toBe('BUTTON')
    expect(statusTrigger.element.tagName).toBe('BUTTON')
    expect(apiKeyTrigger.classes()).toEqual(statusTrigger.classes())
    expect(apiKeyTrigger.find('.select-value').classes()).toContain('select-value')
    expect(statusTrigger.find('.select-icon').exists()).toBe(true)
    expect(apiKeyTrigger.attributes('aria-label')).toBe('asyncImage.filters.apiKey')
    expect(statusTrigger.attributes('aria-label')).toBe('asyncImage.filters.allStatuses')
  })

  it('keeps exhausted keys selectable for history management but not task creation', async () => {
    keysList.mockResolvedValue({
      items: [{
        id: 11,
        user_id: 7,
        key: 'sk-exhausted-key',
        name: 'Exhausted image key',
        status: 'quota_exhausted',
        group: { platform: 'openai', allow_image_generation: true },
      }],
    })
    const wrapper = mountView()
    await flushPromises()

    expect(listAsyncImageTasks).toHaveBeenCalledWith('sk-exhausted-key', expect.objectContaining({ offset: 0 }))
    expect(wrapper.get('[data-testid="async-image-api-key-filter"] .select-trigger').text()).toContain('Exhausted image key')

    findButtonByText('asyncImage.actions.create')?.click()
    await flushPromises()
    expect(wrapper.text()).toContain('asyncImage.create.noKeys')
  })

  it('keeps a key selectable for history after it moves to another platform', async () => {
    keysList.mockResolvedValue({
      items: [{
        id: 12,
        user_id: 7,
        key: 'sk-reassigned-key',
        name: 'Reassigned key',
        status: 'active',
        group: { platform: 'anthropic', allow_image_generation: false },
      }],
    })
    const wrapper = mountView()
    await flushPromises()

    expect(listAsyncImageTasks).toHaveBeenCalledWith('sk-reassigned-key', expect.objectContaining({ offset: 0 }))
    expect(wrapper.get('[data-testid="async-image-api-key-filter"] .select-trigger').text()).toContain('Reassigned key')

    findButtonByText('asyncImage.actions.create')?.click()
    await flushPromises()
    expect(wrapper.text()).toContain('asyncImage.create.noKeys')
  })

  it('renews an expired object URL before rendering the task result', async () => {
    listAsyncImageTasks.mockResolvedValue({
      object: 'list',
      data: [{
        ...completedTask,
        image_url: 'https://expired.test/image.png',
        result: { data: [{ url: 'https://expired.test/image.png', object_id: 'imgobj_expired', url_expires_at: 1 }] },
      }],
      has_more: false,
    })

    const wrapper = mountView()
    await flushPromises()

    expect(getAsyncImageObjectURL).toHaveBeenCalledWith('sk-selected-key', 'imgobj_expired')
    expect(wrapper.get('img').attributes('src')).toBe('https://fresh.test/imgobj_expired')
    expect(wrapper.html()).not.toContain('https://expired.test/image.png')
  })

  it('uses a quota-exhausted key to renew its existing history', async () => {
    keysList.mockResolvedValue({
      items: [{
        id: 11,
        user_id: 7,
        key: 'sk-exhausted-key',
        name: 'Exhausted image key',
        status: 'quota_exhausted',
        group: { platform: 'openai', allow_image_generation: true },
      }],
    })
    listAsyncImageTasks.mockResolvedValue({
      object: 'list',
      data: [{
        ...completedTask,
        result: { data: [{ object_id: 'imgobj_exhausted' }] },
      }],
      has_more: false,
    })

    const wrapper = mountView()
    await flushPromises()

    expect(getAsyncImageObjectURL).toHaveBeenCalledWith('sk-exhausted-key', 'imgobj_exhausted')
    expect(wrapper.get('img').attributes('src')).toBe('https://fresh.test/imgobj_exhausted')
  })

  it('does not renew a signed URL that remains valid', async () => {
    const validURL = 'https://signed.test/still-valid.png'
    listAsyncImageTasks.mockResolvedValue({
      object: 'list',
      data: [{
        ...completedTask,
        image_url: validURL,
        result: { data: [{ url: validURL, object_id: 'imgobj_valid', url_expires_at: Math.floor(Date.now() / 1000) + 3600 }] },
      }],
      has_more: false,
    })

    const wrapper = mountView()
    await flushPromises()

    expect(getAsyncImageObjectURL).not.toHaveBeenCalled()
    expect(wrapper.get('img').attributes('src')).toBe(validURL)
  })

  it('renews a signed URL that will expire inside the safety window', async () => {
    const expiringURL = 'https://signed.test/expiring.png'
    listAsyncImageTasks.mockResolvedValue({
      object: 'list',
      data: [{
        ...completedTask,
        image_url: expiringURL,
        result: { data: [{ url: expiringURL, object_id: 'imgobj_expiring', url_expires_at: Math.floor(Date.now() / 1000) + 30 }] },
      }],
      has_more: false,
    })

    const wrapper = mountView()
    await flushPromises()

    expect(getAsyncImageObjectURL).toHaveBeenCalledWith('sk-selected-key', 'imgobj_expiring')
    expect(wrapper.get('img').attributes('src')).toBe('https://fresh.test/imgobj_expiring')
    expect(wrapper.html()).not.toContain(expiringURL)
  })

  it('does not render the stale URL when renewal fails', async () => {
    getAsyncImageObjectURL.mockRejectedValue(new Error('renewal failed'))
    listAsyncImageTasks.mockResolvedValue({
      object: 'list',
      data: [{
        ...completedTask,
        image_url: 'https://expired.test/broken.png',
        result: { data: [{ url: 'https://expired.test/broken.png', object_id: 'imgobj_broken', url_expires_at: 1 }] },
      }],
      has_more: false,
    })

    const wrapper = mountView()
    await flushPromises()

    expect(getAsyncImageObjectURL).toHaveBeenCalledOnce()
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.html()).not.toContain('https://expired.test/broken.png')
  })

  it('deduplicates renewal for repeated object IDs in a multi-image result', async () => {
    listAsyncImageTasks.mockResolvedValue({
      object: 'list',
      data: [{
        ...completedTask,
        result: {
          data: [
            { url: 'https://expired.test/first.png', object_id: 'imgobj_shared', url_expires_at: 1 },
            { url: 'https://expired.test/duplicate.png', object_id: 'imgobj_shared', url_expires_at: 1 },
            { url: 'https://expired.test/second.png', object_id: 'imgobj_second', url_expires_at: 1 },
          ],
        },
      }],
      has_more: false,
    })

    const wrapper = mountView()
    await flushPromises()

    expect(getAsyncImageObjectURL).toHaveBeenCalledTimes(2)
    expect(getAsyncImageObjectURL).toHaveBeenCalledWith('sk-selected-key', 'imgobj_shared')
    expect(getAsyncImageObjectURL).toHaveBeenCalledWith('sk-selected-key', 'imgobj_second')
    expect(wrapper.findAll('img').map(image => image.attributes('src'))).toEqual([
      'https://fresh.test/imgobj_shared',
      'https://fresh.test/imgobj_second',
    ])
  })

  it('refreshes an already-open detail dialog with the renewed URL', async () => {
    const initialURL = 'https://signed.test/initial.png'
    listAsyncImageTasks
      .mockResolvedValueOnce({
        object: 'list',
        data: [{
          ...completedTask,
          image_url: initialURL,
          result: { data: [{ url: initialURL, object_id: 'imgobj_detail', url_expires_at: Math.floor(Date.now() / 1000) + 3600 }] },
        }],
        has_more: false,
      })
      .mockResolvedValueOnce({
        object: 'list',
        data: [{
          ...completedTask,
          image_url: 'https://expired.test/detail.png',
          result: { data: [{ url: 'https://expired.test/detail.png', object_id: 'imgobj_detail', url_expires_at: 1 }] },
        }],
        has_more: false,
      })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="view-task-imgtask_completed"]').trigger('click')
    expect(wrapper.get('.base-dialog img').attributes('src')).toBe(initialURL)

    await wrapper.get('button[title="asyncImage.actions.refresh"]').trigger('click')
    await flushPromises()

    expect(getAsyncImageObjectURL).toHaveBeenCalledWith('sk-selected-key', 'imgobj_detail')
    expect(wrapper.get('.base-dialog img').attributes('src')).toBe('https://fresh.test/imgobj_detail')
    expect(wrapper.html()).not.toContain('https://expired.test/detail.png')
  })

  it('shows delete only for failed rows and supports cancel', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="delete-task-imgtask_failed"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="delete-task-imgtask_completed"]').exists()).toBe(false)

    await wrapper.get('[data-testid="delete-task-imgtask_failed"]').trigger('click')
    expect(document.body.textContent).toContain('asyncImage.delete.confirm:imgtask_failed')
    findButtonByText('common.cancel')?.click()
    await flushPromises()

    expect(deleteAsyncImageTask).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="delete-task-imgtask_failed"]').exists()).toBe(true)
  })

  it('uses the selected API key and shows pending state only on the deleted row', async () => {
    let finishDelete!: () => void
    deleteAsyncImageTask.mockReturnValue(new Promise<void>((resolve) => { finishDelete = resolve }))
    listAsyncImageTasks
      .mockResolvedValueOnce({ object: 'list', data: [failedTask, completedTask], has_more: false })
      .mockResolvedValue({ object: 'list', data: [completedTask], has_more: false })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="view-task-imgtask_failed"]').trigger('click')
    expect(wrapper.text()).toContain('asyncImage.detail.taskId')
    await wrapper.get('[data-testid="delete-task-imgtask_failed"]').trigger('click')
    findButtonByText('common.delete')?.click()
    await flushPromises()

    expect(deleteAsyncImageTask).toHaveBeenCalledWith('sk-selected-key', 'imgtask_failed')
    const deleteButton = wrapper.get('[data-testid="delete-task-imgtask_failed"]')
    expect(deleteButton.attributes('disabled')).toBeDefined()
    expect(deleteButton.get('[data-icon="refresh"]').classes()).toContain('animate-spin')

    finishDelete()
    await flushPromises()
    expect(showSuccess).toHaveBeenCalledWith('asyncImage.delete.success')
    expect(wrapper.find('[data-testid="delete-task-imgtask_failed"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('asyncImage.detail.taskId')
  })

  it('keeps the failed row and filters when deletion is rejected', async () => {
    deleteAsyncImageTask.mockRejectedValue(Object.assign(new Error('not allowed'), { code: 'IMAGE_TASK_DELETE_NOT_ALLOWED' }))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="delete-task-imgtask_failed"]').trigger('click')
    findButtonByText('common.delete')?.click()
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('asyncImage.errors.deleteNotAllowed')
    expect(wrapper.find('[data-testid="delete-task-imgtask_failed"]').exists()).toBe(true)
    expect(listAsyncImageTasks).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="async-image-status-filter"] .select-trigger').text()).toContain('asyncImage.filters.allStatuses')
  })

  it('falls back to the preceding page when deletion empties a later page', async () => {
    listAsyncImageTasks
      .mockResolvedValueOnce({ object: 'list', data: [completedTask], has_more: true })
      .mockResolvedValueOnce({ object: 'list', data: [failedTask], has_more: false })
      .mockResolvedValueOnce({ object: 'list', data: [], has_more: false })
      .mockResolvedValueOnce({ object: 'list', data: [completedTask], has_more: false })
    const wrapper = mountView()
    await flushPromises()

    findButtonByText('asyncImage.list.next')?.click()
    await flushPromises()
    expect(wrapper.find('[data-testid="delete-task-imgtask_failed"]').exists()).toBe(true)

    await wrapper.get('[data-testid="delete-task-imgtask_failed"]').trigger('click')
    findButtonByText('common.delete')?.click()
    await flushPromises()

    expect(listAsyncImageTasks).toHaveBeenCalledTimes(4)
    expect(listAsyncImageTasks.mock.calls.map(call => call[1]?.offset)).toEqual([0, 20, 20, 0])
    expect(wrapper.text()).toContain('asyncImage.list.page:1')
  })

  it('does not let a stale post-delete refresh overwrite a newly selected API key', async () => {
    let finishDeletedKeyRefresh!: (value: { object: string; data: typeof failedTask[]; has_more: boolean }) => void
    const deletedKeyRefresh = new Promise<{ object: string; data: typeof failedTask[]; has_more: boolean }>((resolve) => {
      finishDeletedKeyRefresh = resolve
    })
    keysList.mockResolvedValue({
      items: [
        {
          id: 9,
          user_id: 7,
          key: 'sk-selected-key',
          name: 'Primary image key',
          status: 'active',
          group: { platform: 'openai', allow_image_generation: true },
        },
        {
          id: 10,
          user_id: 7,
          key: 'sk-other-key',
          name: 'Secondary image key',
          status: 'active',
          group: { platform: 'openai', allow_image_generation: true },
        },
      ],
    })
    listAsyncImageTasks
      .mockResolvedValueOnce({ object: 'list', data: [failedTask], has_more: false })
      .mockReturnValueOnce(deletedKeyRefresh)
      .mockResolvedValueOnce({ object: 'list', data: [otherKeyTask], has_more: false })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="delete-task-imgtask_failed"]').trigger('click')
    findButtonByText('common.delete')?.click()
    await flushPromises()

    await wrapper.get('[data-testid="async-image-api-key-filter"] .select-trigger').trigger('click')
    const otherKeyOption = Array.from(document.body.querySelectorAll<HTMLElement>('[role="option"]'))
      .find(option => option.textContent?.includes('Secondary image key'))
    expect(otherKeyOption).toBeDefined()
    otherKeyOption?.click()
    await flushPromises()

    expect(listAsyncImageTasks).toHaveBeenLastCalledWith('sk-other-key', expect.objectContaining({ offset: 0 }))
    expect(wrapper.find('[data-testid="view-task-imgtask_other_key"]').exists()).toBe(true)

    finishDeletedKeyRefresh({ object: 'list', data: [failedTask], has_more: false })
    await flushPromises()
    expect(wrapper.find('[data-testid="view-task-imgtask_other_key"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="view-task-imgtask_failed"]').exists()).toBe(false)
  })
})

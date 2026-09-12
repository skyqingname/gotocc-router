import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AutoGroupOrderEditor from '../AutoGroupOrderEditor.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('AutoGroupOrderEditor', () => {
  const groups = [
    { id: 1, name: 'Primary', platform: 'openai' as const },
    { id: 2, name: 'Secondary', platform: 'openai' as const },
  ]

  it('moves a preferred group without mutating its input', async () => {
    const value = [1, 2]
    const wrapper = mount(AutoGroupOrderEditor, { props: { modelValue: value, groups } })
    await wrapper.find('[data-action="down"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([[2, 1]])
    expect(value).toEqual([1, 2])
  })

  it('removes only the chosen preference', async () => {
    const wrapper = mount(AutoGroupOrderEditor, { props: { modelValue: [1, 2], groups } })
    await wrapper.find('[data-action="remove"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([[2]])
  })
})

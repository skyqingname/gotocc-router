import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import RoutingPriorityPanel from '../RoutingPriorityPanel.vue'

const { getRoutingPriorities } = vi.hoisted(() => ({ getRoutingPriorities: vi.fn() }))
vi.mock('@/api/groups', () => ({ getRoutingPriorities }))
vi.mock('vue-i18n', async () => {
  const { default: zh } = await import('@/i18n/locales/zh/dashboard')
  return { useI18n: () => ({
    t: (key: string, values: Record<string, string | number> = {}) => {
      const name = key.replace('keys.routingPriority.', '') as keyof typeof zh.keys.routingPriority
      return (zh.keys.routingPriority[name] || key).replace(/\{(\w+)\}/g, (_, field: string) => String(values[field] ?? field))
    },
  }) }
})

const groups = [
  { id: 2, name: '常用分组', platform: 'openai' },
  { id: 1, name: '备用分组', platform: 'openai' },
]
const response = { default_source: 'group_sort', groups, model_rules: [] }
const render = () => mount(RoutingPriorityPanel, {
  props: { scope: 'personal' },
})

describe('RoutingPriorityPanel', () => {
  beforeEach(() => { getRoutingPriorities.mockReset() })

  it('shows the server order, source and explanation without default model rows', async () => {
    getRoutingPriorities.mockResolvedValue(response)
    const wrapper = render()
    expect(wrapper.text()).toContain('正在加载')
    await flushPromises()
    expect(getRoutingPriorities).toHaveBeenCalledWith('personal', expect.any(AbortSignal))
    const names = wrapper.findAll('[data-test="default-group"]')
    expect(names.map(row => row.text())).toEqual(['1常用分组', '2备用分组'])
    expect(wrapper.text()).toContain('按分组排序')
    expect(wrapper.text()).toContain('仅展示同一模型有多个可选分组时')
    expect(wrapper.find('details').exists()).toBe(false)
    wrapper.unmount()
  })

  it('explains independent models and their effective order in expandable details', async () => {
    getRoutingPriorities.mockResolvedValue({ ...response, default_source: 'administrator', model_rules: [
      { model: 'gpt-5', matched_rule: 'gpt-*', groups: [...groups].reverse() },
    ] })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain('管理员默认顺序')
    expect(wrapper.find('summary').text()).toContain('部分模型有独立规则')
    expect(wrapper.find('details').text()).toContain('gpt-5')
    expect(wrapper.find('details').text()).toContain('gpt-*')
    expect(wrapper.findAll('[data-test="model-group"]').map(row => row.text())).toEqual(['1备用分组', '2常用分组'])
    wrapper.unmount()
  })

  it('shows an explanation without a ranking when there are no competing groups', async () => {
    getRoutingPriorities.mockResolvedValue({ ...response, groups: [] })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain('暂无需要比较优先级的分组')
    expect(wrapper.find('ol').exists()).toBe(false)
    wrapper.unmount()
  })

  it('shows a retryable failure without claiming there are no competing groups', async () => {
    getRoutingPriorities.mockRejectedValueOnce(new Error('offline')).mockResolvedValue(response)
    const wrapper = render()
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('暂无需要比较')
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('[data-test="default-group"]')).toHaveLength(2)
    wrapper.unmount()
  })

  it('discards an old scope response and aborts on unmount', async () => {
    let finishOld: (value: typeof response) => void = () => {}
    getRoutingPriorities.mockReturnValueOnce(new Promise(resolve => { finishOld = resolve }))
      .mockResolvedValueOnce({ ...response, groups: [{ id: 3, name: '团队分组', platform: 'openai' }, groups[0]] })
    const wrapper = render()
    const oldSignal = getRoutingPriorities.mock.calls[0][1] as AbortSignal
    await wrapper.setProps({ scope: 'team' })
    await flushPromises()
    expect(oldSignal.aborted).toBe(true)
    finishOld(response)
    await flushPromises()
    expect(wrapper.text()).toContain('团队分组')
    expect(wrapper.text()).not.toContain('备用分组')
    const currentSignal = getRoutingPriorities.mock.calls[1][1] as AbortSignal
    wrapper.unmount()
    expect(currentSignal.aborted).toBe(true)
  })
})

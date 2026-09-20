// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import RateScheduleEditor from '../RateScheduleEditor.vue'
import type { RateScheduleConfig } from '@/utils/rate-schedule'

const initial = (): RateScheduleConfig => ({
  enabled: true, timezone: 'UTC',
  rules: [{ id: 'night', enabled: true, start: '22:00', end: '06:00', multiplier: 0.5 }]
})
const render = (locale = 'en', modelValue = initial()) => mount(RateScheduleEditor, {
  props: { modelValue, serverTimezone: 'UTC', baseMultiplier: 0.4, previewAt: new Date('2026-09-20T23:00:00Z') },
  global: { plugins: [createI18n({ legacy: false, locale })] }
})

describe('RateScheduleEditor', () => {
  it('previews the effective multiplier and localizes both languages', () => {
    for (const locale of ['en', 'zh']) {
      const wrapper = render(locale)
      try {
        expect(wrapper.get('[data-test="preview"]').text()).toContain('0.4 × 0.5 = 0.2')
        expect(wrapper.text()).toContain(locale === 'zh' ? '启用时段倍率' : 'Enable time-window rates')
        expect(wrapper.emitted('validity')?.[0]).toEqual([true])
      } finally { wrapper.unmount() }
    }
  })

  it('emits changes without mutating its input or erasing disabled rules', async () => {
    const input = initial()
    const wrapper = render('en', input)
    try {
      await wrapper.get('[data-test="schedule-enabled"]').setValue(false)
      const next = wrapper.emitted('update:modelValue')?.[0]?.[0] as RateScheduleConfig
      expect(next.enabled).toBe(false)
      expect(next.rules).toEqual(input.rules)
      expect(input.enabled).toBe(true)
      await wrapper.setProps({ modelValue: next })
      expect(wrapper.get('[data-test="preview"]').text()).toContain('0.4 × 1 = 0.4')
    } finally { wrapper.unmount() }
  })

  it('adds/removes rows and reports incomplete or overlapping rules as invalid', async () => {
    const wrapper = render()
    try {
      await wrapper.get('[data-test="add-rule"]').trigger('click')
      const next = wrapper.emitted('update:modelValue')?.[0]?.[0] as RateScheduleConfig
      expect(next.rules).toHaveLength(2)
      await wrapper.setProps({ modelValue: next })
      expect(wrapper.find('[data-test="error"]').exists()).toBe(true)
      const validity = wrapper.emitted('validity')
      expect(validity?.[validity.length - 1]).toEqual([false])
      const overlap = { ...next, rules: [next.rules[0], { ...next.rules[1], start: '23:00', end: '01:00' }] }
      await wrapper.setProps({ modelValue: overlap })
      expect(wrapper.get('[data-test="error"]').text()).toContain('overlap')
      await wrapper.get('[data-test="rule"]:last-child [data-test="remove-rule"]').trigger('click')
      const updates = wrapper.emitted('update:modelValue')
      const removed = updates?.[updates.length - 1]?.[0] as RateScheduleConfig
      expect(removed.rules).toHaveLength(1)
    } finally { wrapper.unmount() }
  })
})

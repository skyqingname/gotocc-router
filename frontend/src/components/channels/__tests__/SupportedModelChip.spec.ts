import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SupportedModelChip from '../SupportedModelChip.vue'
import type { UserSupportedModel } from '@/api/channels'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

function videoModel(): UserSupportedModel {
  return {
    name: 'seedance2.5',
    platform: 'openai',
    pricing: {
      billing_mode: 'video',
      input_price: null,
      output_price: null,
      cache_write_price: null,
      cache_read_price: null,
      image_input_price: null,
      image_output_price: null,
      per_request_price: 0.6,
      intervals: []
    }
  }
}

describe('SupportedModelChip', () => {
  it('展示视频按秒模式、每秒价格与每秒单位', () => {
    const wrapper = mount(SupportedModelChip, {
      props: { model: videoModel() },
      global: { stubs: { Teleport: true } }
    })

    const text = wrapper.text()
    expect(text).toContain('availableChannels.pricing.billingModeVideo')
    expect(text).toContain('availableChannels.pricing.perSecondPrice')
    expect(text).toContain('$0.6 availableChannels.pricing.unitPerSecond')
  })
})

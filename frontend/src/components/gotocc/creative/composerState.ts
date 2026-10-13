import { computed, ref, toRef, watch } from 'vue'
import { isGPTImage2 } from '@/utils/asyncImageSize'
import type { useStudio } from './useStudio'
import type { CreativeOperation } from './types'

export interface CreativeModelOption { model: string; group_name: string }
export const creativeOptionKey = (option: CreativeModelOption) => option.model

export function createComposerState(studio: ReturnType<typeof useStudio>) {
  const { form, config } = studio
  const model = toRef(form, 'model')
  const dimensions = computed<Record<string, Record<string, string>>>(() => isGPTImage2(form.model) ? config.gpt_image_2_dimensions : config.standard_dimensions)
  const imageSize = ref(config.default_resolution)
  const aspectRatio = ref(config.default_aspect_ratio)
  // The persisted pixel dimensions remain the request source of truth.
  function readPixelSelection() {
    const size = form.size === 'custom' ? `${form.customWidth}x${form.customHeight}` : form.size
    for (const [tier, ratios] of Object.entries(dimensions.value)) {
      for (const [ratio, pixels] of Object.entries(ratios)) {
        if (pixels === size) { imageSize.value = tier; aspectRatio.value = ratio; return }
      }
    }
    imageSize.value = 'custom'
    aspectRatio.value = ''
  }
  watch([model, () => form.size, () => form.customWidth, () => form.customHeight], readPixelSelection, { immediate: true, flush: 'sync' })
  function writePixels(tier: string, ratio: string) {
    const pixels = dimensions.value[tier][ratio]
    form.size = pixels
  }
  const sizeSelection = computed({
    get: () => imageSize.value,
    set: (value: string) => writePixels(value, aspectRatio.value || config.default_aspect_ratio),
  })
  const ratioSelection = computed({
    get: () => aspectRatio.value,
    set: (value: string) => writePixels(imageSize.value === 'custom' ? config.default_resolution : imageSize.value, value),
  })
  const models = computed(() => studio.models.value.map(name => ({ model: name, group_name: studio.selectedKey.value?.group?.name || studio.selectedKey.value?.name || '' })))
  return {
    models,
    selectedOption: computed(() => models.value.find(option => option.model === form.model) || null),
    selectedOptionKey: model,
    loadingModels: studio.loadingModels,
    keySelected: computed(() => !!studio.keyID.value),
    selectOption(value: string) { form.model = value; studio.onModelChanged() },
    prompt: toRef(form, 'prompt'),
    operation: toRef(form, 'operation'),
    operationOptions: computed<CreativeOperation[]>(() => ['generate', 'edit', ...(studio.canInpaint.value ? ['inpaint' as const] : [])]),
    imageSize: sizeSelection,
    aspectRatio: ratioSelection,
    imageSizeOptions: computed(() => form.model ? Object.keys(dimensions.value) : []),
    aspectRatioOptions: computed(() => form.model ? Object.keys(dimensions.value[config.default_resolution]) : []),
    quality: toRef(form, 'quality'),
    qualityOptions: computed(() => isGPTImage2(form.model) ? config.qualities : []),
    background: toRef(form, 'background'),
    backgroundOptions: computed(() => isGPTImage2(form.model) ? config.backgrounds : []),
    thinkingLevel: ref(''),
    thinkingLevelOptions: computed<string[]>(() => []),
    busy: studio.submitting,
    canGenerate: studio.ready,
    activityCount: studio.activityCount,
    error: studio.error,
  }
}

export type ComposerState = ReturnType<typeof createComposerState>

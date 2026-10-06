<template>
  <section class="space-y-4 border-t border-gray-200 pt-4 dark:border-dark-600">
    <div>
      <h3 class="font-medium">{{ tr('视频模型', 'Video models') }}</h3>
      <p class="mt-1 text-xs leading-5 text-gray-500">
        {{ tr('用户统一调用 POST /v1/videos。每个模型选一个上游协议，再按上游说明填写能力和价格：网关在扣费前按能力校验请求、补默认值，然后转换成上游需要的格式。地址与密钥来自绑定的账号。', 'Clients always call POST /v1/videos. Pick an upstream protocol per model and fill in its capabilities and price: the gateway checks requests against them before billing, fills defaults and converts to the upstream format. Base URL and keys come from bound accounts.') }}
      </p>
    </div>
    <p v-if="catalogError" role="alert" class="text-sm text-red-600">{{ catalogError }}</p>

    <div class="flex gap-2">
      <input v-model="newName" class="input flex-1" :placeholder="tr('对外模型名，例如 seedance-2.0-720p', 'Public model name, e.g. seedance-2.0-720p')" @keydown.enter.prevent="add" />
      <button type="button" class="btn btn-secondary" :disabled="!newName.trim() || !!modelValue[newName.trim()]" @click="add">{{ tr('添加模型', 'Add model') }}</button>
    </div>

    <article v-for="(config, name) in modelValue" :key="name" class="rounded-xl border border-gray-200 dark:border-dark-600" :class="config.enabled ? '' : 'opacity-70'">
      <header class="flex flex-wrap items-center gap-3 px-4 py-3">
        <input type="checkbox" :checked="config.enabled" :title="tr('启用', 'Enabled')" @change="patch(name, { enabled: checked($event) })" />
        <button type="button" class="min-w-0 flex-1 text-left" @click="expanded[name] = !expanded[name]">
          <span class="font-mono text-sm font-semibold text-gray-900 dark:text-white">{{ name }}</span>
          <span v-if="config.family" class="ml-2 rounded bg-primary-50 px-1.5 py-0.5 text-[11px] text-primary-700 dark:bg-primary-500/15 dark:text-primary-300">{{ config.family }}</span>
          <span class="mt-0.5 block truncate text-xs text-gray-500">{{ protocolName(config) }} · {{ capabilitySummary(config.capabilities) }} · {{ priceSummary(name) }}</span>
        </button>
        <span v-if="problems[name]" role="alert" class="text-xs text-red-600">{{ problems[name] }}</span>
        <button type="button" class="btn btn-ghost btn-sm" @click="duplicate(name)">{{ tr('复制', 'Duplicate') }}</button>
        <button type="button" class="btn btn-ghost btn-sm text-red-600" @click="remove(name)">{{ tr('删除', 'Remove') }}</button>
        <button type="button" class="text-xs text-primary-600" @click="expanded[name] = !expanded[name]">{{ expanded[name] ? tr('收起', 'Collapse') : tr('编辑', 'Edit') }}</button>
      </header>

      <div v-if="expanded[name]" class="space-y-5 border-t border-gray-100 px-4 py-4 dark:border-dark-700">
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="text-sm">{{ tr('对外模型名', 'Public model name') }}
            <input class="input mt-1 w-full font-mono" :value="name" @change="rename(name, text($event))" />
          </label>
          <label class="text-sm">{{ tr('上游模型名', 'Upstream model') }}
            <input class="input mt-1 w-full font-mono" :value="config.upstream_model" @input="patch(name, { upstream_model: text($event) })" />
            <span class="mt-1 block text-xs text-gray-500">{{ tr('上游按分辨率拆成多个模型时可写占位，例如 Artdance 2.0-{resolution}，按请求分辨率替换。', 'If the upstream splits models by resolution, use a placeholder such as Artdance 2.0-{resolution}.') }}</span>
          </label>
          <label class="text-sm">{{ tr('上游协议', 'Upstream protocol') }}
            <select class="input mt-1 w-full" :value="protocolValue(config)" @change="selectProtocol(name, text($event))">
              <option value="openai">{{ tr('原样转发 · OpenAI 兼容 /v1/videos', 'Pass through · OpenAI-compatible /v1/videos') }}</option>
              <option v-if="config.protocol === 'custom_json'" value="custom_json">{{ tr('自定义 JSON 字段映射（旧配置）', 'Custom JSON mapping (legacy)') }}</option>
              <optgroup v-for="[vendor, entries] in vendors" :key="vendor" :label="vendor">
                <option v-for="p in entries" :key="p.id" :value="p.id">{{ p.name }} · {{ p.definition.create.method }} {{ p.definition.create.path }}</option>
              </optgroup>
            </select>
            <span class="mt-1 block text-xs text-gray-500">{{ protocolHint(config) }}</span>
          </label>
          <label class="text-sm">{{ tr('系列', 'Series') }}
            <input class="input mt-1 w-full" :value="config.family" list="video-model-families" :placeholder="tr('例如 Seedance、Wan、MiniMax 海螺', 'e.g. Seedance, Wan, MiniMax')" @input="patch(name, { family: text($event) })" />
          </label>
          <label class="text-sm sm:col-span-2">{{ tr('模型说明（文档与模型广场展示）', 'Description (docs and model plaza)') }}
            <textarea class="input mt-1 min-h-16 w-full" :value="config.description" :placeholder="tr('例如：Seedance 2.0 · 720p · 4–15 秒 · 9 图 3 视频 3 音频 · 不卡真人', 'e.g. Seedance 2.0 · 720p · 4–15 s · 9 images, 3 videos, 3 audios')" @input="patch(name, { description: text($event) })" />
          </label>
        </div>

        <fieldset class="space-y-3 rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
          <legend class="sr-only">{{ tr('能力', 'Capabilities') }}</legend>
          <p class="text-sm font-medium">{{ tr('能力', 'Capabilities') }}
            <span v-if="!config.capabilities" class="ml-2 text-xs font-normal text-amber-600">{{ tr('尚未填写：网关暂不限制此模型的参数', 'Not set: requests are not restricted') }}</span>
          </p>

          <div class="flex flex-wrap items-center gap-3 text-sm">
            <span class="w-20 shrink-0 text-gray-600 dark:text-gray-300">{{ tr('时长', 'Duration') }}</span>
            <label class="flex items-center gap-1"><input type="radio" :checked="durationMode(name) === 'range'" @change="setDurationMode(name, 'range')" />{{ tr('范围', 'Range') }}</label>
            <label class="flex items-center gap-1"><input type="radio" :checked="durationMode(name) === 'fixed'" @change="setDurationMode(name, 'fixed')" />{{ tr('固定档位', 'Fixed values') }}</label>
            <template v-if="durationMode(name) === 'range'">
              <input type="number" min="1" class="input w-20" :value="caps(config).min_seconds" :placeholder="tr('最短', 'Min')" @input="patchCaps(name, { min_seconds: num($event) })" />
              <span>–</span>
              <input type="number" min="1" class="input w-20" :value="caps(config).max_seconds" :placeholder="tr('最长', 'Max')" @input="patchCaps(name, { max_seconds: num($event) })" />
              <span class="text-gray-500">{{ tr('秒', 's') }}</span>
            </template>
            <template v-else>
              <input class="input w-48" :value="fixedDrafts[name] ?? (caps(config).fixed_seconds ?? []).join(', ')" :placeholder="tr('例如 5, 10, 15', 'e.g. 5, 10, 15')" @input="setFixedSeconds(name, text($event))" />
              <span class="text-gray-500">{{ tr('秒', 's') }}</span>
            </template>
          </div>

          <div class="flex flex-wrap items-center gap-2 text-sm">
            <span class="w-20 shrink-0 text-gray-600 dark:text-gray-300">{{ tr('分辨率', 'Resolution') }}</span>
            <button v-for="value in resolutionOptions(config)" :key="value" type="button" class="chip" :class="caps(config).resolutions?.includes(value) ? 'chip-on' : ''" @click="toggleResolution(name, value)">{{ value }}</button>
            <input class="input w-28" :placeholder="tr('其他，回车添加', 'Other, Enter')" @keydown.enter.prevent="addResolution(name, $event)" />
          </div>

          <div class="flex flex-wrap items-center gap-2 text-sm">
            <span class="w-20 shrink-0 text-gray-600 dark:text-gray-300">{{ tr('画面比例', 'Aspect ratio') }}</span>
            <button v-for="value in ratioOptions(config)" :key="value" type="button" class="chip" :class="caps(config).aspect_ratios?.includes(value) ? 'chip-on' : ''" @click="toggleRatio(name, value)">{{ value }}</button>
            <span class="text-xs text-gray-500">{{ tr('不选则不限制；选中的第一个为默认值', 'None means unrestricted; the first selected is the default') }}</span>
          </div>

          <div class="grid gap-2 text-sm sm:grid-cols-2">
            <label v-for="item in referenceFields" :key="item.flag" class="flex items-center gap-2">
              <input type="checkbox" :checked="caps(config)[item.flag]" @change="patchCaps(name, capsField(item.flag, checked($event)))" />
              <span class="w-16">{{ tr(item.zh, item.en) }}</span>
              <span class="text-gray-500">{{ tr('最多', 'max') }}</span>
              <input type="number" min="1" class="input w-20" :disabled="!caps(config)[item.flag]" :value="caps(config)[item.max]" :placeholder="tr('不限', 'any')" @input="patchCaps(name, capsField(item.max, num($event)))" />
              <span class="text-gray-500">{{ tr(item.unit, '') }}</span>
            </label>
            <label class="flex items-center gap-2">
              <span class="w-[5.25rem] pl-6">{{ tr('素材合计', 'Total') }}</span>
              <span class="text-gray-500">{{ tr('最多', 'max') }}</span>
              <input type="number" min="1" class="input w-20" :value="caps(config).max_reference_total" :placeholder="tr('不限', 'any')" @input="patchCaps(name, { max_reference_total: num($event) })" />
              <span class="text-gray-500">{{ tr('个', '') }}</span>
            </label>
          </div>

          <label class="flex items-center gap-2 text-sm">
            <input type="checkbox" :checked="caps(config).audio_output" @change="patchCaps(name, { audio_output: checked($event) })" />
            {{ tr('支持生成音频（请求可传 generate_audio: true）', 'Can generate audio (generate_audio: true)') }}
          </label>
        </fieldset>

        <fieldset class="space-y-3 rounded-lg bg-primary-50/50 p-4 dark:bg-primary-950/20">
          <legend class="sr-only">{{ tr('价格', 'Price') }}</legend>
          <div class="flex flex-wrap items-center gap-3 text-sm">
            <span class="font-medium">{{ tr('价格（USD）', 'Price (USD)') }}</span>
            <label class="flex items-center gap-1"><input type="radio" :checked="priceMode(name) === 'video'" @change="setMode(name, 'video')" />{{ tr('按秒', 'Per second') }}</label>
            <label class="flex items-center gap-1"><input type="radio" :checked="priceMode(name) === 'per_request'" @change="setMode(name, 'per_request')" />{{ tr('按次', 'Per request') }}</label>
          </div>
          <div v-if="caps(config).resolutions?.length" class="flex flex-wrap gap-3">
            <label v-for="resolution in caps(config).resolutions" :key="resolution" class="text-xs text-gray-600 dark:text-gray-300">{{ resolution }}
              <input type="number" min="0" step="any" class="input mt-1 w-28" :value="tierValue(name, resolution)" @input="setTier(name, resolution, num($event) ?? null)" />
            </label>
          </div>
          <label v-else class="block text-xs text-gray-600 dark:text-gray-300">{{ tr('统一价', 'Price') }}
            <input type="number" min="0" step="any" class="input mt-1 w-32" :value="price(name).price" @input="setUniform(name, num($event) ?? null)" />
          </label>
          <p class="text-xs text-gray-500">{{ priceMode(name) === 'video' ? tr('按秒：单价 × 请求时长。', 'Per second: price × requested seconds.') : tr('按次：每个视频一个价格，与时长无关。', 'Per request: one price per video.') }} {{ tr('分组倍率照常生效；没有价格的分辨率请求会被拒绝。', 'Group multipliers apply; resolutions without a price are rejected.') }}</p>
        </fieldset>

        <details v-if="config.protocol === 'yingce'" class="text-sm">
          <summary class="cursor-pointer text-gray-600 dark:text-gray-300">{{ tr('协议扩展参数（少数协议需要，如工作流 ID、项目与地区）', 'Protocol options (only some protocols need them)') }}</summary>
          <textarea class="input mt-2 min-h-24 w-full font-mono text-xs" :value="jsonDrafts[name] ?? JSON.stringify(config.provider_options || {}, null, 2)" @input="setProviderOptions(name, text($event))" />
          <pre v-if="entry(config.provider_id)?.documentation" class="mt-2 max-h-60 overflow-auto whitespace-pre-wrap rounded bg-gray-50 p-3 text-xs dark:bg-dark-800">{{ entry(config.provider_id)!.documentation }}</pre>
        </details>
      </div>
    </article>
    <datalist id="video-model-families"><option v-for="family in COMMON_FAMILIES" :key="family" :value="family" /></datalist>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import type { BillingMode } from '@/api/admin/channels'
import type { PricingFormEntry } from './types'
import {
  COMMON_ASPECT_RATIOS, COMMON_FAMILIES, COMMON_RESOLUTIONS, RESOLUTION_PLACEHOLDER,
  capabilitySummary, newVideoCapabilities, newVideoModel, readVideoModelPrice, removeVideoModelPrice,
  renameVideoModelPrice, unrestrictedVideoCapabilities, writeVideoModelPrice,
  type VideoCapabilities, type VideoModelConfig, type VideoModelPrice, type VideoProtocolEntry,
} from './video-models'

const props = defineProps<{ modelValue: Record<string, VideoModelConfig>; pricing: PricingFormEntry[] }>()
const emit = defineEmits<{
  (event: 'update:modelValue', value: Record<string, VideoModelConfig>): void
  (event: 'update:pricing', value: PricingFormEntry[]): void
  (event: 'validity', value: boolean): void
}>()
const { locale } = useI18n()
const tr = (zh: string, en: string) => (locale.value.startsWith('zh') ? zh : en || zh)
const text = (event: Event) => (event.target as HTMLInputElement).value
const checked = (event: Event) => (event.target as HTMLInputElement).checked
const num = (event: Event) => (text(event) === '' ? undefined : Number(text(event)))

const catalog = ref<VideoProtocolEntry[]>([])
const catalogError = ref('')
onMounted(async () => {
  try {
    catalog.value = (await apiClient.get<{ protocols: VideoProtocolEntry[] }>('/admin/groups/video-protocols')).data.protocols
  } catch {
    catalogError.value = tr('读取上游协议目录失败，请重新打开分组编辑。', 'Failed to load upstream protocols; reopen the editor.')
  }
})
const entry = (id?: string) => catalog.value.find((p) => p.id === id)
const vendors = computed(() => {
  const groups = new Map<string, VideoProtocolEntry[]>()
  for (const p of catalog.value) groups.set(p.vendor || 'Other', [...(groups.get(p.vendor || 'Other') ?? []), p])
  return [...groups.entries()]
})
const protocolValue = (config: VideoModelConfig) => (config.protocol === 'yingce' ? config.provider_id ?? '' : config.protocol)
const protocolName = (config: VideoModelConfig) => {
  if (config.protocol === 'yingce') return entry(config.provider_id)?.name ?? config.provider_id ?? ''
  return config.protocol === 'openai' ? tr('原样转发', 'Pass through') : tr('自定义 JSON', 'Custom JSON')
}
const protocolHint = (config: VideoModelConfig) => {
  const p = entry(config.provider_id)
  if (config.protocol !== 'yingce' || !p) return tr('请求原样发给上游 /v1/videos，仅替换模型名并补默认值；上游字段与本站一致时使用。', 'Forwarded unchanged except the model name and defaults; use when the upstream speaks the same fields.')
  const secret = p.configuration.fields.some((field) => field.name === 'secretKey') ? tr('；此协议还需在 Video 账号填写 Secret Key', '; also needs a Secret Key on the Video account') : ''
  return `${tr('转换为', 'Converted to')} ${p.definition.create.method} ${p.definition.create.path} · ${p.definition.create.contentType || 'application/json'}${secret}`
}

const expanded = ref<Record<string, boolean>>({})
const newName = ref('')
const jsonDrafts = ref<Record<string, string>>({})
const jsonErrors = ref<Record<string, boolean>>({})
const fixedDrafts = ref<Record<string, string>>({})
const durationModes = ref<Record<string, 'range' | 'fixed'>>({})
const modeDrafts = ref<Record<string, BillingMode>>({})

function emitModels(models: Record<string, VideoModelConfig>) {
  emit('update:modelValue', models)
}
function patch(name: string, fields: Partial<VideoModelConfig>) {
  emitModels({ ...props.modelValue, [name]: { ...props.modelValue[name], ...fields } })
}
function protocolConfig(id: string, upstream: string): VideoModelConfig {
  const p = entry(id)
  return p
    ? { ...newVideoModel(upstream), ...JSON.parse(JSON.stringify(p.template)), upstream_model: upstream }
    : { ...newVideoModel(upstream), protocol: id as 'openai' | 'custom_json' }
}
function selectProtocol(name: string, id: string) {
  const current = props.modelValue[name]
  delete jsonDrafts.value[name]
  delete jsonErrors.value[name]
  patch(name, {
    ...protocolConfig(id, current.upstream_model),
    enabled: current.enabled, description: current.description, family: current.family, capabilities: current.capabilities,
  })
}
function add() {
  const name = newName.value.trim()
  if (!name || props.modelValue[name]) return
  const previous = Object.values(props.modelValue).slice(-1)[0]
  emitModels({ ...props.modelValue, [name]: { ...protocolConfig(previous ? protocolValue(previous) : 'openai', name), family: previous?.family, capabilities: newVideoCapabilities() } })
  expanded.value[name] = true
  newName.value = ''
}
function duplicate(name: string) {
  let copy = `${name}-copy`
  for (let index = 2; props.modelValue[copy]; index++) copy = `${name}-copy${index}`
  emitModels({ ...props.modelValue, [copy]: JSON.parse(JSON.stringify(props.modelValue[name])) })
  emit('update:pricing', writeVideoModelPrice(props.pricing, copy, price(name)))
  expanded.value[copy] = true
}
function remove(name: string) {
  const next = { ...props.modelValue }
  delete next[name]
  emitModels(next)
  emit('update:pricing', removeVideoModelPrice(props.pricing, name))
}
function rename(from: string, value: string) {
  const to = value.trim()
  if (!to || to === from || props.modelValue[to]) return
  emitModels(Object.fromEntries(Object.entries(props.modelValue).map(([key, config]) => [key === from ? to : key, config])))
  emit('update:pricing', renameVideoModelPrice(props.pricing, from, to))
  expanded.value[to] = true
}

const caps = (config: VideoModelConfig): VideoCapabilities => config.capabilities ?? unrestrictedVideoCapabilities()
const capsField = (key: keyof VideoCapabilities, value: unknown) => ({ [key]: value }) as Partial<VideoCapabilities>
function patchCaps(name: string, fields: Partial<VideoCapabilities>) {
  patch(name, { capabilities: { ...caps(props.modelValue[name]), ...fields } })
}
const durationMode = (name: string) => durationModes.value[name] ?? (caps(props.modelValue[name]).fixed_seconds?.length ? 'fixed' : 'range')
function setDurationMode(name: string, mode: 'range' | 'fixed') {
  durationModes.value[name] = mode
  delete fixedDrafts.value[name]
  patchCaps(name, mode === 'fixed' ? { min_seconds: undefined, max_seconds: undefined } : { fixed_seconds: undefined })
}
function setFixedSeconds(name: string, value: string) {
  fixedDrafts.value[name] = value
  const seconds = value.split(/[,，\s]+/).filter(Boolean).map(Number).filter((n) => Number.isInteger(n) && n > 0)
  patchCaps(name, { fixed_seconds: seconds.length ? seconds : undefined })
}
const resolutionOptions = (config: VideoModelConfig) => [...new Set([...COMMON_RESOLUTIONS, ...(caps(config).resolutions ?? [])])]
const ratioOptions = (config: VideoModelConfig) => [...new Set([...COMMON_ASPECT_RATIOS, ...(caps(config).aspect_ratios ?? [])])]
function toggleResolution(name: string, value: string) {
  const current = caps(props.modelValue[name]).resolutions ?? []
  const next = current.includes(value) ? current.filter((item) => item !== value) : [...current, value]
  patchCaps(name, { resolutions: next.length ? next : undefined })
  const p = price(name)
  if (current.includes(value) && value in p.tiers) {
    const tiers = { ...p.tiers }
    delete tiers[value]
    setPrice(name, { ...p, tiers })
  }
}
function addResolution(name: string, event: Event) {
  const value = text(event).trim()
  if (value && !(caps(props.modelValue[name]).resolutions ?? []).includes(value)) toggleResolution(name, value)
  ;(event.target as HTMLInputElement).value = ''
}
function toggleRatio(name: string, value: string) {
  const current = caps(props.modelValue[name]).aspect_ratios ?? []
  const next = current.includes(value) ? current.filter((item) => item !== value) : [...current, value]
  patchCaps(name, { aspect_ratios: next.length ? next : undefined })
}
const referenceFields = [
  { flag: 'reference_images', max: 'max_reference_images', zh: '参考图', en: 'Images', unit: '张' },
  { flag: 'reference_videos', max: 'max_reference_videos', zh: '参考视频', en: 'Videos', unit: '个' },
  { flag: 'reference_audios', max: 'max_reference_audios', zh: '参考音频', en: 'Audios', unit: '段' },
] as const

const price = (name: string): VideoModelPrice => readVideoModelPrice(props.pricing, name)
const hasPrice = (name: string) => props.pricing.some((item) => item.models.includes(name))
const priceMode = (name: string): BillingMode => (hasPrice(name) ? price(name).mode : modeDrafts.value[name] ?? 'video')
function setPrice(name: string, next: VideoModelPrice) {
  emit('update:pricing', writeVideoModelPrice(props.pricing, name, next))
}
function setMode(name: string, mode: BillingMode) {
  modeDrafts.value[name] = mode
  if (hasPrice(name)) setPrice(name, { ...price(name), mode })
}
const tierValue = (name: string, resolution: string) => price(name).tiers[resolution] ?? price(name).price
function setTier(name: string, resolution: string, value: number | null) {
  const p = price(name)
  const resolutions = caps(props.modelValue[name]).resolutions ?? []
  const tiers = Object.fromEntries(resolutions.map((item) => [item, item === resolution ? value : p.tiers[item] ?? p.price]))
  setPrice(name, { mode: priceMode(name), price: null, tiers })
}
function setUniform(name: string, value: number | null) {
  setPrice(name, { mode: priceMode(name), price: value, tiers: {} })
}
function priceSummary(name: string) {
  if (!hasPrice(name)) return tr('未定价', 'No price')
  const p = price(name)
  const unit = p.mode === 'video' ? tr('/秒', '/s') : tr('/次', '/req')
  const tiers = Object.entries(p.tiers).filter(([, value]) => value !== null)
  return tiers.length ? `${tiers.map(([label, value]) => `${label} $${value}`).join(' · ')} ${unit}` : `$${p.price} ${unit}`
}

function setProviderOptions(name: string, value: string) {
  jsonDrafts.value[name] = value
  try {
    const parsed = JSON.parse(value)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error('object required')
    jsonErrors.value[name] = false
    patch(name, { provider_options: parsed })
  } catch {
    jsonErrors.value[name] = true
  }
}

const problems = computed(() => {
  const out: Record<string, string> = {}
  for (const [name, config] of Object.entries(props.modelValue)) {
    const c = config.capabilities
    if (!config.upstream_model.trim()) out[name] = tr('请填写上游模型名', 'Upstream model is required')
    else if (config.upstream_model.includes(RESOLUTION_PLACEHOLDER) && !c?.resolutions?.length) out[name] = tr('上游模型名含 {resolution} 时需选择分辨率', 'Select resolutions for the {resolution} placeholder')
    else if (c?.min_seconds && c.max_seconds && c.min_seconds > c.max_seconds) out[name] = tr('最短时长不能大于最长时长', 'Min duration exceeds max')
    else if (jsonErrors.value[name]) out[name] = tr('协议扩展参数不是有效 JSON 对象', 'Protocol options must be a JSON object')
  }
  return out
})
watch(problems, (value) => emit('validity', Object.keys(value).length === 0), { immediate: true })
</script>

<style scoped>
.chip {
  @apply rounded-full border border-gray-300 bg-white px-3 py-1 text-xs text-gray-600 dark:border-dark-500 dark:bg-dark-900 dark:text-gray-300;
}
.chip-on {
  @apply border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-500/15 dark:text-primary-200;
}
</style>

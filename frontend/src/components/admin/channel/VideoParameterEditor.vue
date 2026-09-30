<template>
  <div class="space-y-4">
    <div class="rounded-xl bg-cyan-50 p-4 dark:bg-cyan-950/20">
      <p class="text-sm font-medium text-cyan-900 dark:text-cyan-100">{{ tr('当前分组 · 当前模型', 'This group · This model') }}</p>
      <p class="mt-1 text-xs leading-5 text-cyan-800 dark:text-cyan-200">{{ tr('选择模型支持的参数，设置允许值和默认值。请求未传入时使用默认值；关闭参数后该模型不接受该参数。', 'Choose supported parameters, allowed values and defaults. Defaults apply only when a value is omitted. Disabled parameters are not accepted.') }}</p>
      <div class="mt-3 flex flex-wrap gap-2">
        <button v-for="preset in presets.parameters" :key="preset.name" type="button" class="rounded-full border border-cyan-200 bg-white px-3 py-1.5 text-xs text-cyan-800 disabled:opacity-40 dark:border-cyan-800 dark:bg-dark-800 dark:text-cyan-200" :disabled="value.parameters.some(p => p.name === preset.name)" @click="add(preset)">+ {{ tr(preset.label, preset.label_en) }}</button>
      </div>
    </div>
    <div v-for="(parameter, index) in value.parameters" :key="index" class="rounded-xl border border-gray-200 p-4 dark:border-dark-600">
      <div class="mb-4 flex items-center justify-between gap-3">
        <label class="flex items-center gap-2 text-sm font-medium"><input type="checkbox" :checked="!parameter.disabled" @change="patch(index, 'disabled', !($event.target as HTMLInputElement).checked)" />{{ parameter.label || parameter.name || tr('自定义参数', 'Custom parameter') }}</label>
        <button type="button" class="text-xs text-red-600" @click="remove(index)">{{ tr('删除参数', 'Remove') }}</button>
      </div>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3" :class="parameter.disabled ? 'opacity-60' : ''">
        <label class="text-xs text-gray-600 dark:text-gray-300">{{ tr('参数名', 'Parameter name') }}<input class="input mt-1 w-full font-mono" :value="parameter.name" @input="rename(index, text($event))" /></label>
        <label class="text-xs text-gray-600 dark:text-gray-300">{{ tr('显示名称', 'Display label') }}<input class="input mt-1 w-full" :value="parameter.label" @input="patch(index, 'label', text($event))" /></label>
        <label class="text-xs text-gray-600 dark:text-gray-300">{{ tr('数据类型', 'Type') }}<select class="input mt-1 w-full" :value="parameter.type" @change="patch(index, 'type', text($event))"><option v-for="type in types" :key="type">{{ type }}</option></select></label>
        <label class="flex items-center gap-2 text-sm"><input type="checkbox" :checked="parameter.required" :disabled="parameter.disabled" @change="patch(index, 'required', ($event.target as HTMLInputElement).checked)" />{{ tr('必填', 'Required') }}</label>
        <label class="flex items-center gap-2 text-sm"><input type="checkbox" :checked="hasDefault(parameter.name)" :disabled="parameter.disabled || !parameter.name" @change="toggleDefault(parameter, ($event.target as HTMLInputElement).checked)" />{{ tr('设置默认值', 'Set default') }}</label>
        <label v-if="hasDefault(parameter.name)" class="text-xs text-gray-600 dark:text-gray-300">{{ tr('默认值', 'Default value') }}
          <select v-if="parameter.type === 'boolean'" class="input mt-1 w-full" :value="String(value.defaults[parameter.name])" :disabled="parameter.disabled" @change="setDefault(parameter, text($event))"><option value="true">true</option><option value="false">false</option></select>
          <input v-else class="input mt-1 w-full" :value="drafts[parameter.name] ?? displayDefault(parameter.name)" :disabled="parameter.disabled" @input="setDefault(parameter, text($event))" />
        </label>
        <label v-if="['string', 'integer', 'number'].includes(parameter.type)" class="text-xs text-gray-600 dark:text-gray-300 sm:col-span-2">{{ tr('允许值（逗号分隔，留空不限制枚举）', 'Allowed values (comma separated; empty means no enum restriction)') }}<input class="input mt-1 w-full" :value="(parameter.values ?? []).join(', ')" :disabled="parameter.disabled" @input="patch(index, 'values', text($event).split(',').map(v => v.trim()).filter(Boolean))" /></label>
        <template v-if="['integer', 'number'].includes(parameter.type)">
          <label v-for="field in rangeFields" :key="field.key" class="text-xs text-gray-600 dark:text-gray-300">{{ tr(field.zh, field.en) }}<input type="number" step="any" class="input mt-1 w-full" :value="parameter[field.key]" :disabled="parameter.disabled" @input="patch(index, field.key, text($event) === '' ? undefined : Number(text($event)))" /></label>
        </template>
        <label class="text-xs text-gray-600 dark:text-gray-300 sm:col-span-2">{{ tr('上游字段路径（自定义 JSON 协议）', 'Upstream field path (Custom JSON)') }}<input class="input mt-1 w-full font-mono" :value="value.request_fields[parameter.name]" :placeholder="parameter.name" :disabled="!parameter.name || value.protocol !== 'custom_json'" @input="mapField(parameter.name, text($event))" /></label>
      </div>
      <p v-if="errors[parameter.name]" role="alert" class="mt-2 text-xs text-red-600">{{ errors[parameter.name] }}</p>
    </div>
    <button type="button" class="btn btn-secondary" @click="add()">+ {{ tr('自定义参数', 'Custom parameter') }}</button>
    <p v-if="validationError" role="alert" class="text-sm text-red-600">{{ validationError }}</p>
  </div>
</template>
<script setup lang="ts">
import { computed, ref, watchEffect } from 'vue'
import { useI18n } from 'vue-i18n'
import presets from '../../../../../video-parameter-presets.json'
import type { VideoModelConfig, VideoParameter } from './video-models'
const props = defineProps<{ value: VideoModelConfig }>()
const emit = defineEmits<{ (e: 'update:value', value: VideoModelConfig): void; (e: 'validity', value: boolean): void }>()
const { locale } = useI18n()
const tr = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const types = ['string', 'integer', 'number', 'boolean', 'array', 'object']
const rangeFields = [{ key: 'min', zh: '最小值', en: 'Minimum' }, { key: 'max', zh: '最大值', en: 'Maximum' }, { key: 'step', zh: '步长', en: 'Step' }] as const
const drafts = ref<Record<string, string>>({})
const errors = ref<Record<string, string>>({})
const text = (event: Event) => (event.target as HTMLInputElement).value
const hasDefault = (name: string) => Object.prototype.hasOwnProperty.call(props.value.defaults, name)
const displayDefault = (name: string) => typeof props.value.defaults[name] === 'string' ? String(props.value.defaults[name]) : JSON.stringify(props.value.defaults[name])
function patch(index: number, key: keyof VideoParameter, value: unknown) { emit('update:value', { ...props.value, parameters: props.value.parameters.map((p, i) => i === index ? { ...p, [key]: value } : p) }) }
function add(preset?: { name: string; type: string; label: string }) { emit('update:value', { ...props.value, parameters: [...props.value.parameters, { name: preset?.name ?? '', label: preset?.label ?? '', type: preset?.type ?? 'string', required: false, values: [] }] }) }
function remove(index: number) {
  const name = props.value.parameters[index].name
  const defaults = { ...props.value.defaults }; delete defaults[name]
  const request_fields = { ...props.value.request_fields }; delete request_fields[name]
  delete errors.value[name]; delete drafts.value[name]
  emit('update:value', { ...props.value, defaults, request_fields, parameters: props.value.parameters.filter((_, i) => i !== index) })
}
function rename(index: number, name: string) {
  const previous = props.value.parameters[index].name
  const defaults = { ...props.value.defaults }; const request_fields = { ...props.value.request_fields }
  if (previous && previous !== name && !props.value.parameters.some((p, i) => i !== index && p.name === name)) {
    if (hasDefault(previous)) { defaults[name] = defaults[previous]; delete defaults[previous] }
    if (request_fields[previous]) { request_fields[name] = request_fields[previous]; delete request_fields[previous] }
    delete drafts.value[previous]; delete errors.value[previous]
  }
  emit('update:value', { ...props.value, defaults, request_fields, parameters: props.value.parameters.map((p, i) => i === index ? { ...p, name } : p) })
}
function toggleDefault(p: VideoParameter, enabled: boolean) {
  const defaults = { ...props.value.defaults }
  if (enabled) defaults[p.name] = p.type === 'boolean' ? false : p.type === 'array' ? [] : p.type === 'object' ? {} : ['integer', 'number'].includes(p.type) ? (p.min ?? 0) : ''
  else delete defaults[p.name]
  delete drafts.value[p.name]; delete errors.value[p.name]
  emit('update:value', { ...props.value, defaults })
}
function setDefault(p: VideoParameter, raw: string) {
  drafts.value[p.name] = raw
  try {
    const value = p.type === 'string' ? raw : JSON.parse(raw)
    const valid = p.type === 'string' || p.type === 'boolean' && typeof value === 'boolean' || p.type === 'number' && typeof value === 'number' || p.type === 'integer' && Number.isInteger(value) || p.type === 'array' && Array.isArray(value) || p.type === 'object' && value !== null && typeof value === 'object' && !Array.isArray(value)
    if (!valid) throw new Error()
    delete errors.value[p.name]
    emit('update:value', { ...props.value, defaults: { ...props.value.defaults, [p.name]: value } })
  } catch { errors.value[p.name] = tr('默认值与所选数据类型不符', 'Default does not match the selected type') }
}
function mapField(name: string, target: string) { const fields = { ...props.value.request_fields }; if (target.trim()) fields[name] = target.trim(); else delete fields[name]; emit('update:value', { ...props.value, request_fields: fields }) }
const validationError = computed(() => {
  const names = props.value.parameters.map(p => p.name.trim())
  if (names.some(name => !name) || new Set(names).size !== names.length) return tr('参数名不能为空或重复', 'Parameter names must be nonempty and unique')
  if (props.value.parameters.some(p => p.min !== undefined && p.max !== undefined && p.min > p.max || p.step !== undefined && p.step <= 0)) return tr('请检查最小值、最大值和步长', 'Check the minimum, maximum and step')
  return ''
})
watchEffect(() => emit('validity', !validationError.value && !Object.values(errors.value).some(Boolean)))
</script>

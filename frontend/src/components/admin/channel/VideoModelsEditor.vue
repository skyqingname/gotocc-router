<template>
  <section class="space-y-4 border-t border-gray-200 pt-4 dark:border-dark-600">
    <div>
      <h3 class="font-medium">{{ t('title') }}</h3>
      <p class="mt-1 text-xs text-gray-500">{{ t('hint') }}</p>
    </div>
    <div class="flex gap-2">
      <input v-model="newName" class="input flex-1" :placeholder="t('publicModel')" />
      <button type="button" class="btn btn-secondary" :disabled="!newName.trim() || !!modelValue[newName.trim()]" @click="add">{{ t('add') }}</button>
    </div>
    <details v-for="(config, name) in modelValue" :key="name" class="rounded-lg border border-gray-200 p-4 dark:border-dark-600" open>
      <summary class="cursor-pointer font-medium">{{ name }} · {{ config.protocol === 'openai' ? 'OpenAI compatible' : t('custom') }}</summary>
      <div class="mt-4 space-y-4">
        <div class="flex items-center justify-between">
          <label class="flex items-center gap-2 text-sm"><input type="checkbox" :checked="config.enabled" @change="set(name, 'enabled', ($event.target as HTMLInputElement).checked)" />{{ t('enabled') }}</label>
          <button type="button" class="btn btn-ghost text-red-600" @click="remove(name)">{{ t('remove') }}</button>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="text-sm">{{ t('upstreamModel') }}<input class="input mt-1 w-full" :value="config.upstream_model" required @input="set(name, 'upstream_model', text($event))" /></label>
          <label class="text-sm">{{ t('protocol') }}<select class="input mt-1 w-full" :value="config.protocol" @change="set(name, 'protocol', text($event))"><option value="openai">OpenAI · JSON / multipart</option><option value="custom_json">{{ t('custom') }}</option></select></label>
          <label v-for="field in paths" :key="field" class="text-sm">{{ t(field) }}<input class="input mt-1 w-full font-mono text-xs" :value="config[field]" @input="set(name, field, text($event))" /></label>
        </div>
        <p class="text-xs text-gray-500">{{ t('pathsHint', { task_id: '{task_id}' }) }}</p>
        <div class="space-y-2">
          <div class="flex justify-between"><h4 class="text-sm font-medium">{{ t('parameters') }}</h4><button type="button" class="btn btn-secondary btn-sm" @click="set(name, 'parameters', [...config.parameters, { name: '', type: 'string', required: false, values: [] }])">{{ t('addParameter') }}</button></div>
          <div v-for="(parameter, index) in config.parameters" :key="index" class="grid grid-cols-2 gap-2 rounded-lg bg-gray-50 p-3 text-xs dark:bg-dark-800 sm:grid-cols-3">
            <input class="input" :placeholder="t('parameterName')" :value="parameter.name" @input="setParameter(name, index, 'name', text($event))" />
            <select class="input" :value="parameter.type" @change="setParameter(name, index, 'type', text($event))"><option v-for="type in ['string', 'integer', 'number', 'boolean', 'array', 'object']" :key="type">{{ type }}</option></select>
            <label class="flex items-center gap-2"><input type="checkbox" :checked="parameter.required" @change="setParameter(name, index, 'required', ($event.target as HTMLInputElement).checked)" />{{ t('required') }}</label>
            <input class="input col-span-2" :placeholder="t('values')" :value="parameter.values.join(', ')" @input="setParameter(name, index, 'values', text($event).split(',').map(value => value.trim()).filter(Boolean))" />
            <button type="button" class="text-red-600" @click="set(name, 'parameters', config.parameters.filter((_, row) => row !== index))">{{ t('remove') }}</button>
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <label v-for="field in jsonFields" :key="field" class="text-sm">{{ t(field) }}<textarea class="input mt-1 min-h-28 w-full font-mono text-xs" :value="jsonDrafts[`${name}:${field}`] ?? JSON.stringify(config[field], null, 2)" @input="setJSON(name, field, text($event))" /></label>
        </div>
      </div>
    </details>
    <p v-if="Object.values(jsonErrors).some(Boolean)" role="alert" class="text-sm text-red-600">{{ t('invalidJson') }}</p>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { newVideoModel, type VideoModelConfig } from './video-models'

const props = defineProps<{ modelValue: Record<string, VideoModelConfig> }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: Record<string, VideoModelConfig>): void; (event: 'validity', value: boolean): void }>()
const newName = ref('')
const jsonDrafts = ref<Record<string, string>>({})
const jsonErrors = ref<Record<string, boolean>>({})
const paths = ['create_status', 'create_path', 'status_path', 'content_path', 'id_field', 'status_field', 'video_url_field'] as const
const jsonFields = ['defaults', 'request_fields', 'headers', 'statuses'] as const
const en = {
  title: 'Video models and protocols', hint: 'Configure each channel model here; set its price below. Bind OpenAI-compatible API key accounts to the Video group for the upstream URL and credentials.',
  publicModel: 'Model name exposed to clients', add: 'Add model', custom: 'Custom JSON', enabled: 'Enabled', remove: 'Remove', upstreamModel: 'Upstream model', protocol: 'Protocol',
  create_status: 'Status when create response omits it', create_path: 'Create path', status_path: 'Status path', content_path: 'Content path', id_field: 'Task ID response field', status_field: 'Status response field', video_url_field: 'Video URL response field',
  pathsHint: 'Paths start at the API origin; include /v1 when needed. Use {task_id} in task paths. Leave content path empty to read the video URL from the status response. Dotted fields address nested JSON.',
  parameters: 'Model parameters', addParameter: 'Add parameter', parameterName: 'Name, e.g. seconds', required: 'Required', values: 'Allowed values, separated by commas',
  defaults: 'Default parameters (JSON)', request_fields: 'Request field mapping (JSON)', headers: 'Provider headers (JSON; credentials belong to the account)', statuses: 'Provider status mapping (JSON)', invalidJson: 'Fix the JSON fields before saving.',
}
const zh: typeof en = {
  title: 'Video 模型与协议', hint: '逐模型配置调用方式，并在下方配置价格。Video 分组绑定 OpenAI 兼容 API Key 账号，使用账号中的上游地址和凭据。',
  publicModel: '提供给用户调用的模型名', add: '添加模型', custom: '自定义 JSON 协议', enabled: '启用', remove: '删除', upstreamModel: '上游模型名', protocol: '调用协议',
  create_status: '创建响应无状态时的明确状态', create_path: '创建路径', status_path: '查询路径', content_path: '视频内容路径', id_field: '任务 ID 字段', status_field: '任务状态字段', video_url_field: '视频 URL 字段',
  pathsHint: '路径从上游站点根地址开始，需要时包含 /v1。任务路径使用 {task_id}；内容路径留空时，从查询结果读取视频 URL。嵌套字段使用点号。',
  parameters: '模型参数', addParameter: '添加参数', parameterName: '参数名，例如 seconds', required: '必填', values: '允许值，用英文逗号分隔',
  defaults: '默认参数（JSON）', request_fields: '请求字段映射（JSON）', headers: '上游请求头（JSON；凭据在账号中配置）', statuses: '上游状态映射（JSON）', invalidJson: '请修正 JSON 格式后保存。',
}
const { t } = useI18n({ useScope: 'local', messages: { en, zh } })
const text = (event: Event) => (event.target as HTMLInputElement).value
function add() { const name = newName.value.trim(); emit('update:modelValue', { ...props.modelValue, [name]: newVideoModel(name) }); newName.value = '' }
function set(name: string, field: keyof VideoModelConfig, value: unknown) { emit('update:modelValue', { ...props.modelValue, [name]: { ...props.modelValue[name], [field]: value } }) }
function remove(name: string) { const next = { ...props.modelValue }; delete next[name]; for (const key of Object.keys(jsonErrors.value)) if (key.startsWith(`${name}:`)) { delete jsonErrors.value[key]; delete jsonDrafts.value[key] }; emit('update:modelValue', next); emit('validity', !Object.values(jsonErrors.value).some(Boolean)) }
function setParameter(name: string, index: number, field: string, value: unknown) { set(name, 'parameters', props.modelValue[name].parameters.map((parameter, row) => row === index ? { ...parameter, [field]: value } : parameter)) }
function setJSON(name: string, field: typeof jsonFields[number], value: string) {
  const key = `${name}:${field}`; jsonDrafts.value[key] = value
  try { const parsed = JSON.parse(value); if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error('object required'); set(name, field, parsed); jsonErrors.value[key] = false }
  catch { jsonErrors.value[key] = true }
  emit('validity', !Object.values(jsonErrors.value).some(Boolean))
}
</script>

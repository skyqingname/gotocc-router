<template>
  <section class="space-y-4 border-t border-gray-200 pt-4 dark:border-dark-600">
    <div>
      <h3 class="font-medium">{{ t('title') }}</h3>
      <p class="mt-1 text-xs text-gray-500">{{ t('hint') }}</p>
    </div>
    <p v-if="catalogError" role="alert" class="text-sm text-red-600">{{ catalogError }}</p>
    <div class="rounded-xl bg-primary-50 p-4 dark:bg-primary-950/20">
      <p class="mb-3 text-sm font-medium">影策视频协议 · {{ catalog.length }} 种</p>
      <input v-model="protocolSearch" class="input mb-3 w-full" placeholder="搜索厂商、协议名或请求路径" />
      <label class="block text-sm">新增模型使用的接口<select v-model="newProtocol" class="input mt-1 w-full"><option v-for="p in filteredCatalog" :key="p.id" :value="p.id">{{ protocolLabel(p) }}</option><option value="openai">自定义 OpenAI JSON / multipart</option><option value="custom_json">自定义 JSON 字段映射</option></select></label>
    </div>
    <div class="flex gap-2">
      <input v-model="newName" class="input flex-1" :placeholder="t('publicModel')" />
      <button type="button" class="btn btn-secondary" :disabled="!newName.trim() || !!modelValue[newName.trim()]" @click="add">{{ t('add') }}</button>
    </div>
    <details v-for="(config, name) in modelValue" :key="name" class="rounded-lg border border-gray-200 p-4 dark:border-dark-600" open>
      <summary class="cursor-pointer font-medium">{{ name }} · {{ config.protocol === 'yingce' ? entry(config.provider_id)?.name || config.provider_id : config.create_path }}</summary>
      <div class="mt-4 space-y-4">
        <div class="flex items-center justify-between">
          <label class="flex items-center gap-2 text-sm"><input type="checkbox" :checked="config.enabled" @change="set(name, 'enabled', ($event.target as HTMLInputElement).checked)" />{{ t('enabled') }}</label>
          <button type="button" class="btn btn-ghost text-red-600" @click="remove(name)">{{ t('remove') }}</button>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="text-sm">{{ t('upstreamModel') }}<input class="input mt-1 w-full" :value="config.upstream_model" required @input="set(name, 'upstream_model', text($event))" /></label>
          <label class="text-sm">上游接口协议<select class="input mt-1 w-full" :value="config.protocol === 'yingce' ? config.provider_id : config.protocol" @change="selectProtocol(name, text($event))"><option v-for="p in catalog" :key="p.id" :value="p.id">{{ protocolLabel(p) }}</option><option value="openai">自定义 OpenAI JSON / multipart</option><option value="custom_json">自定义 JSON 字段映射</option></select></label>
          <template v-if="config.protocol !== 'yingce'"><label v-for="field in paths" :key="field" class="text-sm">{{ t(field) }}<input class="input mt-1 w-full font-mono text-xs" :value="config[field]" @input="set(name, field, text($event))" /></label></template>
        </div>
        <p v-if="config.protocol !== 'yingce'" class="text-xs text-gray-500">{{ t('pathsHint', { task_id: '{task_id}' }) }}</p>
        <div v-if="config.protocol === 'yingce' && entry(config.provider_id)" class="space-y-3 rounded-lg bg-gray-50 p-4 text-sm dark:bg-dark-800">
          <p class="font-mono">{{ entry(config.provider_id)!.definition.create.method }} {{ entry(config.provider_id)!.definition.create.path }}</p>
          <p v-if="entry(config.provider_id)!.definition.poll" class="font-mono">查询：{{ entry(config.provider_id)!.definition.poll!.method }} {{ entry(config.provider_id)!.definition.poll!.path }}</p>
          <p>请求类型：{{ entry(config.provider_id)!.definition.create.contentType || 'application/json' }} · 鉴权：{{ entry(config.provider_id)!.definition.auth.type }}</p>
          <p>切换协议会重新载入对应参数；时长、分辨率等可选值按实际上游模型填写。</p>
          <p v-if="entry(config.provider_id)!.configuration.fields.some(f => f.name === 'secretKey')">此协议还需要 Secret Key，请在绑定的 Video 账号中配置。</p>
          <details><summary class="cursor-pointer font-medium">全部参数与接口定义（高级）</summary><p class="my-2 text-xs text-gray-500">默认继承该协议。上游有差异时可修改本模型的 create、poll、result、auth 和 response；凭据仍在账号中填写。</p><textarea class="input min-h-80 w-full font-mono text-xs" :value="jsonDrafts[`${name}:provider_definition`] ?? JSON.stringify(config.provider_definition || entry(config.provider_id)!.definition, null, 2)" @input="setJSON(name, 'provider_definition', text($event))" /></details>
          <details><summary class="cursor-pointer font-medium">协议说明</summary><pre class="mt-3 max-h-80 overflow-auto whitespace-pre-wrap text-xs">{{ entry(config.provider_id)!.documentation }}</pre></details>
          <label class="block">协议扩展参数（JSON，按 providerOptions 中的协议 ID 分组）<textarea class="input mt-1 min-h-28 w-full font-mono text-xs" :value="jsonDrafts[`${name}:provider_options`] ?? JSON.stringify(config.provider_options || {}, null, 2)" @input="setJSON(name, 'provider_options', text($event))" /></label>
        </div>
        <VideoParameterEditor :value="config" @update:value="replaceModel(name, $event)" @validity="setParameterValidity(name, $event)" />
        <details class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <summary class="cursor-pointer text-sm">{{ t('advanced') }}</summary>
        <div class="grid gap-3 sm:grid-cols-2">
          <label v-for="field in jsonFields" :key="field" class="text-sm">{{ t(field) }}<textarea class="input mt-1 min-h-28 w-full font-mono text-xs" :value="jsonDrafts[`${name}:${field}`] ?? JSON.stringify(config[field], null, 2)" @input="setJSON(name, field, text($event))" /></label>
        </div>
        </details>
      </div>
    </details>
    <p v-if="Object.values(jsonErrors).some(Boolean)" role="alert" class="text-sm text-red-600">{{ t('invalidJson') }}</p>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { apiClient } from '@/api/client'
import VideoParameterEditor from './VideoParameterEditor.vue'
import { useI18n } from 'vue-i18n'
import { newVideoModel, type VideoModelConfig, type VideoProtocolEntry } from './video-models'

const props = defineProps<{ modelValue: Record<string, VideoModelConfig> }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: Record<string, VideoModelConfig>): void; (event: 'validity', value: boolean): void }>()
const newName = ref('')
const catalog = ref<VideoProtocolEntry[]>([])
const catalogError = ref('')
const protocolSearch = ref('')
const newProtocol = ref('newapi')
const filteredCatalog = computed(() => catalog.value.filter(p => `${p.name} ${p.id} ${p.vendor} ${p.definition.create.path}`.toLowerCase().includes(protocolSearch.value.toLowerCase())))
onMounted(async () => { try { catalog.value = (await apiClient.get<{protocols:VideoProtocolEntry[]}>('/admin/groups/video-protocols')).data.protocols } catch { catalogError.value = '读取视频协议目录失败，请重新打开分组编辑。' } })
const entry = (id?: string) => catalog.value.find(p => p.id === id)
const protocolLabel = (p: VideoProtocolEntry) => `${p.name} · ${p.definition.create.method} ${p.definition.create.path}`
function protocolConfig(id:string, name:string):VideoModelConfig { const p=entry(id); return p ? {...newVideoModel(name),...JSON.parse(JSON.stringify(p.template)),upstream_model:name} : {...newVideoModel(name),protocol:id as 'openai'|'custom_json'} }
function selectProtocol(name:string,id:string) { const current=props.modelValue[name]; for(const key of Object.keys(jsonErrors.value)) if(key.startsWith(`${name}:`)){delete jsonErrors.value[key];delete jsonDrafts.value[key]}; replaceModel(name,{...protocolConfig(id,current.upstream_model),enabled:current.enabled}) }

const jsonDrafts = ref<Record<string, string>>({})
const jsonErrors = ref<Record<string, boolean>>({})
const parameterValidity = ref<Record<string, boolean>>({})
const paths = ['create_status', 'create_path', 'status_path', 'content_path', 'id_field', 'status_field', 'video_url_field'] as const
const jsonFields = ['defaults', 'request_fields', 'headers', 'statuses'] as const
const en = {
  advanced: 'Advanced JSON configuration', title: 'Video models and protocols', hint: 'Select the upstream protocol for each group model. Protocol templates inherit the full Yingce video request and response contract. Credentials come from bound accounts.',
  publicModel: 'Model name exposed to clients', add: 'Add model', custom: 'Custom JSON', enabled: 'Enabled', remove: 'Remove', upstreamModel: 'Upstream model', protocol: 'Protocol',
  create_status: 'Status when create response omits it', create_path: 'Create path', status_path: 'Status path', content_path: 'Content path', id_field: 'Task ID response field', status_field: 'Status response field', video_url_field: 'Video URL response field',
  pathsHint: 'Paths start at the API origin; include /v1 when needed. Use {task_id} in task paths. Leave content path empty to read the video URL from the status response. Dotted fields address nested JSON.',
  parameters: 'Model parameters', addParameter: 'Add parameter', parameterName: 'Name, e.g. seconds', required: 'Required', values: 'Allowed values, separated by commas',
  defaults: 'Default parameters (JSON)', request_fields: 'Request field mapping (JSON)', headers: 'Provider headers (JSON; credentials belong to the account)', statuses: 'Provider status mapping (JSON)', invalidJson: 'Fix the JSON fields before saving.',
}
const zh: typeof en = {
  advanced: '高级 JSON 配置', title: 'Video 模型与协议', hint: '在此分组为每个模型选择上游接口，继承影策后端的请求参数、查询和结果解析规则。地址与凭据由绑定账号提供。',
  publicModel: '提供给用户调用的模型名', add: '添加模型', custom: '自定义 JSON 协议', enabled: '启用', remove: '删除', upstreamModel: '上游模型名', protocol: '调用协议',
  create_status: '创建响应无状态时的明确状态', create_path: '创建路径', status_path: '查询路径', content_path: '视频内容路径', id_field: '任务 ID 字段', status_field: '任务状态字段', video_url_field: '视频 URL 字段',
  pathsHint: '路径从上游站点根地址开始，需要时包含 /v1。任务路径使用 {task_id}；内容路径留空时，从查询结果读取视频 URL。嵌套字段使用点号。',
  parameters: '模型参数', addParameter: '添加参数', parameterName: '参数名，例如 seconds', required: '必填', values: '允许值，用英文逗号分隔',
  defaults: '默认参数（JSON）', request_fields: '请求字段映射（JSON）', headers: '上游请求头（JSON；凭据在账号中配置）', statuses: '上游状态映射（JSON）', invalidJson: '请修正 JSON 格式后保存。',
}
const { t } = useI18n({ useScope: 'local', messages: { en, zh } })
const text = (event: Event) => (event.target as HTMLInputElement).value
function add() { const name = newName.value.trim(); emit('update:modelValue', { ...props.modelValue, [name]: protocolConfig(newProtocol.value,name) }); newName.value = '' }
function set(name: string, field: keyof VideoModelConfig, value: unknown) { emit('update:modelValue', { ...props.modelValue, [name]: { ...props.modelValue[name], [field]: value } }) }
function remove(name: string) { delete parameterValidity.value[name]; const next = { ...props.modelValue }; delete next[name]; for (const key of Object.keys(jsonErrors.value)) if (key.startsWith(`${name}:`)) { delete jsonErrors.value[key]; delete jsonDrafts.value[key] }; emit('update:modelValue', next); emitValidity() }
function emitValidity() { emit('validity', !Object.values(jsonErrors.value).some(Boolean) && Object.values(parameterValidity.value).every(Boolean)) }
function setParameterValidity(name: string, valid: boolean) { parameterValidity.value[name] = valid; emitValidity() }
function replaceModel(name: string, config: VideoModelConfig) { for (const field of ['defaults', 'request_fields'] as const) { delete jsonDrafts.value[`${name}:${field}`]; delete jsonErrors.value[`${name}:${field}`] }; emit('update:modelValue', { ...props.modelValue, [name]: config }); emitValidity() }
function setJSON(name: string, field: typeof jsonFields[number] | 'provider_options' | 'provider_definition', value: string) {
  const key = `${name}:${field}`; jsonDrafts.value[key] = value
  try { const parsed = JSON.parse(value); if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error('object required'); set(name, field, parsed); jsonErrors.value[key] = false }
  catch { jsonErrors.value[key] = true }
  emitValidity()
}
</script>

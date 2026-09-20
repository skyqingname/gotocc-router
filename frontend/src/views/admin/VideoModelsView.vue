<template>
  <AppLayout>
    <main class="mx-auto max-w-6xl space-y-5 p-6">
      <header class="flex flex-wrap items-center justify-between gap-3">
        <h1 class="text-xl font-semibold">{{ t('title') }}</h1>
        <RouterLink to="/admin/groups" class="btn btn-secondary">{{ t('back') }}</RouterLink>
      </header>
      <p class="rounded border border-amber-300 p-3 text-sm text-amber-700">{{ t('notActive') }}</p>
      <div class="flex items-center gap-3">
        <label class="flex-1">{{ t('group') }}
          <select v-model.number="selected" class="input w-full" :disabled="busy">
            <option :value="0" disabled>{{ t('choose') }}</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">#{{ group.id }} · {{ group.name }}</option>
          </select>
        </label>
        <button type="button" class="btn btn-secondary" :disabled="!selected || busy" @click="load">{{ t('load') }}</button>
      </div>
      <p v-if="error" role="alert" class="text-red-600">{{ error }}</p>
      <p v-if="notice" role="status" class="text-emerald-700">{{ notice }}</p>
      <fieldset v-if="view" :disabled="busy" class="space-y-5">
        <p class="text-sm">{{ t('editing') }} #{{ view.group_id }} · {{ t('version') }} {{ view.version }}</p>
        <section v-for="(binding, index) in bindings" :key="binding.id" class="space-y-4 rounded-lg border border-gray-300 p-4 dark:border-gray-700">
          <div class="flex items-center justify-between">
            <label><input v-model="binding.enabled" type="checkbox" /> {{ t('enabled') }}</label>
            <button type="button" class="btn btn-secondary" @click="bindings.splice(index, 1)">{{ t('remove') }}</button>
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <label>{{ t('name') }}<input v-model="binding.name" class="input w-full" maxlength="200" /></label>
            <label>{{ t('account') }}<input v-model.number="binding.account_id" type="number" min="1" class="input w-full" /></label>
            <label>{{ t('publicModel') }}<input v-model="binding.public_model" class="input w-full" maxlength="128" /></label>
            <label>{{ t('upstreamModel') }}<input v-model="binding.upstream_model" class="input w-full" maxlength="128" /></label>
            <label>{{ t('protocol') }}
              <select :value="binding.protocol" class="input w-full" @change="changeProtocol(binding, $event)">
                <option value="openai_video_json">OpenAI-compatible JSON /videos</option>
                <option value="xai_video">xAI JSON /videos/generations</option>
              </select>
            </label>
          </div>
          <p class="text-xs text-gray-500">{{ t('credentials') }}</p>
          <h2 class="font-medium">{{ t('parameters') }}</h2>
          <div v-for="(parameter, name) in binding.parameters" :key="name" class="flex flex-wrap items-center gap-3 rounded border border-gray-200 p-2 dark:border-gray-700">
            <span class="w-32 font-mono text-sm">{{ name }}</span>
            <label class="flex-1">{{ t('default') }}
              <input v-if="parameter.type === 'boolean'" v-model="parameter.default" type="checkbox" />
              <select v-else-if="parameter.enum?.length" v-model="parameter.default" class="input w-full">
                <option v-for="value in parameter.enum" :key="String(value)" :value="value">{{ value }}</option>
              </select>
              <input v-else-if="parameter.type === 'integer'" v-model.number="parameter.default" type="number" :min="parameter.minimum" :max="parameter.maximum" step="1" class="input w-full" />
              <input v-else v-model="parameter.default" class="input w-full" />
            </label>
            <label><input v-model="parameter.locked" type="checkbox" /> {{ t('locked') }}</label>
            <label><input v-model="parameter.required" type="checkbox" /> {{ t('required') }}</label>
            <template v-if="parameter.type === 'integer'">
              <label>{{ t('min') }}<input v-model.number="parameter.minimum" type="number" class="input w-20" /></label>
              <label>{{ t('max') }}<input v-model.number="parameter.maximum" type="number" class="input w-20" /></label>
            </template>
          </div>
          <button v-if="!binding.parameters.reference_image" type="button" class="btn btn-secondary" @click="addReference(binding)">{{ t('addReference') }}</button>
          <h2 class="font-medium">{{ t('prices') }}</h2>
          <div v-for="(price, priceIndex) in binding.prices" :key="priceIndex" class="flex flex-wrap items-end gap-3">
            <label>{{ t('unit') }}<select v-model="price.unit" class="input"><option value="per_job">{{ t('perJob') }}</option><option value="per_second">{{ t('perSecond') }}</option></select></label>
            <label>USD<input v-model="price.usd" type="text" inputmode="decimal" placeholder="0.00" class="input w-28" /></label>
            <label v-if="binding.protocol === 'xai_video'">{{ t('resolution') }}<select v-model="price.resolution" class="input"><option value="">{{ t('any') }}</option><option v-for="value in binding.parameters.resolution?.enum ?? []" :key="String(value)" :value="value">{{ value }}</option></select></label>
            <label v-else>{{ t('size') }}<select v-model="price.size" class="input"><option value="">{{ t('any') }}</option><option v-for="value in binding.parameters.size?.enum ?? []" :key="String(value)" :value="value">{{ value }}</option></select></label>
            <button type="button" class="btn btn-secondary" @click="binding.prices.splice(priceIndex, 1)">{{ t('remove') }}</button>
          </div>
          <button type="button" class="btn btn-secondary" @click="binding.prices.push({ unit: 'per_second', usd: '' })">{{ t('addPrice') }}</button>
          <p class="text-xs text-gray-500">{{ t('priceHelp') }}</p>
          <button type="button" class="btn btn-secondary" :disabled="dirty || !binding.enabled" @click="quote(binding)">{{ t('quote') }}</button>
        </section>
        <div class="flex gap-3">
          <button type="button" class="btn btn-secondary" :disabled="bindings.length >= 128" @click="addBinding">{{ t('add') }}</button>
          <button type="button" class="btn btn-primary" :disabled="!dirty" @click="save">{{ t('save') }}</button>
        </div>
      </fieldset>
    </main>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, onBeforeRouteLeave } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { isAxiosError } from 'axios'
import AppLayout from '@/components/layout/AppLayout.vue'
import { getAllIncludingInactive } from '@/api/admin/groups'
import { getVideoConfig, saveVideoConfig, previewVideoBinding, type VideoBinding, type VideoConfigView, type VideoParameter } from '@/api/admin/videoModels'
import type { AdminGroup } from '@/types'

const en = {
  title: 'Video channel models', back: 'Back to groups', group: 'Existing group', choose: 'Choose a group', load: 'Load', editing: 'Editing group', version: 'Version',
  notActive: 'Configuration and quote management only. These bindings are not yet connected to video submission or settlement; saving them does not enable a new gateway platform.',
  enabled: 'Enable binding', remove: 'Remove', name: 'Channel model name', account: 'Credential-owning account ID', publicModel: 'Public model ID', upstreamModel: 'Upstream model ID', protocol: 'Protocol',
  credentials: 'Use an API-key account already bound to this group. Credentials and provider URLs stay in account management, not this configuration.',
  parameters: 'Capabilities and parameters', default: 'Default', locked: 'Locked', required: 'Required', min: 'Min', max: 'Max', addReference: 'Add HTTPS reference-image parameter',
  prices: 'Variant prices', unit: 'Billing unit', perJob: 'Per job', perSecond: 'Per second', resolution: 'Resolution', size: 'Size', any: 'Any', addPrice: 'Add price tier',
  priceHelp: 'Enter your actual price; no vendor price is assumed. Enabled variants must have one matching tier. Overlapping tiers are rejected by the server.',
  quote: 'Preview saved default quote (no submission)', add: 'Add channel model', save: 'Save configuration', saved: 'Configuration saved; task execution is still not enabled.',
  discard: 'Discard unsaved changes?', reset: 'Changing protocol resets capability defaults and prices. Continue?', failed: 'Operation failed. Check required fields, account binding and non-overlapping prices.',
  conflict: 'The configuration changed elsewhere. Your edits are retained; reload and reconcile.', quoted: 'Default quote: USD {amount}. No video submitted or charged.'
}
const zh: typeof en = {
  title: 'Video 渠道模型', back: '返回分组管理', group: '已有分组', choose: '请选择分组', load: '加载', editing: '正在编辑分组', version: '版本',
  notActive: '目前提供配置保存和报价预览，尚未接入视频提交及结算。保存此配置不会启用新的网关平台。',
  enabled: '启用此绑定', remove: '删除', name: '渠道模型名称', account: '凭证所属账号 ID', publicModel: '对外模型 ID', upstreamModel: '上游模型 ID', protocol: '调用协议',
  credentials: '选择已绑定本分组的 API Key 账号。密钥和供应商地址继续在账号管理中保存，不写入此配置。',
  parameters: '能力与参数', default: '默认值', locked: '锁定', required: '必填', min: '最小值', max: '最大值', addReference: '添加 HTTPS 参考图参数',
  prices: '规格定价', unit: '计价单位', perJob: '按次', perSecond: '按秒', resolution: '分辨率', size: '尺寸', any: '任意', addPrice: '添加价格档位',
  priceHelp: '请填写自己的实际价格，本页不预设供应商价格。每个使用规格必须且只能匹配一个档位；重叠配置会被服务端拒绝。',
  quote: '预览已保存的默认规格报价（不提交视频）', add: '添加渠道模型', save: '保存配置', saved: '配置已保存，但视频任务执行仍未启用。',
  discard: '放弃尚未保存的修改？', reset: '更换协议会重置能力默认值和价格，是否继续？', failed: '操作失败，请核对必填字段、账号绑定和价格档位是否重叠。',
  conflict: '其他位置已修改此配置。当前编辑仍保留，请重新加载并核对。', quoted: '默认规格报价：USD {amount}，没有提交视频或扣费。'
}
const { t } = useI18n({ useScope: 'local', inheritLocale: true, messages: { en, zh } })
const groups = ref<AdminGroup[]>([])
const selected = ref(0)
const view = ref<VideoConfigView | null>(null)
const bindings = ref<VideoBinding[]>([])
const baseline = ref('[]')
const busy = ref(false)
const error = ref('')
const notice = ref('')
const dirty = computed(() => JSON.stringify(bindings.value) !== baseline.value)

function defaults(protocol: VideoBinding['protocol']): Record<string, VideoParameter> {
  if (protocol === 'openai_video_json') return {
    seconds: { type: 'integer', required: true, locked: false, default: 4, minimum: 4, maximum: 12, enum: [4, 8, 12] },
    size: { type: 'string', required: true, locked: false, default: '1280x720', enum: ['720x1280', '1280x720', '1024x1792', '1792x1024'] }
  }
  return {
    seconds: { type: 'integer', required: true, locked: false, default: 5, minimum: 1, maximum: 15 },
    aspect_ratio: { type: 'string', required: true, locked: false, default: '16:9', enum: ['1:1', '16:9', '9:16', '4:3', '3:4', '3:2', '2:3'] },
    resolution: { type: 'string', required: true, locked: false, default: '720p', enum: ['480p', '720p'] },
    generate_audio: { type: 'boolean', required: false, locked: false, default: true }
  }
}
function accept(next: VideoConfigView): void {
  view.value = next
  bindings.value = JSON.parse(JSON.stringify(next.config.bindings)) as VideoBinding[]
  baseline.value = JSON.stringify(bindings.value)
}
function failure(reason: unknown): void {
  error.value = isAxiosError(reason) && reason.response?.status === 409 ? t('conflict') : t('failed')
}
async function load(): Promise<void> {
  if (!selected.value || busy.value || (dirty.value && !window.confirm(t('discard')))) return
  busy.value = true; error.value = notice.value = ''
  try { accept(await getVideoConfig(selected.value)) } catch (reason) { failure(reason) } finally { busy.value = false }
}
async function save(): Promise<void> {
  if (!view.value || busy.value) return
  busy.value = true; error.value = notice.value = ''
  const payload = JSON.parse(JSON.stringify(bindings.value)) as VideoBinding[]
  for (const binding of payload) {
    for (const parameter of Object.values(binding.parameters)) {
      if (parameter.default === '' && !parameter.required && !parameter.locked) delete parameter.default
    }
  }
  try { accept(await saveVideoConfig(view.value.group_id, view.value.version, payload)); notice.value = t('saved') }
  catch (reason) { failure(reason) } finally { busy.value = false }
}
async function quote(binding: VideoBinding): Promise<void> {
  if (!view.value || busy.value || dirty.value) return
  busy.value = true; error.value = notice.value = ''
  try { const result = await previewVideoBinding(view.value.group_id, binding.id); notice.value = t('quoted', { amount: result.quote.base_cost_usd }) }
  catch (reason) { failure(reason) } finally { busy.value = false }
}
function addBinding(): void {
  const ids = new Set(bindings.value.map(binding => binding.id))
  let number = 1
  while (ids.has(`channel_${number}`)) number++
  bindings.value.push({ id: `channel_${number}`, name: '', enabled: false, account_id: 0, public_model: '', upstream_model: '', protocol: 'xai_video', parameters: defaults('xai_video'), prices: [{ unit: 'per_second', usd: '' }] })
}
function changeProtocol(binding: VideoBinding, event: Event): void {
  const select = event.target as HTMLSelectElement
  const protocol = select.value
  if ((protocol !== 'xai_video' && protocol !== 'openai_video_json') || !window.confirm(t('reset'))) { select.value = binding.protocol; return }
  binding.protocol = protocol; binding.parameters = defaults(protocol); binding.prices = [{ unit: 'per_second', usd: '' }]
}
function addReference(binding: VideoBinding): void {
  binding.parameters.reference_image = { type: 'string', required: false, locked: false }
}
onBeforeRouteLeave(() => !dirty.value || window.confirm(t('discard')))
onMounted(async () => {
  busy.value = true
  try { groups.value = await getAllIncludingInactive(); selected.value = groups.value[0]?.id ?? 0 }
  catch (reason) { failure(reason) } finally { busy.value = false }
  if (selected.value) await load()
})
</script>

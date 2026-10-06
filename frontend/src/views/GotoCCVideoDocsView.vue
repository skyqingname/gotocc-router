<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <PlazaNavBar />
    <div class="mx-auto flex max-w-7xl gap-8 px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
      <!-- 目录 -->
      <aside class="sticky top-24 hidden h-fit w-52 shrink-0 lg:block" aria-label="文档目录">
        <p class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">目录</p>
        <nav class="space-y-1 text-sm">
          <a v-for="item in toc" :key="item.id" :href="`#${item.id}`" class="block rounded-lg px-3 py-1.5 text-gray-600 hover:bg-white hover:text-primary-700 dark:text-dark-300 dark:hover:bg-dark-800">{{ item.label }}</a>
        </nav>
      </aside>

      <main ref="mainRef" class="min-w-0 flex-1 space-y-6">
        <!-- 概览 -->
        <section id="overview" class="card space-y-4 p-6">
          <p class="text-sm font-medium text-primary-600 dark:text-primary-400">统一认证，一个接口调用全部视频模型</p>
          <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-3xl">视频接口文档</h1>
          <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">所有视频模型使用同一套创建、查询和下载接口，平台按模型自动转换为对应上游的参数格式。模型列表、能力与价格与模型广场实时同步。</p>
          <div class="flex flex-wrap items-center gap-3 rounded-xl bg-gray-50 px-4 py-3 dark:bg-dark-900/60">
            <span class="text-xs text-gray-500 dark:text-dark-400">API Base URL</span>
            <code class="font-mono text-sm text-gray-900 dark:text-white">{{ baseURL }}</code>
            <button type="button" class="btn btn-secondary btn-sm" @click="copy(baseURL)">复制</button>
          </div>
          <div class="flex flex-wrap items-center gap-3">
            <button type="button" class="btn btn-primary btn-sm" @click="copyMarkdown">复制全部 Markdown</button>
            <span class="text-xs text-gray-500 dark:text-dark-400">把完整接入文档（模型、参数、示例与错误处理）复制给 AI 或开发同事。</span>
          </div>
          <p class="text-xs text-gray-500 dark:text-dark-400">请求头使用 <code>Authorization: Bearer sk-…</code>，密钥在「API 密钥」页面创建；密钥所属分组需包含要调用的模型。</p>
        </section>

        <!-- 可用模型 -->
        <section id="models" class="card space-y-4 p-6">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">可用模型</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">按系列展开查看模型 ID、所属分组、价格与说明。</p>
          </div>
          <p v-if="loading" class="text-sm text-gray-500">正在同步模型…</p>
          <p v-else-if="loadFailed" class="text-sm text-red-600">模型列表暂不可用，请稍后刷新。</p>
          <p v-else-if="!entries.length" class="text-sm text-gray-500">当前没有可调用的视频模型。</p>
          <details v-for="[family, items] in families" :key="family" class="rounded-xl border border-gray-200 dark:border-dark-700" :open="families.length === 1">
            <summary class="flex cursor-pointer items-center justify-between px-4 py-3 text-sm font-medium text-gray-900 dark:text-white">
              <span>{{ family }}</span><span class="text-xs font-normal text-gray-500">{{ items.length }} 个模型</span>
            </summary>
            <div class="divide-y divide-gray-100 border-t border-gray-100 dark:divide-dark-700 dark:border-dark-700">
              <div v-for="item in items" :key="item.name" class="grid gap-1 px-4 py-3 text-sm sm:grid-cols-[minmax(0,14rem)_1fr_auto] sm:items-center sm:gap-4">
                <button type="button" class="truncate text-left font-mono text-primary-700 hover:underline dark:text-primary-300" @click="selectModel(item.name)">{{ item.name }}</button>
                <span class="min-w-0 text-xs leading-5 text-gray-600 dark:text-dark-300">
                  <span class="text-gray-900 dark:text-white">{{ priceText(item) }}</span>
                  <template v-if="item.model.info?.description"> · {{ item.model.info.description }}</template>
                </span>
                <span class="flex flex-wrap gap-1">
                  <span v-for="group in item.groups" :key="group.id" class="rounded-md bg-gray-100 px-1.5 py-0.5 text-[11px] text-gray-600 dark:bg-dark-700 dark:text-dark-200">{{ group.name }}</span>
                  <span class="rounded-md bg-emerald-50 px-1.5 py-0.5 text-[11px] text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300">可用</span>
                </span>
              </div>
            </div>
          </details>
        </section>

        <!-- 创建视频 -->
        <section id="create" class="card space-y-4 p-6">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">创建视频</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">全部视频模型使用同一个创建接口，选择模型后显示它支持的素材与限制。</p>
            </div>
            <label class="text-xs text-gray-500 dark:text-dark-400">示例模型
              <select v-model="selectedName" class="input mt-1 w-64">
                <option v-for="item in entries" :key="item.name" :value="item.name">{{ item.name }}</option>
              </select>
            </label>
          </div>
          <p class="font-mono text-sm"><span class="mr-2 rounded bg-primary-50 px-1.5 py-0.5 text-xs font-semibold text-primary-700 dark:bg-primary-500/15 dark:text-primary-300">POST</span>/v1/videos</p>

          <template v-if="selected">
            <div class="flex flex-wrap gap-2">
              <button v-for="kind in supportedKinds" :key="kind.key" type="button" class="rounded-full border px-3 py-1 text-xs" :class="kind.key === activeKind ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-500/15 dark:text-primary-200' : 'border-gray-200 text-gray-600 dark:border-dark-600 dark:text-dark-300'" @click="exampleKind = kind.key">{{ kind.label }}</button>
            </div>
            <p class="rounded-xl border border-dashed border-gray-300 px-4 py-3 text-sm leading-6 text-gray-700 dark:border-dark-600 dark:text-dark-200">
              <b class="text-primary-700 dark:text-primary-300">模型能力：</b>{{ capabilityText }}
            </p>
            <div>
              <div class="flex gap-1 border-b border-gray-200 dark:border-dark-700">
                <button v-for="lang in languages" :key="lang.key" type="button" class="-mb-px border-b-2 px-3 py-2 text-sm" :class="lang.key === language ? 'border-primary-500 text-primary-700 dark:text-primary-300' : 'border-transparent text-gray-500'" @click="language = lang.key">{{ lang.label }}</button>
              </div>
              <div class="relative mt-3">
                <pre class="max-h-[28rem] overflow-auto rounded-xl bg-gray-900 p-4 text-xs leading-5 text-gray-100"><code>{{ example[language] }}</code></pre>
                <button type="button" class="absolute right-3 top-3 rounded-lg bg-white/10 px-2.5 py-1 text-xs text-white hover:bg-white/20" @click="copy(example[language])">复制</button>
              </div>
            </div>
          </template>

          <dl class="divide-y divide-gray-50 rounded-xl border border-gray-100 text-xs text-gray-700 dark:divide-dark-700/60 dark:border-dark-700 dark:text-dark-200">
            <div v-for="field in fields" :key="field[0]" class="grid gap-1 px-3 py-2 sm:grid-cols-[17rem_1fr] sm:gap-4">
              <dt class="break-all font-mono text-gray-900 dark:text-white">{{ field[0] }}</dt>
              <dd class="leading-5">{{ field[1] }}</dd>
            </div>
          </dl>
          <div class="rounded-xl bg-gray-50 px-4 py-3 text-sm leading-6 text-gray-600 dark:bg-dark-900/60 dark:text-dark-300">
            <p class="font-medium text-gray-900 dark:text-white">参考素材</p>
            <p>参考图、参考视频、参考音频都可以用公网可直接访问的 URL；图片也可以写成 base64 data URL（<code>data:image/jpeg;base64,…</code>）直接放进请求体，平台原样交给上游。若提示获取不到图片，多半是上游拉不到你的图床，改用 base64 直传。数组元素也可以写成 <code>{"url": "…", "role": "first_frame"}</code> 指定首帧 / 尾帧。超出模型能力的请求会直接返回 400 并说明原因，不会扣费。</p>
          </div>
        </section>

        <!-- 查询与下载 -->
        <section id="status" class="card space-y-4 p-6">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">查询与下载</h2>
          <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">创建成功后保存返回的 <code>id</code>，每 10–15 秒查询一次；<code>status</code> 为 <code>completed</code> 后下载视频，避免重复创建。</p>
          <p class="text-sm font-medium text-gray-700 dark:text-dark-200">创建任务 → 保存 id → 轮询状态 → 下载 MP4</p>
          <div class="space-y-1 font-mono text-sm">
            <p><span class="mr-2 rounded bg-emerald-50 px-1.5 py-0.5 text-xs font-semibold text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300">GET</span>/v1/videos/{id}</p>
            <p><span class="mr-2 rounded bg-emerald-50 px-1.5 py-0.5 text-xs font-semibold text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300">GET</span>/v1/videos/{id}/content</p>
          </div>
          <div class="relative">
            <pre class="overflow-auto rounded-xl bg-gray-900 p-4 text-xs leading-5 text-gray-100"><code>{{ statusExample }}</code></pre>
            <button type="button" class="absolute right-3 top-3 rounded-lg bg-white/10 px-2.5 py-1 text-xs text-white hover:bg-white/20" @click="copy(statusExample)">复制</button>
          </div>
          <p class="text-xs text-gray-500 dark:text-dark-400">查询和下载使用创建时的同一个密钥；任务会固定在创建时的上游账号上，无需再次传模型。</p>
        </section>

        <!-- 计费与失败 -->
        <section id="billing" class="card space-y-4 p-6">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">计费与失败处理</h2>
          <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">提交时按模型当前价格预占余额：按秒模型为「单价 × 请求时长」，按次模型为每个视频一个价格。任务成功且视频可以读取后才结算；失败、取消或过期全额释放。</p>
          <div class="overflow-x-auto rounded-xl border border-gray-100 dark:border-dark-700">
            <table class="min-w-full text-left text-xs">
              <thead class="bg-gray-50 text-gray-500 dark:bg-dark-900/60 dark:text-dark-400"><tr><th class="px-3 py-2">结果</th><th class="px-3 py-2">费用</th><th class="px-3 py-2">说明</th></tr></thead>
              <tbody class="divide-y divide-gray-50 text-gray-700 dark:divide-dark-700/60 dark:text-dark-200">
                <tr><td class="px-3 py-2">出片成功</td><td class="px-3 py-2">按预占金额结算</td><td class="px-3 py-2">status 变为 completed，可查询和下载。</td></tr>
                <tr><td class="px-3 py-2">上游失败 / 取消 / 过期</td><td class="px-3 py-2">全额释放，不计消费</td><td class="px-3 py-2">包括内容被拒、上游额度不足、出片超时等；修正后可重新提交。</td></tr>
                <tr><td class="px-3 py-2">参数不符合模型能力</td><td class="px-3 py-2">不扣费</td><td class="px-3 py-2">提交时直接返回 400，并说明哪一项超出限制。</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- 错误处理 -->
        <section id="errors" class="card space-y-4 p-6">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">错误处理</h2>
          <div class="overflow-x-auto rounded-xl border border-gray-100 dark:border-dark-700">
            <table class="min-w-full text-left text-xs">
              <thead class="bg-gray-50 text-gray-500 dark:bg-dark-900/60 dark:text-dark-400"><tr><th class="px-3 py-2">状态码</th><th class="px-3 py-2">含义</th><th class="px-3 py-2">处理方式</th></tr></thead>
              <tbody class="divide-y divide-gray-50 text-gray-700 dark:divide-dark-700/60 dark:text-dark-200">
                <tr v-for="row in errors" :key="row[0]"><td class="px-3 py-2 font-mono">{{ row[0] }}</td><td class="px-3 py-2">{{ row[1] }}</td><td class="px-3 py-2">{{ row[2] }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import { getModelPlaza, type ModelPlazaGroup, type PlazaModel } from '@/api/modelPlaza'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { modelRate, paidPerUnit, perUnitKey, requestIntervals, tierLabel } from '@/components/gotocc/plaza/plazaPricing'
import {
  VIDEO_EXAMPLE_KINDS, buildVideoExample, videoCapabilityItems, videoExampleSupported, type VideoExampleKind,
} from '@/components/gotocc/video/videoDocs'

// GoToCC 视频接口文档：模型、能力与价格取自模型广场，示例按模型能力生成。
const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const baseURL = `${window.location.origin}/v1`
const toc = [
  { id: 'overview', label: '概览' },
  { id: 'models', label: '可用模型' },
  { id: 'create', label: '创建视频' },
  { id: 'status', label: '查询与下载' },
  { id: 'billing', label: '计费与失败' },
  { id: 'errors', label: '错误处理' },
]
const languages = [
  { key: 'curl', label: 'cURL' },
  { key: 'javascript', label: 'JavaScript' },
  { key: 'python', label: 'Python' },
] as const
const fields: [string, string][] = [
  ['model', '必填。模型 ID，见上方可用模型。'],
  ['prompt', '提示词。'],
  ['duration', '视频时长（秒，整数）。也接受 seconds、duration_seconds；不传时使用模型默认时长。'],
  ['resolution', '分辨率，如 480p、720p、1080p。不传时使用模型支持的第一档。'],
  ['aspect_ratio', '画面比例，如 16:9、9:16。也接受 ratio。'],
  ['reference_images', '参考图数组：公网 URL、base64 data URL 或 {"url","role"} 对象。也接受 images、image_urls、input_reference。'],
  ['first_frame_image / last_frame_image', '首帧 / 尾帧图片（URL 或 data URL），等同于 role 为 first_frame / last_frame 的参考图。'],
  ['reference_videos', '参考视频 URL 数组。也接受 videos、video_urls。'],
  ['reference_audios', '参考音频 URL 数组（输入素材，不等于让模型出声）。也接受 audios、audio_urls。'],
  ['generate_audio', '是否生成声音；仅标注「可生成音频」的模型支持。'],
]
const errors: [string, string, string][] = [
  ['400', '参数不符合模型要求', '按返回信息检查模型 ID、时长、分辨率、比例和参考素材数量。'],
  ['401', '认证失败', '检查 Bearer 密钥是否正确、是否已停用。'],
  ['403', '余额不足或无权使用', '充值，或确认密钥所属分组包含该模型。'],
  ['429', '请求过快或达到额度上限', '降低并发并指数退避；额度类限制按返回的重置时间再试。'],
  ['404', '任务不存在', '检查任务 id 是否来自同一个密钥。'],
  ['502 / 503 / 504', '上游暂时不可用', '创建未返回 id 时再谨慎重试；已拿到 id 的继续轮询。'],
]

interface DocsEntry { name: string; model: PlazaModel; group: ModelPlazaGroup; groups: ModelPlazaGroup[] }
const loading = ref(true)
const loadFailed = ref(false)
const groups = ref<ModelPlazaGroup[]>([])
onMounted(async () => {
  void appStore.fetchPublicSettings()
  try {
    groups.value = (await getModelPlaza()).groups
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
})

// 同名模型可能出现在多个分组：示例取第一个分组的能力，所属分组全部列出。
const entries = computed<DocsEntry[]>(() => {
  const byName = new Map<string, DocsEntry>()
  for (const group of groups.value) {
    for (const model of group.models) {
      if (!model.video) continue
      const existing = byName.get(model.name)
      if (existing) existing.groups.push(group)
      else byName.set(model.name, { name: model.name, model, group, groups: [group] })
    }
  }
  return [...byName.values()].sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))
})
const families = computed(() => {
  const map = new Map<string, DocsEntry[]>()
  for (const entry of entries.value) {
    const family = entry.model.video?.family || '其他视频模型'
    map.set(family, [...(map.get(family) ?? []), entry])
  }
  return [...map.entries()]
})

const selectedName = ref('')
watch(entries, (list) => {
  if (list.some((item) => item.name === selectedName.value)) return
  const requested = String(route.query.model ?? '')
  selectedName.value = list.find((item) => item.name === requested)?.name ?? list[0]?.name ?? ''
})
const selected = computed(() => entries.value.find((item) => item.name === selectedName.value))
const capabilities = computed(() => selected.value?.model.video?.capabilities)
const capabilityText = computed(() => [selected.value?.model.info?.description?.replace(/[。；;.\s]+$/, ''), ...videoCapabilityItems(capabilities.value)].filter(Boolean).join('；'))
const supportedKinds = computed(() => VIDEO_EXAMPLE_KINDS.filter((kind) => videoExampleSupported(capabilities.value, kind.key)))
const exampleKind = ref<VideoExampleKind>('text')
const activeKind = computed(() => (supportedKinds.value.some((kind) => kind.key === exampleKind.value) ? exampleKind.value : 'text'))
const language = ref<(typeof languages)[number]['key']>('curl')
const example = computed(() => buildVideoExample(selectedName.value || '请选择模型', capabilities.value, activeKind.value, baseURL))
function selectModel(name: string) {
  selectedName.value = name
  document.getElementById('create')?.scrollIntoView({ behavior: 'smooth' })
}

const statusExample = computed(() => `# 查询任务状态
curl ${baseURL}/videos/{id} \\
  -H "Authorization: Bearer sk-你的API密钥"

# 完成后下载视频
curl -L ${baseURL}/videos/{id}/content \\
  -H "Authorization: Bearer sk-你的API密钥" \\
  -o result.mp4`)

function priceText(entry: DocsEntry) {
  const rate = modelRate(entry.group, entry.model)
  const unit = t(perUnitKey(entry.model))
  const tiers = requestIntervals(entry.model)
  if (tiers.length) return tiers.map((tier) => `${tierLabel(tier)} ${paidPerUnit(tier.per_request_price, rate)}`).join(' · ') + ` ${unit}`
  return entry.model.pricing ? `${paidPerUnit(entry.model.pricing.per_request_price, rate)} ${unit}` : '价格待定'
}

const copy = (value: string) => copyToClipboard(value, '已复制')

// 把渲染后的文档（含当前模型的示例）整理成 Markdown，便于交给 AI 或同事。
const mainRef = ref<HTMLElement | null>(null)
function toMarkdown(node: Node): string {
  if (node.nodeType === Node.TEXT_NODE) return node.nodeValue ?? ''
  if (!(node instanceof HTMLElement) || ['BUTTON', 'SELECT', 'SCRIPT', 'STYLE'].includes(node.tagName)) return ''
  if (node.tagName === 'PRE') return `\n\n\`\`\`\n${node.textContent ?? ''}\n\`\`\`\n\n`
  if (/^H[1-6]$/.test(node.tagName)) return `\n\n${'#'.repeat(Number(node.tagName[1]))} ${node.textContent?.trim()}\n\n`
  if (node.tagName === 'CODE') return `\`${node.textContent?.trim()}\``
  if (node.tagName === 'TR') return `\n| ${[...node.children].map((cell) => cell.textContent?.trim()).join(' | ')} |`
  if (node.tagName === 'DT') return `\n- \`${node.textContent?.trim()}\`：`
  if (node.tagName === 'DD') return node.textContent?.trim() ?? ''
  const content = [...node.childNodes].map(toMarkdown).join('')
  return ['P', 'DIV', 'SECTION', 'DETAILS', 'SUMMARY', 'TABLE', 'LABEL'].includes(node.tagName) ? `\n${content.trim()}\n` : content
}
function copyMarkdown() {
  if (mainRef.value) void copy(toMarkdown(mainRef.value).replace(/\n{3,}/g, '\n\n').trim() + '\n')
}
</script>

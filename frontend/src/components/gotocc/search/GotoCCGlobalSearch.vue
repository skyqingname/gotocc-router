<!-- Derived from TokenFlux/TokenRouter 38ca040de (v0.7.3) frontend/src/components/layout/search/GlobalSearchTrigger.vue, GlobalSearchPalette.vue, GlobalSearchResultRow.vue and searchShortcuts.ts. Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation. -->
<template>
  <div class="flex shrink-0 items-center">
    <!-- 窄屏是与铃铛同样的图标按钮；从 md 起像一个搜索框，右侧是快捷键提示 -->
    <button
      type="button"
      class="flex h-9 w-9 items-center justify-center rounded-lg transition-colors hover:surface-hover surface-control surface-muted md:hidden"
      aria-haspopup="dialog"
      :aria-expanded="open"
      :aria-label="t('gotocc.search.open')"
      @click="open = true"
    >
      <Icon name="search" size="md" />
    </button>
    <button type="button" class="gc-search-trigger hidden md:flex" aria-haspopup="dialog" :aria-expanded="open" :aria-keyshortcuts="ariaShortcut" @click="open = true">
      <Icon name="search" size="sm" class="shrink-0" />
      <span class="min-w-0 flex-1 truncate text-left">{{ t('gotocc.search.placeholder') }}</span>
      <kbd class="gc-kbd">{{ shortcutLabel }}</kbd>
    </button>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="open" class="modal-overlay gc-search-overlay z-[100]" @click.self="open = false">
          <section
            ref="panelRef"
            role="dialog"
            aria-modal="true"
            :aria-label="t('gotocc.search.dialogLabel')"
            tabindex="-1"
            class="gc-search-panel modal-content max-w-2xl overflow-hidden"
          >
            <div class="gc-search-input-row">
              <Icon name="search" size="md" class="shrink-0 surface-muted" />
              <input
                ref="inputRef"
                v-model="query"
                type="text"
                role="combobox"
                aria-autocomplete="list"
                aria-expanded="true"
                :aria-controls="LISTBOX_ID"
                :aria-activedescendant="activeOptionId"
                :placeholder="t('gotocc.search.placeholder')"
                autocomplete="off"
                autocapitalize="off"
                spellcheck="false"
                enterkeyhint="go"
                class="gc-search-input"
                @keydown="onInputKeydown"
              />
              <kbd class="gc-kbd hidden sm:inline-flex">Esc</kbd>
              <button type="button" class="gc-search-close sm:hidden" :aria-label="t('common.close')" @click="open = false">
                <Icon name="x" size="sm" />
              </button>
            </div>

            <!-- 范围切换不抢输入框的焦点，点完可以接着打字 -->
            <div class="shrink-0 px-3 pt-3" @mousedown.prevent>
              <div v-segmented class="segmented inline-flex max-w-full" role="tablist" :aria-label="t('gotocc.search.scopeLabel')">
                <button
                  v-for="option in scopeOptions"
                  :key="option.key"
                  type="button"
                  role="tab"
                  :aria-selected="scope === option.key"
                  class="segmented-item h-8 px-3 text-sm font-medium"
                  :class="scope === option.key && 'segmented-item-active'"
                  @click="scope = option.key"
                >
                  {{ option.label }}<span v-if="option.count !== null" class="ml-1 text-xs tabular-nums opacity-70">{{ option.count }}</span>
                </button>
              </div>
            </div>

            <!-- 焦点一直留在输入框：上下键移动当前行，回车打开，鼠标点击行同样打开 -->
            <div :id="LISTBOX_ID" role="listbox" :aria-label="t('gotocc.search.dialogLabel')" class="gc-search-results" @mousedown.prevent>
              <p v-if="noResults" class="px-4 py-10 text-center text-sm surface-muted">{{ t('gotocc.search.noResults', { query: trimmedQuery }) }}</p>
              <section v-for="section in layout" :key="section.key" class="menu-section" role="group" :aria-label="section.title">
                <p class="gc-search-heading">{{ section.title }}</p>
                <div v-if="section.status === 'loading'" aria-busy="true">
                  <div v-for="row in config.loading_rows" :key="row" class="flex min-h-11 items-center gap-3 px-2.5">
                    <Skeleton variant="circle" :width="20" :height="20" />
                    <div class="flex-1 space-y-1.5">
                      <Skeleton width="40%" :height="12" />
                      <Skeleton width="24%" :height="10" />
                    </div>
                  </div>
                </div>
                <p v-else-if="section.status === 'failed'" class="px-2.5 py-2 text-sm surface-muted">{{ t('gotocc.search.modelsFailed') }}</p>
                <div
                  v-for="(row, index) in section.rows"
                  :id="optionId(section.offset + index)"
                  :key="row.item.key"
                  role="option"
                  :aria-selected="section.offset + index === activeIndex"
                  class="gc-search-row"
                  :class="{ 'gc-search-row-active': section.offset + index === activeIndex }"
                  @click="select(row.item)"
                  @mousemove="activeIndex = section.offset + index"
                >
                  <ModelIcon v-if="row.item.model" :model="row.item.model" size="20px" class="shrink-0" />
                  <component :is="row.item.navIcon" v-else-if="row.item.navIcon" class="h-5 w-5 shrink-0 surface-muted" />
                  <span v-else-if="row.item.iconSvg" class="gc-search-svg h-5 w-5 shrink-0 surface-muted" v-html="sanitizeSvg(row.item.iconSvg)"></span>
                  <Icon v-else-if="row.item.icon" :name="row.item.icon" size="md" class="shrink-0 surface-muted" />
                  <span class="min-w-0 flex-1">
                    <span class="block truncate text-sm font-medium text-gray-900 dark:text-white">
                      <template v-for="(segment, segmentIndex) in titleSegments(row)" :key="segmentIndex">
                        <mark v-if="segment.matched" class="gc-search-mark">{{ segment.text }}</mark>
                        <template v-else>{{ segment.text }}</template>
                      </template>
                    </span>
                    <span v-if="row.item.subtitle" class="block truncate text-xs surface-muted">{{ row.item.subtitle }}</span>
                  </span>
                </div>
              </section>
            </div>

            <footer class="gc-search-footer">
              <span><kbd class="gc-kbd">↑</kbd><kbd class="gc-kbd">↓</kbd>{{ t('gotocc.search.hints.navigate') }}</span>
              <span><kbd class="gc-kbd">↵</kbd>{{ t('gotocc.search.hints.open') }}</span>
              <span><kbd class="gc-kbd">Esc</kbd>{{ t('gotocc.search.hints.close') }}</span>
            </footer>
          </section>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useEventListener } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { getModelPlaza, type ModelPlazaGroup } from '@/api/modelPlaza'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import { useDialogFocus } from '@/composables/useDialogFocus'
import { availableLocales, loadLocaleMessages } from '@/i18n'
import { useAuthStore } from '@/stores'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { sanitizeSvg } from '@/utils/sanitize'
import { vSegmented } from '../vSegmented'
import { highlightSegments, type TextSegment } from './fuzzyMatch'
import { rankItems, readRecent, recordRecent, useSearchCatalog, type RankedItem, type SearchItem, type SearchKind } from './searchCatalog'
import config from '../../../../../global-search.json'

// 顶栏全局搜索：点入口、按 Cmd/Ctrl+K 或在输入区以外按「/」打开居中弹窗，按「全部 / 模型 / 页面 / 动作」分段筛选。
// 没输入时列出最近访问、各厂商最新上线的模型和其余各类条目；输入后各类分别模糊匹配排序。最近访问按账号存在浏览器本地，只存条目键。
// 页面行用侧栏同款动效图标，成为当前行（键盘移动或鼠标经过）时播放动效，参照 TokenRouter 的全局搜索。
// 输入法组合输入期间的按键全部交给输入法，不触发回车、上下键、Esc 和快捷键。

const LISTBOX_ID = 'gc-search-listbox'

type Scope = 'all' | SearchKind

interface Section {
  key: string
  title: string
  rows: RankedItem[]
  status?: 'loading' | 'failed'
  offset: number
}

const { t } = useI18n()
const route = useRoute()
const authStore = useAuthStore()

const open = ref(false)
const query = ref('')
const scope = ref<Scope>('all')
const activeIndex = ref(0)
const recentKeys = ref<string[]>([])
const panelRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const plazaGroups = ref<ModelPlazaGroup[]>([])
const modelStatus = ref<'idle' | 'loading' | 'loaded' | 'failed'>('idle')

const catalog = useSearchCatalog(plazaGroups)
const trimmedQuery = computed(() => query.value.trim())

// 模型广场关闭时没有「模型」范围。
const kinds = computed<SearchKind[]>(() => (isFeatureFlagEnabled(FeatureFlags.modelPlaza) ? ['model', 'page', 'action'] : ['page', 'action']))
const itemsByKind = computed<Record<SearchKind, SearchItem[]>>(() => ({
  model: catalog.models.value,
  page: catalog.pages.value,
  action: catalog.actions.value,
}))
const ranked = computed(() => (trimmedQuery.value
  ? Object.fromEntries(kinds.value.map((kind) => [kind, rankItems(trimmedQuery.value, itemsByKind.value[kind], recentKeys.value)])) as Record<SearchKind, RankedItem[]>
  : null))

const isApple = /mac|iphone|ipad|ipod/i.test(navigator.platform)
const shortcutKey = config.shortcut_key.toUpperCase()
const shortcutLabel = isApple ? `⌘${shortcutKey}` : `Ctrl ${shortcutKey}`
const ariaShortcut = `${isApple ? 'Meta' : 'Control'}+${shortcutKey}`

const unranked = (item: SearchItem): RankedItem => ({ item, match: null })

// 「全部」每类最多 scope_limits.all 行，单一范围最多 scope_limits.single 行。没输入时「全部」先列最近访问，下面各类不再重复。
// 没输入时模型一节是各厂商最新上线的几款：「全部」里每家 recommended_models_per_vendor.all 款，「模型」里每家 .single 款，下面接着列其余模型。
const sections = computed<Omit<Section, 'offset'>[]>(() => {
  const limit = scope.value === 'all' ? config.scope_limits.all : config.scope_limits.single
  const perVendor = scope.value === 'all' ? config.recommended_models_per_vendor.all : config.recommended_models_per_vendor.single
  const result: Omit<Section, 'offset'>[] = []
  const listed = new Set<string>()
  if (!ranked.value && scope.value === 'all') {
    const byKey = new Map(kinds.value.flatMap((kind) => itemsByKind.value[kind]).map((item) => [item.key, item]))
    const recent = recentKeys.value.flatMap((key) => (byKey.has(key) ? [byKey.get(key)!] : []))
    recent.forEach((item) => listed.add(item.key))
    if (recent.length > 0) result.push({ key: 'recent', title: t('gotocc.search.sections.recent'), rows: recent.map(unranked) })
  }
  for (const kind of scope.value === 'all' ? kinds.value : [scope.value]) {
    const latest = kind === 'model' && !ranked.value
    const rows = ranked.value
      ? ranked.value[kind].slice(0, limit)
      : latest
        ? catalog.latestByVendor.value.flatMap((list) => list.filter((item) => !listed.has(item.key)).slice(0, perVendor)).map(unranked)
        : itemsByKind.value[kind].filter((item) => !listed.has(item.key)).map(unranked).slice(0, limit)
    const status = kind === 'model' && (modelStatus.value === 'loading' || modelStatus.value === 'failed') ? modelStatus.value : undefined
    if (rows.length === 0 && !status) continue
    result.push({ key: kind, title: t(latest ? 'gotocc.search.sections.latestModels' : `gotocc.search.sections.${kind}`), rows, status })
    if (latest && scope.value === 'model') {
      rows.forEach((row) => listed.add(row.item.key))
      const rest = itemsByKind.value.model.filter((item) => !listed.has(item.key)).map(unranked).slice(0, limit)
      if (rest.length > 0) result.push({ key: 'model-rest', title: t('gotocc.search.sections.model'), rows: rest })
    }
  }
  return result
})

const noResults = computed(() => trimmedQuery.value !== '' && sections.value.length === 0)

const layout = computed<Section[]>(() => {
  let offset = 0
  return sections.value.map((section) => {
    const laidOut = { ...section, offset }
    offset += section.rows.length
    return laidOut
  })
})
const flatRows = computed(() => layout.value.flatMap((section) => section.rows))

// 输入后各范围标出命中数。
const scopeOptions = computed(() => (['all', ...kinds.value] as Scope[]).map((key) => ({
  key,
  label: t(`gotocc.search.scopes.${key}`),
  count: ranked.value && key !== 'all' ? ranked.value[key].length : null,
})))

const optionId = (index: number) => `${LISTBOX_ID}-${index}`
const activeOptionId = computed(() => (flatRows.value.length > 0 ? optionId(activeIndex.value) : undefined))

// 只有标题命中时才高亮，命中的是模型 ID、另一种语言或路径时标题原样显示。
const titleSegments = (row: RankedItem): TextSegment[] =>
  row.match?.field === 0 ? highlightSegments(row.item.title, row.match.ranges) : [{ text: row.item.title, matched: false }]

async function loadModels() {
  modelStatus.value = 'loading'
  try {
    plazaGroups.value = (await getModelPlaza()).groups
    modelStatus.value = 'loaded'
  } catch {
    modelStatus.value = 'failed'
  }
}

// 模型和页面记入最近访问，动作不记。
function select(item: SearchItem) {
  if (item.kind !== 'action') recentKeys.value = recordRecent(authStore.user!.id, item.key)
  open.value = false
  void item.run()
}

function onInputKeydown(event: KeyboardEvent) {
  if (event.isComposing || event.keyCode === 229) {
    event.stopPropagation()
    return
  }
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    const count = flatRows.value.length
    if (count > 0) activeIndex.value = (activeIndex.value + (event.key === 'ArrowDown' ? 1 : -1) + count) % count
  } else if (event.key === 'Enter' && flatRows.value[activeIndex.value]) {
    event.preventDefault()
    select(flatRows.value[activeIndex.value].item)
  }
}

const isEditable = (target: EventTarget | null) =>
  target instanceof HTMLElement && (target.isContentEditable || target.closest('input, textarea, select, [contenteditable=""], [contenteditable="true"]') !== null)

// Cmd/Ctrl+K 开关弹窗；「/」只在输入区以外打开。别的弹窗开着时不叠加打开。
function onGlobalKeydown(event: KeyboardEvent) {
  if (event.isComposing || event.keyCode === 229) return
  const otherDialogOpen = !open.value && document.body.classList.contains('modal-open')
  const modifier = isApple ? event.metaKey : event.ctrlKey
  if (modifier && !event.altKey && !event.shiftKey && event.key.toLowerCase() === config.shortcut_key) {
    if (otherDialogOpen) return
    event.preventDefault()
    open.value = !open.value
  } else if (event.key === config.slash_key && !open.value && !otherDialogOpen && !event.metaKey && !event.ctrlKey && !event.altKey && !isEditable(event.target)) {
    event.preventDefault()
    open.value = true
  }
}

useEventListener(window, 'keydown', onGlobalKeydown)
useDialogFocus(() => open.value, panelRef, () => { open.value = false })

// 每次打开回到「全部」并清空输入，读取本账号的最近访问；第一次打开时加载模型和两种语言的文案。
watch(open, (value) => {
  if (!value) return
  query.value = ''
  scope.value = 'all'
  activeIndex.value = 0
  recentKeys.value = readRecent(authStore.user!.id)
  void Promise.all(availableLocales.map(({ code }) => loadLocaleMessages(code)))
  if (kinds.value.includes('model') && (modelStatus.value === 'idle' || modelStatus.value === 'failed')) void loadModels()
})

watch(() => route.fullPath, () => { open.value = false })
watch([trimmedQuery, scope], () => { activeIndex.value = 0 })
watch(() => flatRows.value.length, (count) => {
  if (activeIndex.value >= count) activeIndex.value = Math.max(0, count - 1)
})
watch(activeIndex, async (index) => {
  await nextTick()
  document.getElementById(optionId(index))?.scrollIntoView({ block: 'nearest' })
})
</script>

<style scoped>
.gc-search-svg :deep(svg) {
  width: 100%;
  height: 100%;
}

/* 入口：与顶栏其它按钮同为 44px 高，底色和描边沿用暖色外壳变量，悬停时描边带主色。 */
.gc-search-trigger {
  @apply h-11 w-44 min-w-0 items-center gap-2 rounded-lg border px-3 text-sm lg:w-52 xl:w-64;
  border-color: var(--surface-edge);
  background-color: var(--surface-soft);
  color: var(--surface-muted);
  transition: border-color var(--motion-fast) var(--motion-ease);
}

.gc-search-trigger:hover {
  border-color: theme('colors.primary.400 / 60%');
}

.gc-kbd {
  @apply inline-flex h-5 min-w-[1.25rem] shrink-0 items-center justify-center rounded border px-1 font-sans text-[11px] font-medium leading-none;
  border-color: var(--surface-edge);
  background-color: var(--surface-field);
  color: var(--surface-muted);
}

/* 弹窗：手机上贴顶留 8px，从 sm 起水平居中、距顶 12%，输入框位置不随结果多少跳动。 */
.gc-search-overlay {
  @apply items-start sm:pt-[12vh];
}

/* 管理员把卡片调成半透明时，弹窗后面的页面先模糊，文字不会和背后的内容叠在一起（同公告弹窗）。 */
.gc-search-panel {
  max-height: min(80dvh, 40rem);
  -webkit-backdrop-filter: blur(var(--glass-blur));
  backdrop-filter: blur(var(--glass-blur));
}

.gc-search-input-row {
  @apply flex h-14 shrink-0 items-center gap-3 border-b px-4;
  border-color: var(--surface-edge);
}

/* 16px 字号避免手机 Safari 聚焦时放大页面。 */
.gc-search-input {
  @apply h-full min-w-0 flex-1 bg-transparent text-base outline-none;
  color: var(--surface-text);
}

.gc-search-input::placeholder {
  color: var(--surface-muted);
}

.gc-search-close {
  @apply flex h-8 w-8 shrink-0 items-center justify-center rounded-lg;
  color: var(--surface-muted);
}

.gc-search-results {
  @apply min-h-0 flex-1 overflow-y-auto overscroll-contain pt-1;
}

.gc-search-heading {
  @apply px-2.5 pb-1 pt-1 text-xs font-medium;
  color: var(--surface-muted);
}

.gc-search-row {
  @apply flex min-h-11 cursor-pointer items-center gap-3 rounded-lg px-2.5 py-1.5;
  transition: background-color var(--motion-fast) var(--motion-ease);
}

.gc-search-row-active {
  background-color: var(--surface-hover);
}

.gc-search-mark {
  @apply bg-transparent font-semibold text-primary-600 dark:text-primary-300;
}

.gc-search-footer {
  @apply hidden shrink-0 items-center gap-4 border-t px-4 py-2 text-xs sm:flex;
  border-color: var(--surface-edge);
  background-color: var(--surface-soft);
  color: var(--surface-muted);
}

.gc-search-footer > span {
  @apply flex items-center gap-1;
}
</style>

<template>
  <div class="space-y-5">
    <!-- 独立形态展示页头；后台形态顶栏已有页面标题 -->
    <div v-if="!embedded">
      <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-3xl">{{ t('modelPlaza.title') }}</h1>
      <p class="mt-1.5 text-sm text-gray-500 dark:text-dark-400">{{ t('modelPlaza.description') }}</p>
    </div>

    <!-- 全局价格说明（管理员配置，Markdown） -->
    <div v-if="descriptionHtml" class="plaza-description card px-5 py-4 text-sm" v-html="descriptionHtml"></div>

    <p v-if="!isAuthenticated" class="flex items-center gap-1.5 text-xs text-gray-400 dark:text-dark-500">
      <Icon name="infoCircle" size="xs" />
      {{ t('modelPlaza.anonymousHint') }}
    </p>

    <div v-if="loading" class="grid gap-4 md:grid-cols-2 2xl:grid-cols-3" aria-busy="true">
      <div v-for="item in 6" :key="item" class="card space-y-3 p-4">
        <Skeleton width="60%" :height="20" />
        <Skeleton width="90%" :height="14" />
        <Skeleton :height="92" />
      </div>
    </div>
    <div v-else-if="error" class="rounded-2xl border border-red-200 bg-red-50 px-5 py-8 text-center text-sm text-red-600 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300">
      {{ t('modelPlaza.loadFailed') }}
    </div>
    <template v-else>
      <!-- 筛选：搜索、厂商、用途、分组 -->
      <div class="card space-y-3 p-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="relative w-full sm:w-72">
            <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model.trim="search" type="search" class="input pl-9" :placeholder="t('gotocc.plaza.searchPlaceholder')" />
          </div>
          <span class="flex items-center gap-3 text-sm text-gray-500 dark:text-dark-400">
            <RouterLink v-if="hasVideoModels" to="/docs/video" class="inline-flex items-center gap-1 rounded-full border border-primary-200 px-3 py-1 text-xs font-medium text-primary-700 hover:bg-primary-50 dark:border-primary-500/30 dark:text-primary-300 dark:hover:bg-primary-500/10">
              <Icon name="book" size="xs" />{{ t('gotocc.plaza.videoDocs') }}
            </RouterLink>
            {{ t('gotocc.plaza.summary', { models: visibleCount, groups: visibleGroups.length }) }}
          </span>
        </div>
        <div v-for="row in filterRows" :key="row.key" class="flex flex-wrap items-center gap-2">
          <span class="w-10 shrink-0 text-xs text-gray-500 dark:text-dark-400">{{ row.label }}</span>
          <button
            v-for="option in row.options"
            :key="String(option.value)"
            type="button"
            class="plaza-chip"
            :class="row.selected === option.value ? 'plaza-chip-on' : 'plaza-chip-off'"
            :aria-pressed="row.selected === option.value"
            @click="row.select(option.value)"
          >
            {{ option.label }}<span class="ml-1 tabular-nums opacity-70">{{ option.count }}</span>
          </button>
        </div>
      </div>

      <!-- 分组：头部说明，内含模型卡片；可收起 -->
      <section v-for="group in visibleGroups" :key="group.id" class="plaza-group rounded-2xl border border-primary-100 p-4 dark:border-primary-500/20" :class="{ 'plaza-group-open': !collapsed.has(group.id) }">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="flex flex-wrap items-center gap-2">
              <Icon name="cube" size="sm" class="text-primary-600 dark:text-primary-400" />
              <span class="font-semibold text-gray-900 dark:text-white">{{ group.name }}</span>
              <span class="rounded-full bg-primary-600 px-2 py-0.5 text-xs font-semibold tabular-nums text-white">{{ formatRate(groupRate(group)) }}</span>
              <span v-if="group.is_exclusive" class="badge badge-purple">{{ t('modelPlaza.badges.exclusive') }}</span>
              <span v-if="group.subscription_type === 'subscription'" class="badge badge-primary">{{ t('modelPlaza.badges.subscription') }}</span>
            </p>
            <p v-if="group.description" class="mt-1.5 text-sm text-gray-600 dark:text-dark-300">{{ group.description }}</p>
            <p v-for="note in groupNotes(group)" :key="note" class="mt-2 inline-flex items-center gap-1.5 rounded-lg bg-primary-50 px-2.5 py-1 text-xs text-primary-700 dark:bg-primary-500/10 dark:text-primary-300">
              <Icon name="infoCircle" size="xs" />{{ note }}
            </p>
          </div>
          <button type="button" class="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-gray-200 px-3 py-1 text-xs text-gray-600 hover:border-gray-300 dark:border-dark-600 dark:text-dark-300" :aria-expanded="!collapsed.has(group.id)" @click="toggle(group.id)">
            {{ t('gotocc.plaza.modelCount', { count: displayedGroupCount(group) }) }}
            <Icon name="chevronDown" size="xs" class="plaza-chevron" />
          </button>
        </div>
        <div class="plaza-collapse" :inert="collapsed.has(group.id)">
          <div class="plaza-collapse-inner">
            <PlazaVideoModels v-if="group.platform === 'video'" :group="group" @count="videoCounts[group.id] = $event" @detail="detail = { group, model: $event }" />
            <div v-else class="grid gap-4 pt-4 md:grid-cols-2 2xl:grid-cols-3">
              <PlazaModelCard
                v-for="(model, index) in group.models"
                :key="`${model.platform}:${model.name}`"
                class="plaza-rise"
                :style="{ '--i': Math.min(index, 11) }"
                :group="group"
                :model="model"
                @detail="detail = { group, model }"
              />
            </div>
          </div>
        </div>
      </section>

      <div v-if="visibleGroups.length === 0" class="rounded-2xl border border-dashed border-gray-300 px-5 py-12 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400">
        {{ hasFilter ? t('modelPlaza.noSearchResult') : t('modelPlaza.empty') }}
      </div>
    </template>

    <PlazaModelDetail :group="detail?.group ?? null" :model="detail?.model ?? null" @close="detail = null" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import type { ModelPlazaGroup, ModelPlazaResponse, PlazaModel } from '@/api/modelPlaza'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { formatPeakRateWindow, hasPeakRate, serverTimezoneLabel } from '@/utils/peak-rate'
import PlazaModelCard from './PlazaModelCard.vue'
import PlazaModelDetail from './PlazaModelDetail.vue'
import PlazaVideoModels from './PlazaVideoModels.vue'
import { formatRate, groupRate, sortModels } from './plazaPricing'
import { PURPOSES, canonicalVendor, vendorLabel } from './vendors'

// GoToCC 模型广场：按分组展示模型卡片，可按厂商、用途、分组和名称筛选；模型官方信息来自随版 models.dev 快照。
const props = defineProps<{
  response: ModelPlazaResponse | null
  loading: boolean
  error?: boolean
  // 后台内嵌形态（AppLayout 内）隐藏页头。
  embedded?: boolean
}>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const isAuthenticated = computed(() => authStore.isAuthenticated)

const search = ref('')
const vendor = ref<string>('all')
const purpose = ref<string>('all')
const groupId = ref<number | 'all'>('all')
const videoCounts = ref<Record<number, number>>({})
const collapsed = ref(new Set<number>())
const detail = ref<{ group: ModelPlazaGroup; model: PlazaModel } | null>(null)

const descriptionHtml = computed(() => {
  const markdown = props.response?.description?.trim()
  return markdown ? DOMPurify.sanitize(marked.parse(markdown) as string) : ''
})

const groups = computed(() => props.response?.groups ?? [])
const hasVideoModels = computed(() => groups.value.some((group) => group.models.some((model) => model.video)))
const vendorOf = (model: PlazaModel) => canonicalVendor(model.info?.vendor ?? model.platform)
const purposesOf = (model: PlazaModel) => model.info?.purposes ?? []

const matchesSearch = (model: PlazaModel) => {
  const keyword = search.value.toLowerCase()
  return !keyword || model.name.toLowerCase().includes(keyword) || (model.info?.display_name ?? '').toLowerCase().includes(keyword)
}

// 筛选后的分组：组内只留命中的模型，按名称、厂商、用途过滤；整组无命中则隐藏；按生效倍率升序。
const visibleGroups = computed(() => groups.value
  .filter((group) => groupId.value === 'all' || group.id === groupId.value)
  .map((group) => ({
    ...group,
    models: sortModels(group.models.filter((model) =>
      matchesSearch(model)
      && (vendor.value === 'all' || vendorOf(model) === vendor.value)
      && (purpose.value === 'all' || purposesOf(model).includes(purpose.value)))),
  }))
  .filter((group) => group.models.length > 0)
  .sort((a, b) => groupRate(a) - groupRate(b) || a.name.localeCompare(b.name)))

const displayedGroupCount = (group: ModelPlazaGroup) => group.platform === 'video' ? (videoCounts.value[group.id] ?? group.models.length) : group.models.length
const visibleCount = computed(() => visibleGroups.value.reduce((sum, group) => sum + displayedGroupCount(group), 0))
const hasFilter = computed(() => search.value !== '' || vendor.value !== 'all' || purpose.value !== 'all' || groupId.value !== 'all')

// 筛选项的计数基于全部模型，选中项变化时不跳动。
const allModels = computed(() => groups.value.flatMap((group) => group.models))
const countBy = (keys: string[]) => keys.reduce<Record<string, number>>((counts, key) => ({ ...counts, [key]: (counts[key] ?? 0) + 1 }), {})

const filterRows = computed(() => {
  const vendorCounts = countBy(allModels.value.map(vendorOf))
  const purposeCounts = countBy(allModels.value.flatMap(purposesOf))
  const all = { value: 'all' as string | number, label: t('gotocc.plaza.all'), count: allModels.value.length }
  const rows = [
    {
      key: 'vendor',
      label: t('gotocc.plaza.vendor'),
      selected: vendor.value as string | number,
      select: (value: string | number) => { vendor.value = String(value) },
      options: [all, ...Object.entries(vendorCounts).sort((a, b) => b[1] - a[1]).map(([value, count]) => ({ value, label: vendorLabel(value), count }))],
    },
    {
      key: 'purpose',
      label: t('gotocc.plaza.purpose'),
      selected: purpose.value as string | number,
      select: (value: string | number) => { purpose.value = String(value) },
      options: [all, ...PURPOSES.filter((value) => purposeCounts[value]).map((value) => ({ value, label: t(`gotocc.plaza.purposes.${value}`), count: purposeCounts[value] }))],
    },
  ]
  if (groups.value.length > 1) {
    rows.push({
      key: 'group',
      label: t('gotocc.plaza.group'),
      selected: groupId.value,
      select: (value: string | number) => { groupId.value = value === 'all' ? 'all' : Number(value) },
      options: [all, ...groups.value.map((group) => ({ value: group.id, label: group.name, count: group.models.length }))],
    })
  }
  return rows
})

// 分组提示：高峰倍率时段；未开长上下文分档但官方有分档时提示官方分档仅供参考。
const groupNotes = (group: ModelPlazaGroup): string[] => {
  const notes: string[] = []
  if (hasPeakRate(group)) {
    const window = formatPeakRateWindow(group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
    notes.push(t('modelPlaza.detail.peakNote', { window, multiplier: group.peak_rate_multiplier }))
  }
  if (group.long_context_pricing_enabled === false && group.models.some((model) => (model.official_pricing?.intervals?.length ?? 0) > 1)) {
    notes.push(t('modelPlaza.detail.longContextDisabledNote'))
  }
  return notes
}

const toggle = (id: number) => {
  const next = new Set(collapsed.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  collapsed.value = next
}
</script>

<style scoped>
.plaza-group {
  background-image: linear-gradient(180deg, theme('colors.primary.50'), transparent 160px);
}

.dark .plaza-group {
  background-image: linear-gradient(180deg, rgb(99 102 241 / 0.06), transparent 160px);
}

.plaza-chip {
  @apply inline-flex items-center rounded-full border px-3 py-1 text-xs;
  transition:
    background-color var(--motion-fast) var(--motion-ease),
    color var(--motion-fast) var(--motion-ease),
    border-color var(--motion-fast) var(--motion-ease);
}

.plaza-chip-on {
  @apply border-primary-600 bg-primary-600 text-white;
}

.plaza-chip-off {
  @apply border-gray-200 text-gray-600 hover:border-gray-300 dark:border-dark-600 dark:text-dark-300 dark:hover:border-dark-500;
}

/* 分组展开收起：grid 行高过渡。 */
.plaza-collapse {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows var(--motion-layout) var(--motion-ease);
}

.plaza-group-open .plaza-collapse {
  grid-template-rows: 1fr;
}

.plaza-collapse-inner {
  min-height: 0;
  overflow: hidden;
}

.plaza-chevron {
  transition: transform var(--motion-normal) var(--motion-ease);
}

.plaza-group-open .plaza-chevron {
  transform: rotate(180deg);
}

/* 卡片首次出现时依次上浮淡入，最多错开 12 张。 */
.plaza-rise {
  animation: plaza-rise var(--motion-layout) var(--motion-ease) backwards;
  animation-delay: calc(var(--i) * 30ms);
}

@keyframes plaza-rise {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .plaza-rise {
    animation: none;
  }
}

.plaza-description :deep(p + p) {
  margin-top: 0.5rem;
}

.plaza-description :deep(a) {
  @apply text-primary-600 underline dark:text-primary-400;
}
</style>

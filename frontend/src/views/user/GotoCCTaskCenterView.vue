<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('gotocc.taskCenter.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('gotocc.taskCenter.subtitle') }}</p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="refreshing" @click="refresh">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': refreshing }" />
          {{ t('common.refresh') }}
        </button>
      </div>

      <!-- 统计随当前页签和筛选条件变化 -->
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <MetricCard v-for="card in cards" :key="card.key" v-bind="card.props" :loading="summary === null && !summaryError" />
      </div>

      <div class="flex items-start gap-2 text-xs text-gray-500 dark:text-dark-400">
        <Icon name="infoCircle" size="sm" class="mt-px shrink-0 text-primary-500" />
        <p class="flex flex-wrap gap-x-1">
          <span>{{ t('gotocc.taskCenter.hint') }}</span>
          <span v-if="tab === 'image'">{{ t('gotocc.taskCenter.imageHint') }}</span>
        </p>
      </div>

      <TaskFilterBar v-model:start="draft.start" v-model:end="draft.end" v-model:model="draft.model" :models="models" @reset="reset" @query="query" />

      <div class="card overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-5 py-3 dark:border-dark-700">
          <div class="flex flex-wrap items-center gap-3">
            <div v-segmented class="segmented inline-flex" role="tablist" :aria-label="t('gotocc.taskCenter.tabs.label')">
              <button
                v-for="item in tabs"
                :key="item.value"
                type="button"
                role="tab"
                class="segmented-item h-8 px-3 text-sm font-medium"
                :class="tab === item.value && 'segmented-item-active'"
                :aria-selected="tab === item.value"
                @click="selectTab(item.value)"
              >
                {{ item.label }}
              </button>
            </div>
            <span v-if="summary" class="text-xs tabular-nums text-gray-500 dark:text-dark-400">{{ t('gotocc.taskCenter.resultCount', { count: summary.total }) }}</span>
          </div>
          <RouterLink to="/usage" class="inline-flex items-center gap-1 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300">
            {{ t('gotocc.taskCenter.usageLink') }}
            <Icon name="arrowRight" size="xs" />
          </RouterLink>
        </div>

        <div v-if="listError" class="flex flex-col items-center gap-3 px-6 py-14 text-center">
          <Icon name="exclamationCircle" size="lg" class="text-red-500" />
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('gotocc.taskCenter.loadFailed') }}</p>
          <p class="max-w-md text-xs text-gray-500 dark:text-dark-400">{{ listError }}</p>
          <button type="button" class="btn btn-secondary btn-sm" @click="loadList()">
            <Icon name="refresh" size="sm" />
            {{ t('gotocc.taskCenter.retry') }}
          </button>
        </div>
        <ul v-else-if="!listLoaded" class="divide-y divide-gray-100 dark:divide-dark-700" :aria-busy="true">
          <li v-for="row in SKELETON_ROWS" :key="row" class="space-y-2 px-5 py-4">
            <Skeleton :width="'40%'" :height="16" />
            <Skeleton :width="'65%'" :height="12" />
          </li>
        </ul>
        <EmptyState
          v-else-if="rowCount === 0"
          :title="t('gotocc.taskCenter.empty.title')"
          :description="t(`gotocc.taskCenter.empty.${tab}`)"
        />
        <div v-else :class="{ 'opacity-60 transition-opacity': listLoading }">
          <VideoTaskList v-if="tab === 'video'" :tasks="videoTasks" @preview="previewTaskId = $event" />
          <ImageTaskList v-else :tasks="imageTasks" />
        </div>

        <div v-if="listLoaded && !listError && (offset > 0 || hasMore)" class="flex items-center justify-end gap-2 border-t border-gray-100 px-5 py-3 dark:border-dark-700">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="offset === 0 || listLoading" @click="changePage(-1)">{{ t('gotocc.taskCenter.previous') }}</button>
          <span class="text-xs tabular-nums text-gray-500 dark:text-dark-400">{{ t('gotocc.taskCenter.page', { page: offset / PAGE_SIZE + 1 }) }}</span>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="!hasMore || listLoading" @click="changePage(1)">{{ t('gotocc.taskCenter.next') }}</button>
        </div>
      </div>
    </div>

    <VideoPreviewDialog :task-id="previewTaskId" @close="previewTaskId = null" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useDocumentVisibility } from '@vueuse/core'
import AppLayout from '@/components/layout/AppLayout.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import MetricCard from '@/components/gotocc/dashboard/MetricCard.vue'
import { formatCostAuto } from '@/components/gotocc/dashboard/format'
import type { MetricTone } from '@/components/gotocc/dashboard/tones'
import { vSegmented } from '@/components/gotocc/vSegmented'
import ImageTaskList from '@/components/gotocc/tasks/ImageTaskList.vue'
import TaskFilterBar from '@/components/gotocc/tasks/TaskFilterBar.vue'
import VideoPreviewDialog from '@/components/gotocc/tasks/VideoPreviewDialog.vue'
import VideoTaskList from '@/components/gotocc/tasks/VideoTaskList.vue'
import { dayRangeToUnix, defaultDayRange } from '@/components/gotocc/tasks/taskFormat'
import type { AsyncImageTask } from '@/api/asyncImage'
import { getUserImageTaskFilters, listUserImageTasks } from '@/api/imageTasks'
import {
  getImageTaskSummary,
  getVideoTaskModels,
  getVideoTaskSummary,
  listVideoTasks,
  type ImageTaskSummary,
  type TaskCenterRange,
  type TaskSummary,
  type VideoTask,
} from '@/api/taskCenter'
import { useAppStore } from '@/stores'
import config from '../../../../task-center.json'

// 任务中心：视频与图片两个页签共用日期和模型筛选、统计卡与分页列表。页签写入地址栏 ?tab=。
// 每页条数、默认天数和处理中任务的刷新间隔来自产品根目录 task-center.json。
type Tab = 'video' | 'image'

const PAGE_SIZE = config.page_size
const SKELETON_ROWS = 5

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const visibility = useDocumentVisibility()

const tab = computed<Tab>(() => (route.query.tab === 'image' ? 'image' : 'video'))
const tabs = computed(() => [
  { value: 'video' as Tab, label: t('gotocc.taskCenter.tabs.video') },
  { value: 'image' as Tab, label: t('gotocc.taskCenter.tabs.image') },
])

// draft 是筛选条上正在编辑的值，applied 是当前列表和统计所用的条件。
const draft = reactive({ ...defaultDayRange(config.default_range_days), model: null as string | null })
const applied = ref({ ...draft })
const range = computed<TaskCenterRange>(() => ({ ...dayRangeToUnix(applied.value.start, applied.value.end), model: applied.value.model ?? '' }))

const models = ref<string[]>([])
const summary = ref<TaskSummary | ImageTaskSummary | null>(null)
const summaryLoading = ref(false)
const summaryError = ref('')
const videoTasks = ref<VideoTask[]>([])
const imageTasks = ref<AsyncImageTask[]>([])
const offset = ref(0)
const hasMore = ref(false)
const listLoading = ref(false)
const listLoaded = ref(false)
const listError = ref('')
const previewTaskId = ref<string | null>(null)

const rowCount = computed(() => (tab.value === 'video' ? videoTasks.value.length : imageTasks.value.length))
const hasProcessing = computed(() => (tab.value === 'video' ? videoTasks.value : imageTasks.value).some(task => task.status === 'processing'))
const refreshing = computed(() => listLoading.value || summaryLoading.value)

// 统计卡。统计未取到时卡片显示骨架，取数失败时显示「—」和原因；图片费用无法精确对应时显示「见使用记录」。
type CardIcon = InstanceType<typeof Icon>['$props']['name']
const cards = computed(() => {
  const current = summary.value
  const failure = summaryError.value
  const card = (key: string, icon: CardIcon, tone: MetricTone, value: number | string, hint: string, format?: (value: number) => string) => ({
    key,
    props: {
      icon, tone, format,
      label: t(`gotocc.taskCenter.cards.${key}`),
      value: failure ? '—' : value,
      hint: failure ? t('gotocc.taskCenter.cards.loadFailed', { message: failure }) : hint,
    },
  })
  const imageCostHidden = tab.value === 'image' && current !== null && !(current as ImageTaskSummary).cost_available
  return [
    card('total', 'clipboard', 'tasks', current?.total ?? 0, t('gotocc.taskCenter.cards.inRange')),
    card('completed', 'checkCircle', 'completed', current?.completed ?? 0, t('gotocc.taskCenter.cards.failedCount', { count: current?.failed ?? 0 })),
    card('processing', 'clock', 'processing', current?.processing ?? 0, current?.processing
      ? t('gotocc.taskCenter.cards.autoRefresh', { seconds: config.processing_poll_interval_ms / 1000 })
      : t('gotocc.taskCenter.cards.noProcessing')),
    imageCostHidden
      ? card('cost', 'dollar', 'cost', t('gotocc.taskCenter.cards.seeUsage'), t('gotocc.taskCenter.cards.costUnavailable'))
      : card('cost', 'dollar', 'cost', current?.cost ?? 0, t('gotocc.taskCenter.cards.costHint'), formatCostAuto),
  ]
})

function errorText(error: unknown): string {
  return (error as { message?: string }).message || t('gotocc.taskCenter.loadFailed')
}

// 用递增序号丢弃过期响应：切换页签、翻页或重复刷新时只保留最后一次结果。
let listSeq = 0
let summarySeq = 0
let modelsSeq = 0

async function loadList(silent = false) {
  const seq = ++listSeq
  const currentTab = tab.value
  if (!silent) listLoading.value = true
  listError.value = ''
  try {
    if (currentTab === 'video') {
      const page = await listVideoTasks(range.value, PAGE_SIZE, offset.value)
      if (seq !== listSeq) return
      videoTasks.value = page.data
      hasMore.value = page.has_more
    } else {
      const page = await listUserImageTasks({ model: range.value.model, since: range.value.since, until: range.value.until, limit: PAGE_SIZE, offset: offset.value })
      if (seq !== listSeq) return
      imageTasks.value = page.data
      hasMore.value = page.has_more
    }
    listLoaded.value = true
  } catch (error) {
    if (seq === listSeq) listError.value = errorText(error)
  } finally {
    if (seq === listSeq) listLoading.value = false
    schedulePoll()
  }
}

async function loadSummary(silent = false) {
  const seq = ++summarySeq
  if (!silent) summaryLoading.value = true
  try {
    const result = tab.value === 'video' ? await getVideoTaskSummary(range.value) : await getImageTaskSummary(range.value)
    if (seq !== summarySeq) return
    summary.value = result
    summaryError.value = ''
  } catch (error) {
    if (seq === summarySeq) summaryError.value = errorText(error)
  } finally {
    if (seq === summarySeq) summaryLoading.value = false
  }
}

async function loadModels() {
  const seq = ++modelsSeq
  try {
    const result = tab.value === 'video' ? await getVideoTaskModels() : (await getUserImageTaskFilters()).models
    if (seq === modelsSeq) models.value = result
  } catch (error) {
    if (seq === modelsSeq) appStore.showError(t('gotocc.taskCenter.filters.modelsFailed', { message: errorText(error) }))
  }
}

function refresh() {
  void loadList()
  void loadSummary()
}

function query() {
  applied.value = { ...draft }
  offset.value = 0
  refresh()
}

function reset() {
  Object.assign(draft, defaultDayRange(config.default_range_days), { model: null })
  query()
}

function changePage(step: number) {
  offset.value += step * PAGE_SIZE
  void loadList()
}

function selectTab(next: Tab) {
  if (next !== tab.value) void router.replace({ query: { ...route.query, tab: next } })
}

// 切换页签时模型选项随之更换，模型条件清空，列表回到第一页。
watch(tab, () => {
  draft.model = null
  applied.value = { ...applied.value, model: null }
  offset.value = 0
  listLoaded.value = false
  summary.value = null
  summaryError.value = ''
  models.value = []
  void loadModels()
  refresh()
}, { immediate: true })

// 当前页有处理中任务时定时重取当前页和统计；页面隐藏时暂停，回到页面后继续。
let pollTimer: ReturnType<typeof setTimeout> | undefined

function schedulePoll() {
  clearTimeout(pollTimer)
  pollTimer = undefined
  if (!hasProcessing.value || listError.value || visibility.value !== 'visible') return
  pollTimer = setTimeout(() => {
    void loadList(true)
    void loadSummary(true)
  }, config.processing_poll_interval_ms)
}

watch(visibility, schedulePoll)
onBeforeUnmount(() => clearTimeout(pollTimer))
</script>

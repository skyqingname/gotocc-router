<!-- Derived from TokenFlux/TokenRouter e74bba115 (v0.7.3) frontend/src/components/announcements/AnnouncementBell.vue, AnnouncementArticle.vue and styles/announcement-markdown.css. Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation. -->
<template>
  <div v-if="showAnnouncements">
    <!-- 顶栏入口：铃铛动效图标加文字（窄顶栏只留铃铛），有未读时铃铛变主色并在右上角挂红点 -->
    <button type="button" class="header-nav-link" :title="t('announcements.title')" :aria-label="t('announcements.title')" @click="open = true">
      <span class="relative flex" :class="{ 'text-primary-600 dark:text-primary-400': unreadCount > 0 }">
        <BellIcon />
        <span v-if="unreadCount > 0" class="absolute -right-0.5 -top-0.5 h-2 w-2 rounded-full bg-rose-500 ring-2 ring-white dark:ring-dark-900" aria-hidden="true"></span>
      </span>
      <span class="header-nav-label">{{ t('announcements.title') }}</span>
    </button>

    <!-- 列表和详情在同一个固定高度的弹窗里：详情从右侧盖住列表，列表留在下面保留滚动位置 -->
    <Teleport to="body">
      <Transition name="modal" @after-leave="selected = null">
        <div v-if="open" class="modal-overlay z-[100]" @click.self="open = false">
          <section
            ref="panelRef"
            role="dialog"
            aria-modal="true"
            :aria-label="t('announcements.title')"
            tabindex="-1"
            class="gc-announcement-panel modal-content relative h-[36rem] max-w-[640px] overflow-hidden"
          >
            <div ref="listRef" class="gc-announcement-list flex min-h-0 flex-1 flex-col" :inert="selected ? true : undefined">
              <header class="gc-announcement-bar">
                <div class="flex min-w-0 items-center gap-2.5">
                  <h2 class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ t('announcements.title') }}</h2>
                  <span v-if="unreadCount > 0" class="shrink-0 rounded-md bg-primary-50 px-2 py-1 text-xs font-medium leading-none text-primary-700 dark:bg-primary-500/15 dark:text-primary-300">
                    {{ t('gotocc.dashboard.announcements.unread', { count: unreadCount }) }}
                  </span>
                </div>
                <div class="flex shrink-0 items-center gap-1">
                  <button v-if="unreadCount > 0" v-support-readonly type="button" class="gc-announcement-action px-2.5" :disabled="loading" @click="markAllAsRead">
                    <Icon name="checkCircle" size="sm" />
                    <span>{{ t('announcements.markAllRead') }}</span>
                  </button>
                  <button type="button" class="gc-announcement-action -mr-2 w-8 justify-center" :aria-label="t('common.close')" @click="open = false">
                    <Icon name="x" size="sm" />
                  </button>
                </div>
              </header>

              <div class="min-h-0 flex-1 overflow-y-auto">
                <!-- 首次加载画与条目同样几何的骨架行；刷新或全部已读时保留已加载的条目 -->
                <div v-if="loading && rows.length === 0" class="p-2" role="status" :aria-label="t('common.loading')" aria-busy="true">
                  <div v-for="row in LOADING_ROWS" :key="row" class="space-y-1 px-4 py-3" aria-hidden="true">
                    <div class="flex h-5 items-center justify-between gap-3">
                      <Skeleton width="45%" :height="14" />
                      <Skeleton :width="48" :height="12" />
                    </div>
                    <div class="flex h-5 items-center">
                      <Skeleton width="75%" :height="12" />
                    </div>
                  </div>
                </div>

                <ul v-else-if="rows.length > 0" class="p-2">
                  <li v-for="row in rows" :key="row.announcement.id">
                    <button
                      type="button"
                      class="gc-announcement-row group block w-full px-4 py-3 text-left"
                      :data-announcement-id="row.announcement.id"
                      @click="openDetail(row.announcement)"
                    >
                      <span class="relative flex items-baseline gap-3">
                        <!-- 未读圆点挂在行左侧留白里，锚在这一行上：标题截断时会裁掉溢出内容 -->
                        <span v-if="!row.announcement.read_at" class="absolute -left-3 top-1/2 h-2 w-2 -translate-y-1/2 rounded-full bg-primary-500" aria-hidden="true"></span>
                        <span v-if="!row.announcement.read_at" class="sr-only">{{ t('announcements.unread') }}</span>
                        <span
                          class="min-w-0 flex-1 truncate text-sm leading-5 transition-colors group-hover:text-primary-700 dark:group-hover:text-primary-300"
                          :class="row.announcement.read_at ? 'font-medium text-gray-800 dark:text-dark-100' : 'font-semibold text-gray-900 dark:text-white'"
                        >{{ row.announcement.title }}</span>
                        <time :datetime="row.announcement.created_at" class="shrink-0 text-xs tabular-nums surface-muted">{{ formatAnnouncementRowDate(row.announcement.created_at) }}</time>
                      </span>
                      <span v-if="row.summary" class="mt-1 block truncate text-sm leading-5 surface-muted">{{ row.summary }}</span>
                    </button>
                  </li>
                </ul>

                <div v-else class="flex h-full flex-col items-center justify-center px-6 text-center">
                  <div class="flex h-10 w-10 items-center justify-center rounded-lg surface-inset surface-muted">
                    <Icon name="inbox" size="md" />
                  </div>
                  <p class="mt-3 text-sm font-medium text-gray-900 dark:text-white">{{ t('announcements.empty') }}</p>
                  <p class="mt-1 text-xs surface-muted">{{ t('announcements.emptyDescription') }}</p>
                </div>
              </div>
            </div>

            <Transition name="gc-announcement-detail">
              <div v-if="selected" class="gc-announcement-detail absolute inset-0 flex flex-col">
                <header class="gc-announcement-bar">
                  <button ref="backRef" type="button" class="gc-announcement-action -ml-2 px-2 text-sm" :aria-label="t('common.back')" @click="showList">
                    <Icon name="arrowLeft" size="sm" />
                    <span>{{ t('announcements.title') }}</span>
                  </button>
                  <button type="button" class="gc-announcement-action -mr-2 w-8 justify-center" :aria-label="t('common.close')" @click="open = false">
                    <Icon name="x" size="sm" />
                  </button>
                </header>
                <!-- 标题和发布时间固定在上方，只有正文滚动 -->
                <article class="flex min-h-0 flex-1 flex-col">
                  <div class="shrink-0 px-6 pt-5">
                    <h3 class="break-words text-lg font-semibold leading-7 text-gray-900 dark:text-white sm:text-xl">{{ selected.title }}</h3>
                    <time :datetime="selected.created_at" class="mt-1 block text-xs tabular-nums surface-muted">{{ formatDateTimeToMinute(selected.created_at) }}</time>
                  </div>
                  <div class="gc-announcement-scroll min-h-0 flex-1 overflow-y-auto px-6 pb-6 pt-4">
                    <div class="gc-announcement-body text-link-content" v-html="renderAnnouncementHtml(selected.content)"></div>
                  </div>
                </article>
              </div>
            </Transition>
          </section>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { BellIcon } from '@/components/gotocc/navIcons'
import Skeleton from '@/components/common/Skeleton.vue'
import { useDialogFocus } from '@/composables/useDialogFocus'
import { useResellerCommunications } from '@/composables/useResellerCommunications'
import { supportReadonly as vSupportReadonly } from '@/directives/supportReadonly'
import { useAppStore } from '@/stores/app'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatDateTimeToMinute } from '@/utils/format'
import type { UserAnnouncement } from '@/types'
import { formatAnnouncementRowDate, renderAnnouncementHtml, summarizeAnnouncement } from './announcementDisplay'

// 顶栏公告：铃铛、列表与详情在同一个弹窗里，交互参照 TokenRouter（LGPL-3.0）。
// 入口是侧栏同款铃铛动效图标加文字（.header-nav-link），顶栏窄时只留铃铛。
// 未读计数、标记已读与全部已读沿用公告 store 的现有接口；站长客户按公告开关显示，管理员只读协助时「全部已读」不可用。
// 详情盖在列表上而不是另开弹窗：列表一直挂载、不可交互，返回时滚动位置不变，焦点回到打开的那一行。
// Esc 先从详情回到列表，再关闭弹窗。

// 首次加载时画的骨架行数。
const LOADING_ROWS = 4

const { t } = useI18n()
const appStore = useAppStore()
const announcementStore = useAnnouncementStore()
const { announcements, loading, unreadCount } = storeToRefs(announcementStore)
const { showAnnouncements } = useResellerCommunications()

const open = ref(false)
// 弹窗关闭动画结束后才清空，关闭过程中保留当前内容。
const selected = ref<UserAnnouncement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const listRef = ref<HTMLElement | null>(null)
const backRef = ref<HTMLButtonElement | null>(null)

// 摘要按内容解析一次，已读状态变化时复用。
const rows = computed(() => announcements.value.map((announcement) => ({ announcement, summary: summarizeAnnouncement(announcement.content) })))

const onEscape = () => {
  if (selected.value) showList()
  else open.value = false
}

useDialogFocus(() => open.value, panelRef, onEscape)

const openDetail = async (announcement: UserAnnouncement) => {
  selected.value = announcement
  // 列表变为不可交互，点击的行失去焦点，把焦点交给返回键。
  void nextTick(() => backRef.value?.focus({ preventScroll: true }))
  if (!announcement.read_at && (await announcementStore.markAsRead(announcement.id)) === false) {
    appStore.showError(t('common.unknownError'))
  }
}

// 回到列表后按公告 id 找回原来那一行：刷新可能已经替换了行元素。
const showList = () => {
  const id = selected.value!.id
  selected.value = null
  void nextTick(() => listRef.value?.querySelector<HTMLElement>(`[data-announcement-id="${id}"]`)?.focus({ preventScroll: true }))
}

const markAllAsRead = async () => {
  try {
    await announcementStore.markAllAsRead()
    appStore.showSuccess(t('announcements.allMarkedAsRead'))
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
  }
}
</script>

<style scoped>
/* 管理员把卡片调成半透明时，弹窗后面的页面先模糊，文字不会和背后的内容叠在一起。 */
.gc-announcement-panel {
  -webkit-backdrop-filter: blur(var(--glass-blur));
  backdrop-filter: blur(var(--glass-blur));
}

/* 列表与详情共用 56px 高的标题栏，底色与边线同站内其它弹窗的标题栏。 */
.gc-announcement-bar {
  @apply flex h-14 shrink-0 items-center justify-between gap-3 border-b px-6;
  background-color: var(--surface-soft);
  border-color: var(--surface-edge);
}

.gc-announcement-action {
  @apply inline-flex h-8 shrink-0 items-center gap-1.5 rounded-lg text-xs font-medium;
  @apply focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50;
  @apply disabled:cursor-not-allowed disabled:opacity-50;
  color: var(--surface-muted);
  transition:
    background-color var(--motion-fast) var(--motion-ease),
    color var(--motion-fast) var(--motion-ease);
}

.gc-announcement-action:hover:not(:disabled) {
  background-color: var(--surface-hover);
  color: var(--surface-text);
}

/* 行内缩在列表留白里，圆角悬停底色贴着弹窗圆角；文字起点与标题栏同为 24px。 */
.gc-announcement-row {
  border-radius: var(--surface-radius-item);
  transition: background-color var(--motion-fast) var(--motion-ease);
}

.gc-announcement-row:hover {
  background-color: var(--surface-hover);
}

.gc-announcement-row:focus-visible {
  @apply outline-none ring-2 ring-inset ring-primary-500/50;
}

/* 详情完全盖住列表后再隐藏列表（卡片半透明时列表不会透出来）；返回时立即恢复，隐藏不影响滚动位置。 */
.gc-announcement-list {
  transition: visibility 0s;
}

.gc-announcement-list[inert] {
  visibility: hidden;
  transition-delay: var(--motion-layout);
}

/* 详情层沿用弹窗自身的底色与高光。 */
.gc-announcement-detail {
  background: inherit;
}

/* 详情从右侧短距离滑入淡入，返回时反向；减少动态效果时位移归零、时长压到 1ms。 */
.gc-announcement-detail-enter-active {
  transition:
    opacity var(--motion-layout) var(--motion-ease),
    transform var(--motion-layout) var(--motion-ease);
}

.gc-announcement-detail-leave-active {
  transition:
    opacity var(--motion-exit) var(--motion-ease-exit),
    transform var(--motion-exit) var(--motion-ease-exit);
}

.gc-announcement-detail-enter-from,
.gc-announcement-detail-leave-to {
  opacity: 0;
  transform: translateX(calc(var(--motion-shift) * 4));
}

/* 正文上下边缘渐隐，滚动的内容在固定标题下方淡出。 */
.gc-announcement-scroll {
  mask-image: linear-gradient(to bottom, transparent, black 1rem, black calc(100% - 1.5rem), transparent);
}

/* 公告正文（管理员写的 Markdown/HTML）：紧凑字号，颜色取暖色外壳变量；链接样式来自 .text-link-content。 */
.gc-announcement-body {
  @apply min-w-0 text-sm leading-6;
  color: var(--surface-text);
  overflow-wrap: anywhere;
}

.gc-announcement-body > :deep(:first-child) {
  @apply mt-0;
}

.gc-announcement-body > :deep(:last-child) {
  @apply mb-0;
}

.gc-announcement-body :deep(:is(h1, h2, h3, h4)) {
  @apply font-semibold;
  color: var(--surface-text);
}

.gc-announcement-body :deep(h1) {
  @apply mb-4 mt-6 text-xl;
}

.gc-announcement-body :deep(h2) {
  @apply mb-3 mt-5 text-lg;
}

.gc-announcement-body :deep(h3) {
  @apply mb-2 mt-4 text-base;
}

.gc-announcement-body :deep(h4) {
  @apply mb-2 mt-4 text-sm;
}

.gc-announcement-body :deep(p) {
  @apply mb-3;
}

.gc-announcement-body :deep(a) {
  @apply font-medium;
}

.gc-announcement-body :deep(:is(ul, ol)) {
  @apply mb-3 space-y-1 pl-5;
}

.gc-announcement-body :deep(ul) {
  @apply list-disc;
}

.gc-announcement-body :deep(ol) {
  @apply list-decimal;
}

.gc-announcement-body :deep(li) {
  @apply pl-1;
}

.gc-announcement-body :deep(li::marker) {
  color: var(--surface-muted);
}

.gc-announcement-body :deep(li > :is(ul, ol)) {
  @apply mb-0 mt-1;
}

.gc-announcement-body :deep(blockquote) {
  @apply my-4 border-l-2 border-primary-500 py-2 pl-4 pr-3;
  background-color: var(--surface-soft);
  color: var(--surface-muted);
}

.dark .gc-announcement-body :deep(blockquote) {
  border-color: theme('colors.primary.400');
}

.gc-announcement-body :deep(blockquote > :last-child) {
  @apply mb-0;
}

.gc-announcement-body :deep(code) {
  @apply rounded px-1.5 py-0.5 font-mono text-[13px];
  background-color: var(--surface-soft);
}

.gc-announcement-body :deep(pre) {
  @apply my-4 overflow-x-auto rounded-xl border p-4;
  border-color: var(--surface-edge);
  background-color: var(--surface-soft);
}

.gc-announcement-body :deep(pre code) {
  @apply bg-transparent p-0;
}

.gc-announcement-body :deep(hr) {
  @apply my-5;
  border-color: var(--surface-edge);
}

/* 表格用分离边框保留圆角：单元格画内部网格，表格画外框。 */
.gc-announcement-body :deep(table) {
  @apply mb-4 w-full border-separate border-spacing-0 overflow-hidden rounded-lg border;
  border-color: var(--surface-edge);
}

.gc-announcement-body :deep(:is(th, td)) {
  @apply border-b border-r px-3 py-2 text-left;
  border-color: var(--surface-edge);
}

.gc-announcement-body :deep(:is(th, td):last-child) {
  @apply border-r-0;
}

.gc-announcement-body :deep(tr:last-child td) {
  @apply border-b-0;
}

.gc-announcement-body :deep(th) {
  @apply font-semibold;
  background-color: var(--surface-soft);
}

.gc-announcement-body :deep(img) {
  @apply my-4 h-auto max-w-full rounded-lg border;
  border-color: var(--surface-edge);
}

.gc-announcement-body :deep(strong) {
  @apply font-semibold;
}
</style>

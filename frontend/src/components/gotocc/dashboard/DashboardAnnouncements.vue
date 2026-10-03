<template>
  <div class="card overflow-hidden">
    <div class="flex items-center justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
      <h2 class="flex min-w-0 items-center gap-2 text-base font-semibold text-gray-900 dark:text-white">
        <Icon name="bell" size="md" class="shrink-0 text-primary-600 dark:text-primary-400" />
        <span class="truncate">{{ t('gotocc.dashboard.announcements.title') }}</span>
      </h2>
      <span
        class="inline-flex shrink-0 items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium"
        :class="unreadCount > 0 ? 'bg-primary-100 text-primary-800 dark:bg-primary-900/30 dark:text-primary-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300'"
      >
        <span class="h-1.5 w-1.5 rounded-full" :class="unreadCount > 0 ? 'bg-primary-500 dark:bg-primary-400' : 'bg-gray-400 dark:bg-dark-400'"></span>
        {{ t('gotocc.dashboard.announcements.unread', { count: unreadCount }) }}
      </span>
    </div>

    <div class="p-4">
      <div v-if="loading && items.length === 0" class="space-y-5 py-2" aria-busy="true">
        <div v-for="row in 3" :key="row" class="flex items-start gap-4" aria-hidden="true">
          <Skeleton variant="circle" :width="12" :height="12" class="shrink-0" />
          <div class="min-w-0 flex-1 space-y-3">
            <Skeleton width="65%" :height="16" />
            <Skeleton width="90%" :height="12" />
          </div>
        </div>
      </div>

      <p v-else-if="items.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('gotocc.dashboard.announcements.empty') }}</p>

      <!-- 时间线：未读节点为品牌色并带呼吸环，节点之间用竖线连接 -->
      <ol v-else class="space-y-1">
        <li v-for="(item, index) in items" :key="item.announcement.id" class="relative pl-7">
          <span v-if="index < items.length - 1" aria-hidden="true" class="absolute bottom-[-1.625rem] left-[5px] top-[1.375rem] w-px bg-gray-200 dark:bg-dark-600"></span>
          <span v-if="!item.announcement.read_at" aria-hidden="true" class="absolute left-0 top-4 h-3 w-3 animate-ping rounded-full bg-primary-400 opacity-75 motion-reduce:animate-none"></span>
          <span
            aria-hidden="true"
            class="absolute left-0 top-4 z-10 h-3 w-3 rounded-full border-[3px]"
            :class="item.announcement.read_at ? 'border-gray-200 bg-gray-400 dark:border-dark-700 dark:bg-dark-500' : 'border-primary-100 bg-primary-500 dark:border-dark-800 dark:bg-primary-400'"
          ></span>
          <button
            type="button"
            class="group flex w-full items-start gap-3 rounded-lg px-3 py-3 text-left transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/60"
            @click="openAnnouncement(item.announcement)"
          >
            <div class="min-w-0 flex-1">
              <h3 class="line-clamp-1 break-words text-sm font-semibold text-gray-900 group-hover:text-primary-700 dark:text-white dark:group-hover:text-primary-300">{{ item.announcement.title }}</h3>
              <p class="mt-1.5 line-clamp-2 break-words text-sm leading-5 text-gray-500 dark:text-dark-300">{{ item.summary || t('gotocc.dashboard.announcements.noSummary') }}</p>
              <time :datetime="item.announcement.created_at" :title="formatDateTime(item.announcement.created_at)" class="mt-2 block text-xs text-gray-400 dark:text-dark-400">{{ formatAnnouncementDate(item.announcement.created_at) }}</time>
            </div>
            <Icon name="chevronRight" size="sm" class="mt-1 shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5 group-hover:text-primary-500 dark:text-dark-500" />
          </button>
        </li>
      </ol>
    </div>

    <AnnouncementPopup :announcement="selected" preview @close="selected = null" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatDate, formatDateTime } from '@/utils/format'
import type { UserAnnouncement } from '@/types'

// 最新公告时间线，结构参照 TokenRouter（LGPL-3.0）。按发布时间倒序展示最近 5 条，点开即标记已读。
const MAX_ITEMS = 5

const { t } = useI18n()
const announcementStore = useAnnouncementStore()
const { announcements, loading, unreadCount } = storeToRefs(announcementStore)
const selected = ref<UserAnnouncement | null>(null)

// 同年省略年份，跨年补充年份。
const formatAnnouncementDate = (createdAt: string): string => {
  const date = new Date(createdAt)
  const options: Intl.DateTimeFormatOptions = { month: 'long', day: 'numeric' }
  if (date.getFullYear() !== new Date().getFullYear()) options.year = 'numeric'
  return formatDate(date, options)
}

// 公告正文支持 Markdown 和 HTML，走与弹窗相同的安全解析后提取纯文本摘要。
const createSummary = (content: string): string => {
  const html = DOMPurify.sanitize(marked.parse(content, { breaks: true, gfm: true }) as string)
  return (new DOMParser().parseFromString(html, 'text/html').body.textContent ?? '').replace(/\s+/g, ' ').trim()
}

// 接口列表未读优先，时间线恢复为发布时间倒序。
const items = computed(() => [...announcements.value]
  .sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at) || b.id - a.id)
  .slice(0, MAX_ITEMS)
  .map((announcement) => ({ announcement, summary: createSummary(announcement.content) })))

const openAnnouncement = (announcement: UserAnnouncement) => {
  selected.value = announcement
  if (!announcement.read_at) void announcementStore.markAsRead(announcement.id)
}
</script>

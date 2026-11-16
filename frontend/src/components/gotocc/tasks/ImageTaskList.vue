<!-- Row and thumbnail structure adapted from TokenFlux/TokenRouter e74bba115fc2cfd77e97a2a22a15dad0605d64d1 (frontend/src/components/creative/CreativeRunHistory.vue). Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation. -->
<template>
  <!-- 图片任务行：状态、模型、张数、提示词、时间与耗时在左，缩略图和下载在右；窄屏改为上下两段。 -->
  <ul class="divide-y divide-gray-100 dark:divide-dark-700">
    <li v-for="task in tasks" :key="task.task_id" class="task-row">
      <div class="min-w-0 flex-1 space-y-1.5">
        <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1.5">
          <TaskStatusBadge :status="task.status as TaskCenterStatus" />
          <span class="flex min-w-0 items-center gap-1.5">
            <ModelIcon :model="task.model || ''" size="16px" />
            <span class="truncate text-sm font-medium text-gray-900 dark:text-white" :title="task.model">{{ task.model }}</span>
          </span>
          <span class="task-chip">{{ t('gotocc.taskCenter.image.count', { actual: task.actual_images, requested: task.requested_images }) }}</span>
        </div>
        <p class="line-clamp-2 break-words text-sm text-gray-700 dark:text-dark-200" :title="task.prompt_preview">
          {{ task.prompt_preview || t('gotocc.taskCenter.image.noPrompt') }}
        </p>
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs tabular-nums text-gray-500 dark:text-dark-400">
          <span :title="absoluteTime(task.created_at)">{{ relativeTime(task.created_at) }}</span>
          <span v-if="task.status !== 'processing' && task.completed_at" class="inline-flex items-center gap-1" :title="t('gotocc.taskCenter.elapsed')">
            <Icon name="clock" size="xs" />
            {{ formatElapsed(task.created_at, task.completed_at) }}
          </span>
        </div>
        <p v-if="task.status === 'failed' && task.error?.message" class="truncate text-xs text-red-600 dark:text-red-400" :title="task.error.message">
          {{ t('gotocc.taskCenter.video.failedReason', { message: task.error.message }) }}
        </p>
        <!-- 已完成但图片读不到时按上游文案说明原因，不显示破图 -->
        <p v-if="task.status === 'completed' && task.read_status !== 'readable'" class="text-xs text-amber-600 dark:text-amber-400">
          {{ t('asyncImage.readStatus.' + task.read_status) }}
        </p>
      </div>

      <div v-if="imageUrls(task).length" class="flex shrink-0 items-center justify-between gap-4 sm:justify-end">
        <div class="flex items-center gap-1.5">
          <button
            v-for="(url, index) in imageUrls(task).slice(0, MAX_THUMBNAILS)"
            :key="url"
            type="button"
            class="task-thumb group"
            :aria-label="t('gotocc.taskCenter.image.previewImage', { index: index + 1 })"
            @click="previewUrl = url"
          >
            <img :src="url" alt="" class="h-full w-full object-cover" loading="lazy" />
            <span class="task-thumb-overlay"><Icon name="eye" size="sm" /></span>
          </button>
          <span v-if="imageUrls(task).length > MAX_THUMBNAILS" class="text-xs tabular-nums text-gray-500 dark:text-dark-400">
            +{{ imageUrls(task).length - MAX_THUMBNAILS }}
          </span>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="downloading.has(task.task_id)" @click="download(task.task_id)">
          <span v-if="downloading.has(task.task_id)" class="h-3.5 w-3.5 animate-spin rounded-full border-[1.5px] border-current border-t-transparent motion-reduce:animate-none" aria-hidden="true"></span>
          <Icon v-else name="download" size="xs" />
          {{ t('gotocc.taskCenter.image.download') }}
        </button>
      </div>
    </li>
  </ul>

  <BaseDialog :show="previewUrl !== ''" :title="t('gotocc.taskCenter.image.previewTitle')" width="wide" @close="previewUrl = ''">
    <img :src="previewUrl" alt="" class="mx-auto max-h-[70vh] w-auto rounded-lg object-contain" />
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { AsyncImageDownloadValidationError, saveAsyncImageBlob, type AsyncImageTask } from '@/api/asyncImage'
import { downloadUserImageTaskZip } from '@/api/imageTasks'
import type { TaskCenterStatus } from '@/api/taskCenter'
import { useAppStore } from '@/stores'
import TaskStatusBadge from './TaskStatusBadge.vue'
import { absoluteTime, formatElapsed, relativeTime } from './taskFormat'

// 每行最多显示的缩略图张数，其余以 +N 表示。
const MAX_THUMBNAILS = 4

defineProps<{ tasks: AsyncImageTask[] }>()

const { t } = useI18n()
const appStore = useAppStore()
const previewUrl = ref('')
const downloading = reactive(new Set<string>())

// 列表返回的签名地址可直接展示；只有已完成且可读取的任务才有图片。
function imageUrls(task: AsyncImageTask): string[] {
  if (task.status !== 'completed' || task.read_status !== 'readable') return []
  return (task.result?.data ?? []).map(item => item.url).filter((url): url is string => Boolean(url))
}

// 下载沿用控制台已有的 ZIP 接口，归档校验不通过时不保存。
async function download(taskId: string) {
  downloading.add(taskId)
  try {
    const blob = await downloadUserImageTaskZip(taskId)
    saveAsyncImageBlob(blob, `${taskId}.zip`)
  } catch (error) {
    appStore.showError(error instanceof AsyncImageDownloadValidationError
      ? t('asyncImage.errors.invalidArchive')
      : (error as { message?: string }).message || t('asyncImage.errors.download'))
  } finally {
    downloading.delete(taskId)
  }
}
</script>

<style scoped>
.task-row {
  @apply flex flex-col gap-3 px-5 py-4 transition-colors sm:flex-row sm:items-center;
  @apply hover:bg-primary-50/50 dark:hover:bg-primary-500/5;
}

.task-chip {
  @apply inline-flex shrink-0 items-center rounded-md bg-primary-50 px-1.5 py-0.5 text-xs text-primary-700;
  @apply dark:bg-primary-500/10 dark:text-primary-300;
}

.task-thumb {
  @apply relative h-12 w-12 shrink-0 overflow-hidden rounded-md border border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-800;
  @apply focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500;
}

/* 悬停或键盘聚焦时显示查看按钮；触屏设备没有悬停，常显。 */
.task-thumb-overlay {
  @apply absolute inset-0 flex items-center justify-center bg-primary-900/45 text-white opacity-0 transition-opacity;
}

.task-thumb:hover .task-thumb-overlay,
.task-thumb:focus-visible .task-thumb-overlay {
  @apply opacity-100;
}

@media (hover: none) {
  .task-thumb-overlay {
    @apply bg-primary-900/25 opacity-100;
  }
}
</style>

<!-- Row structure adapted from TokenFlux/TokenRouter e74bba115fc2cfd77e97a2a22a15dad0605d64d1 (frontend/src/components/creative/CreativeRunHistory.vue). Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation. -->
<template>
  <!-- 视频任务行：状态、模型、规格、时间与耗时在左，费用和预览/下载在右；窄屏改为上下两段。 -->
  <ul class="divide-y divide-gray-100 dark:divide-dark-700">
    <!-- 上游受理前任务没有 id，行按顺序标识 -->
    <li v-for="(task, index) in tasks" :key="index" class="task-row">
      <div class="min-w-0 flex-1 space-y-1.5">
        <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1.5">
          <TaskStatusBadge :status="task.status" />
          <span class="flex min-w-0 items-center gap-1.5">
            <ModelIcon :model="task.model" size="16px" />
            <span class="truncate text-sm font-medium text-gray-900 dark:text-white" :title="task.model">{{ task.model }}</span>
          </span>
          <span class="task-chip">{{ t('gotocc.taskCenter.video.seconds', { count: task.seconds }) }}</span>
          <span v-if="task.resolution" class="task-chip">{{ task.resolution }}</span>
        </div>
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs tabular-nums text-gray-500 dark:text-dark-400">
          <span :title="absoluteTime(task.created_at)">{{ relativeTime(task.created_at) }}</span>
          <span v-if="task.status !== 'processing' && task.finished_at !== null" class="inline-flex items-center gap-1" :title="t('gotocc.taskCenter.elapsed')">
            <Icon name="clock" size="xs" />
            {{ formatElapsed(task.created_at, task.finished_at) }}
          </span>
        </div>
        <p v-if="task.status === 'failed' && task.error?.message" class="truncate text-xs text-red-600 dark:text-red-400" :title="task.error.message">
          {{ t('gotocc.taskCenter.video.failedReason', { message: task.error.message }) }}
        </p>
      </div>

      <div class="flex shrink-0 items-center justify-between gap-4 sm:justify-end">
        <div class="text-left tabular-nums sm:min-w-24 sm:text-right">
          <template v-if="task.status === 'completed'">
            <span v-if="task.cost === null" class="text-sm text-gray-500 dark:text-dark-400">{{ t('gotocc.taskCenter.video.settling') }}</span>
            <span v-else class="text-sm font-semibold text-gray-900 dark:text-white">{{ formatCostAuto(task.cost) }}</span>
          </template>
          <template v-else-if="task.status === 'failed'">
            <span class="block text-sm font-semibold text-gray-900 dark:text-white">{{ formatCostAuto(0) }}</span>
            <span class="block text-[11px] text-gray-500 dark:text-dark-400">{{ t('gotocc.taskCenter.video.notCharged') }}</span>
          </template>
          <span v-else class="text-sm text-gray-500 dark:text-dark-400">{{ t('gotocc.taskCenter.video.processingCost') }}</span>
        </div>
        <div v-if="task.content_available && task.id" class="flex items-center gap-2">
          <button type="button" class="btn btn-secondary btn-sm" @click="emit('preview', task.id)">
            <Icon name="play" size="xs" />
            {{ t('gotocc.taskCenter.video.preview') }}
          </button>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="downloading.has(task.id)" @click="download(task.id)">
            <span v-if="downloading.has(task.id)" class="h-3.5 w-3.5 animate-spin rounded-full border-[1.5px] border-current border-t-transparent motion-reduce:animate-none" aria-hidden="true"></span>
            <Icon v-else name="download" size="xs" />
            {{ t('gotocc.taskCenter.video.download') }}
          </button>
        </div>
      </div>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import type { VideoTask } from '@/api/taskCenter'
import { useAppStore } from '@/stores'
import { formatCostAuto } from '../dashboard/format'
import TaskStatusBadge from './TaskStatusBadge.vue'
import { absoluteTime, formatElapsed, relativeTime } from './taskFormat'
import { downloadVideoTask, videoContentErrorMessage } from './videoContent'

defineProps<{ tasks: VideoTask[] }>()
const emit = defineEmits<{ (event: 'preview', taskId: string): void }>()

const { t } = useI18n()
const appStore = useAppStore()
const downloading = reactive(new Set<string>())

async function download(taskId: string) {
  downloading.add(taskId)
  try {
    await downloadVideoTask(taskId)
  } catch (error) {
    appStore.showError(videoContentErrorMessage(error))
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
</style>

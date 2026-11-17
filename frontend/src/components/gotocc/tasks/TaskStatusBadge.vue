<!-- Status style adapted from TokenFlux/TokenRouter e74bba115fc2cfd77e97a2a22a15dad0605d64d1 (frontend/src/components/creative/CreativeRunHistory.vue). Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation. -->
<template>
  <!-- 状态：圆点加文字，处理中换成转圈。 -->
  <span class="task-status" :class="`task-status-${status}`">
    <span v-if="status === 'processing'" class="task-status-spinner" aria-hidden="true"></span>
    <span v-else class="h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true"></span>
    {{ t(`gotocc.taskCenter.status.${status}`) }}
  </span>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { TaskCenterStatus } from '@/api/taskCenter'

defineProps<{ status: TaskCenterStatus }>()

const { t } = useI18n()
</script>

<style scoped>
.task-status {
  @apply inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap text-xs font-medium;
}

.task-status-processing {
  @apply text-primary-700 dark:text-primary-400;
}

.task-status-completed {
  @apply text-emerald-600 dark:text-emerald-400;
}

.task-status-failed {
  @apply text-red-600 dark:text-red-400;
}

.task-status-spinner {
  @apply h-3 w-3 animate-spin rounded-full border-[1.5px] border-current border-t-transparent motion-reduce:animate-none;
}
</style>

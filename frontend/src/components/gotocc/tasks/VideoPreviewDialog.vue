<template>
  <!-- 视频预览：带登录态取回视频生成对象地址后播放，关闭时释放对象地址。 -->
  <BaseDialog :show="taskId !== null" :title="t('gotocc.taskCenter.video.previewTitle')" width="wide" @close="emit('close')">
    <div class="flex aspect-video w-full items-center justify-center overflow-hidden rounded-lg bg-black">
      <video v-if="objectUrl" :src="objectUrl" class="h-full w-full" controls autoplay playsinline></video>
      <p v-else-if="error" class="px-6 text-center text-sm text-red-300">{{ error }}</p>
      <span v-else class="flex items-center gap-2 text-sm text-gray-300">
        <span class="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent motion-reduce:animate-none" aria-hidden="true"></span>
        {{ t('gotocc.taskCenter.video.loadingVideo') }}
      </span>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { fetchVideoTaskContent } from '@/api/taskCenter'
import { videoContentErrorMessage } from './videoContent'

const props = defineProps<{ taskId: string | null }>()
const emit = defineEmits<{ (event: 'close'): void }>()

const { t } = useI18n()
const objectUrl = ref('')
const error = ref('')
// 用递增序号丢弃已关闭或已切换的请求结果。
let requestSeq = 0

function release() {
  if (objectUrl.value) URL.revokeObjectURL(objectUrl.value)
  objectUrl.value = ''
  error.value = ''
}

watch(() => props.taskId, async (taskId) => {
  const seq = ++requestSeq
  release()
  if (taskId === null) return
  try {
    const blob = await fetchVideoTaskContent(taskId)
    if (seq === requestSeq) objectUrl.value = URL.createObjectURL(blob)
  } catch (err) {
    if (seq === requestSeq) error.value = videoContentErrorMessage(err)
  }
})

onBeforeUnmount(() => {
  requestSeq++
  release()
})
</script>

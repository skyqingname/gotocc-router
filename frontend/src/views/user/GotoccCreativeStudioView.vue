<template>
  <StudioLayout>
    <div ref="stage" class="gotocc-creative relative h-full min-h-0 overflow-hidden" :class="historyOpen && 'history-open'" :style="{ '--submission-duration': `${config.motion.submission_duration_ms}ms`, '--submission-ease': config.motion.submission_ease }">
      <CreativeCanvas ref="canvas" class="absolute inset-0" :operation="form.operation" :resolve-output="studio.resolveOutput" :allowed-mimes="config.allowed_mime_types" @error="error = $event" />

      <div ref="settingsRoot" class="studio-settings absolute left-3 top-3 z-20">
        <div class="canvas-island rounded-xl p-1">
          <button type="button" class="canvas-tool-btn" :title="t('studioGotocc.settings')" :aria-label="t('studioGotocc.settings')" :aria-expanded="settingsOpen" @click="settingsOpen = !settingsOpen"><Icon name="cog" size="md" /></button>
        </div>
        <StudioTransition name="pop-float">
        <div v-if="settingsOpen" class="studio-settings-popover absolute left-0 top-12 w-72 space-y-4 rounded-xl border border-gray-200 bg-white p-4 shadow-lg dark:border-dark-600 dark:bg-dark-900">
          <label class="block"><span class="input-label">{{ t('asyncImage.filters.apiKey') }}</span><Select v-model="keyID" :options="keyOptions" :disabled="submitting || loadingModels" searchable="auto" :aria-label="t('asyncImage.filters.apiKey')" @change="studio.selectKey" /></label>
          <p v-if="!keyID || (!canSubmit && !loadingModels)" class="text-xs leading-5 text-gray-500">{{ !keyID ? t('studioGotocc.selectKey') : t('studioGotocc.readOnlyKey') }}</p>
          <label class="block"><span class="input-label">{{ t('asyncImage.create.count') }}</span><input v-model.number="form.n" type="number" min="1" :max="config.max_image_count" class="input w-full" /></label>
          <details v-if="isGPTImage2(form.model)" class="text-xs text-gray-500">
            <summary class="cursor-pointer">{{ t('studioGotocc.customSize') }}</summary>
            <div class="mt-3 grid grid-cols-2 gap-2">
              <label>{{ t('asyncImage.create.width') }}<input v-model.number="form.customWidth" type="number" class="input w-full" @change="form.size = 'custom'" /></label>
              <label>{{ t('asyncImage.create.height') }}<input v-model.number="form.customHeight" type="number" class="input w-full" @change="form.size = 'custom'" /></label>
            </div>
            <p v-if="customSizeError" class="mt-2 text-red-500">{{ customSizeError }}</p>
          </details>
          <button type="button" class="menu-item" @click="resetConfirm = true; settingsOpen = false"><Icon name="trash" size="sm" />{{ t('creative.canvas.reset') }}</button>
        </div>
        </StudioTransition>
      </div>

      <button ref="historyToggle" v-show="!historyOpen" class="studio-history-toggle canvas-island absolute right-3 top-3 z-20 flex h-[42px] w-[42px] items-center justify-center rounded-xl" :class="historyBumping && 'studio-history-bump'" type="button" :aria-expanded="historyOpen" :aria-label="t('studioGotocc.history')" @click="historyOpen = true" @animationend.self="historyBumping = false">
        <Icon name="history" size="md" /><span v-if="activityCount" class="studio-count">{{ activityCount }}</span>
      </button>

      <StudioTransition name="fade">
      <div v-if="canvas?.sceneReady && canvas.imageCount === 0" class="studio-empty-guide pointer-events-none absolute inset-x-6 top-1/3 flex flex-col items-center text-center">
        <h1 class="text-base font-semibold text-gray-800 dark:text-gray-100">{{ t('creative.empty.title') }}</h1>
        <p class="mt-1 max-w-xl text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('creative.empty.description') }}</p>
        <div class="pointer-events-auto mt-4 flex max-w-xl flex-wrap justify-center gap-2">
          <button type="button" class="canvas-island studio-example" @click="canvas?.openFilePicker()"><Icon name="upload" size="sm" />{{ t('creative.empty.upload') }}</button>
          <button v-for="example in examples" :key="example" type="button" class="canvas-island studio-example" @click="composer?.fillPrompt(t(`creative.empty.examples.${example}.prompt`))"><Icon name="sparkles" size="sm" class="text-primary-500" />{{ t(`creative.empty.examples.${example}.label`) }}</button>
        </div>
      </div>
      </StudioTransition>

      <div v-support-readonly class="studio-composer-dock absolute bottom-4 z-30 -translate-x-1/2">
        <StudioComposer ref="composer" :studio="composerState" :guide="canvas?.guideKey" @generate="generate" />
      </div>

      <StudioTransition name="fade">
        <button v-if="historyOpen" type="button" class="absolute inset-0 z-30 bg-black/15 md:hidden" :aria-label="t('common.close')" @click="historyOpen = false" />
      </StudioTransition>
      <StudioTransition name="history-drawer">
      <aside v-if="historyOpen" class="studio-history absolute bottom-3 right-3 top-3 z-40 flex flex-col overflow-hidden rounded-2xl border border-gray-200/80 bg-white/95 shadow-lg backdrop-blur-xl dark:border-dark-700 dark:bg-dark-900/95">
        <div class="flex items-center justify-between gap-2 border-b border-gray-100 px-4 py-3 dark:border-dark-700"><h2 class="flex items-center gap-2 font-medium"><span ref="historyHeading" class="inline-flex" :class="historyBumping && 'studio-history-bump'" @animationend.self="historyBumping = false"><Icon name="history" size="sm" :animate-on-hover="false" /></span>{{ t('studioGotocc.history') }}</h2><div class="flex gap-1"><button type="button" class="canvas-tool-btn" :disabled="loading" :title="t('asyncImage.actions.refresh')" @click="studio.refreshHistory"><Icon name="refresh" size="sm" :class="loading && 'animate-spin'" /></button><button type="button" class="canvas-tool-btn" :title="t('common.close')" @click="historyOpen = false"><Icon name="x" size="sm" /></button></div></div>
        <div class="px-3 py-2"><Select v-model="status" :options="statusOptions" :searchable="false" :aria-label="t('asyncImage.filters.allStatuses')" @change="studio.filter" /></div>
        <div class="min-h-0 flex-1 space-y-3 overflow-y-auto px-3 pb-3">
          <p v-if="!tasks.length && !loading" class="px-4 py-12 text-center text-sm text-gray-400">{{ keyID ? t('asyncImage.list.empty') : t('studioGotocc.selectKey') }}</p>
          <article v-for="task in tasks" :key="task.task_id" class="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
            <button type="button" class="block w-full px-3 py-3 text-left" :aria-expanded="expandedTask === task.task_id" @click="expandedTask = expandedTask === task.task_id ? '' : task.task_id">
              <div class="flex items-center gap-2"><span class="min-w-0 flex-1 truncate text-xs font-semibold">{{ task.model }}</span><span class="text-[10px]" :class="statusColor(task.status)">{{ statusText(task.status) }}</span></div>
              <p class="mt-2 line-clamp-2 text-xs leading-5 text-gray-600 dark:text-gray-300">{{ task.prompt_preview }}</p>
              <span class="mt-2 block text-[10px] text-gray-400">{{ formatDate(task.created_at) }}</span>
            </button>
            <div v-if="task.result?.data?.some(item => item.url)" class="grid gap-px bg-gray-100 dark:bg-dark-700" :class="(task.result.data.length || 0) > 1 ? 'grid-cols-2' : 'grid-cols-1'">
              <img v-for="(item, index) in task.result.data" :key="item.object_id || index" :src="item.url" v-show="item.url" draggable="true" @dragstart="dragOutput($event, task, index)" :alt="task.prompt_preview" class="max-h-56 w-full object-cover" loading="lazy" />
            </div>
            <div v-if="task.status === 'failed'" class="px-3 pb-3 text-xs leading-5 text-red-500">{{ task.error?.message }}</div>
            <div v-if="task.status === 'processing'" class="flex items-center gap-2 px-3 pb-3 text-xs text-gray-400"><Icon name="refresh" size="sm" class="animate-spin" />{{ t('studioGotocc.processing') }}</div>
            <StudioTransition name="history-details">
            <div v-if="expandedTask === task.task_id" class="studio-history-details-grid">
            <div class="min-h-0 overflow-hidden">
            <div class="space-y-2 border-t border-gray-100 px-3 py-3 dark:border-dark-700">
              <p class="whitespace-pre-wrap break-words text-xs leading-5">{{ params.get(task.task_id)?.prompt || task.prompt_preview }}</p>
              <p v-if="params.has(task.task_id)" class="text-[10px] text-gray-400">{{ params.get(task.task_id)?.imageSize }} · {{ params.get(task.task_id)?.quality }} · {{ t('creative.operations.' + params.get(task.task_id)?.operation) }}</p>
              <p class="break-all font-mono text-[10px] text-gray-400">{{ task.task_id }}</p>
              <p v-if="task.status === 'completed' && task.actual_images != null && task.actual_images < (task.requested_images || 1)" class="text-xs text-amber-600">{{ t('studioGotocc.partial', { actual: task.actual_images, requested: task.requested_images }) }}</p>
              <button type="button" class="studio-link" @click="studio.reuseTask(task)">{{ t('studioGotocc.reuse') }}</button>
            </div>
            </div>
            </div>
            </StudioTransition>
            <div v-if="task.status === 'completed'" class="flex items-center gap-3 px-3 py-2.5"><button type="button" class="studio-link" :disabled="importingTask === task.task_id" @click="importResult(task)"><Icon name="plus" size="xs" />{{ importingTask === task.task_id ? t('common.loading') : t('studioGotocc.toCanvas') }}</button><button type="button" class="studio-link ml-auto" :title="t('asyncImage.actions.downloadAll')" @click="studio.downloadTask(task)"><Icon name="download" size="sm" /></button></div>
            <div v-if="task.status === 'failed'" class="flex justify-end px-3 pb-2"><button v-support-readonly type="button" class="studio-link text-red-500" @click="deleteTarget = task">{{ t('asyncImage.actions.delete') }}</button></div>
          </article>
        </div>
        <div v-if="tasks.length || offset" class="flex items-center justify-between border-t border-gray-100 px-3 py-2 text-xs dark:border-dark-700"><button type="button" class="studio-link" :disabled="loading || !offset" @click="studio.page(-1)">{{ t('asyncImage.list.previous') }}</button><span class="text-gray-400">{{ Math.floor(offset / config.history_page_size) + 1 }}</span><button type="button" class="studio-link" :disabled="loading || !hasMore" @click="studio.page(1)">{{ t('asyncImage.list.next') }}</button></div>
        <p class="border-t border-gray-100 px-4 py-2 text-[10px] leading-4 text-gray-400 dark:border-dark-700">{{ t('studioGotocc.storageHint') }}</p>
      </aside>
      </StudioTransition>
      <div v-if="submissionFlight" :key="submissionFlight.id" class="studio-submit-flight pointer-events-none absolute z-50 h-2 w-2 rounded-full bg-primary-500 ring-4 ring-primary-500/30" :style="submissionFlight.style" aria-hidden="true" @animationend.self="finishSubmissionFlight" />
    </div>
    <ConfirmDialog :show="resetConfirm" :title="t('creative.canvas.reset')" :message="t('studioGotocc.resetHint')" @confirm="resetCanvas" @cancel="resetConfirm = false" />
    <ConfirmDialog :show="!!deleteTarget" :title="t('asyncImage.actions.delete')" :message="t('asyncImage.delete.confirm', { taskId: deleteTarget?.task_id })" danger @confirm="deleteFailed" @cancel="deleteTarget = null" />
  </StudioLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { onClickOutside, useMediaQuery } from '@vueuse/core'
import StudioLayout from '@/components/gotocc/creative/StudioLayout.vue'
import StudioComposer from '@/components/gotocc/creative/StudioComposer.vue'
import Icon from '@/components/gotocc/creative/StudioIcon.vue'
import StudioTransition from '@/components/gotocc/creative/StudioTransition.vue'
import { createComposerState } from '@/components/gotocc/creative/composerState'
import Select from '@/components/common/Select.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import CreativeCanvas from '@/components/gotocc/creative/CreativeCanvas.vue'
import { useStudio } from '@/components/gotocc/creative/useStudio'
import { supportReadonly as vSupportReadonly } from '@/directives/supportReadonly'
import { isGPTImage2 } from '@/utils/asyncImageSize'
import { CREATIVE_OUTPUT_DRAG_MIME, serializeCreativeOutputDrag } from '@/components/gotocc/creative/drag'
import type { AsyncImageTask } from '@/api/asyncImage'
import '@/components/gotocc/creative/studio.css'

const { t } = useI18n()
const canvas = ref<InstanceType<typeof CreativeCanvas> | null>(null)
const studio = useStudio(async output => { await canvas.value?.placeOutput(output) })
const { config, keys, keyID, loading, loadingModels, submitting, tasks, form, status, offset, hasMore, error, activityCount, params, canSubmit, ready, customSizeError } = studio
const composerState = createComposerState(studio)
const composer = ref<InstanceType<typeof StudioComposer> | null>(null)
const historyOpen = ref(false)
const stage = ref<HTMLElement | null>(null)
const historyToggle = ref<HTMLElement | null>(null)
const historyHeading = ref<HTMLElement | null>(null)
const historyBumping = ref(false)
const submissionFlight = ref<{ id: number; style: Record<string, string> } | null>(null)
let flightID = 0
const settingsOpen = ref(false)
const settingsRoot = ref<HTMLElement | null>(null)
const resetConfirm = ref(false)
const deleteTarget = ref<AsyncImageTask | null>(null)
const expandedTask = ref('')
const importingTask = ref('')
const narrow = useMediaQuery('(max-width: 767px)')
const examples = ['poster', 'product', 'illustration'] as const
// The Key dropdown is teleported to body and stops click propagation there.
onClickOutside(settingsRoot, () => { settingsOpen.value = false }, { capture: false })
watch(submitting, (started) => { if (started) playSubmissionFlight() })

function playSubmissionFlight() {
  const stageRect = stage.value!.getBoundingClientRect()
  const sourceRect = composer.value!.sendButtonRef!.getBoundingClientRect()
  const targetRect = (historyOpen.value ? historyHeading.value! : historyToggle.value!).getBoundingClientRect()
  const startX = sourceRect.left + sourceRect.width / 2
  const startY = sourceRect.top + sourceRect.height / 2
  historyBumping.value = false
  submissionFlight.value = { id: ++flightID, style: {
    left: `${startX - stageRect.left}px`, top: `${startY - stageRect.top}px`,
    '--flight-x': `${targetRect.left + targetRect.width / 2 - startX}px`,
    '--flight-y': `${targetRect.top + targetRect.height / 2 - startY}px`,
  } }
}
function finishSubmissionFlight() { submissionFlight.value = null; historyBumping.value = true }

const keyOptions = computed(() => [{ value: 0, label: t('asyncImage.filters.selectKey') }, ...keys.value.map(key => ({ value: key.id, label: key.name }))])
const statusOptions = computed(() => [{ value: '', label: t('asyncImage.filters.allStatuses') }, ...['processing', 'completed', 'failed'].map(value => ({ value, label: statusText(value) }))])
function statusText(value: string) { return t(`asyncImage.status.${value}`) }
function statusColor(value: string) { return value === 'completed' ? 'text-emerald-600 dark:text-emerald-400' : value === 'failed' ? 'text-red-500' : 'text-primary-500' }
function formatDate(value: number) { return new Date(value * 1000).toLocaleString() }
function resetCanvas() { canvas.value?.resetCanvas(); resetConfirm.value = false }
async function deleteFailed() { if (deleteTarget.value) await studio.deleteTask(deleteTarget.value); deleteTarget.value = null }
async function importResult(task: AsyncImageTask) {
  importingTask.value = task.task_id
  await studio.importTask(task)
  importingTask.value = ''
  if (narrow.value) historyOpen.value = false
}
function dragOutput(event: DragEvent, task: AsyncImageTask, index: number) {
  event.dataTransfer!.setData(CREATIVE_OUTPUT_DRAG_MIME, serializeCreativeOutputDrag({ runId: task.task_id, outputIndex: index }))
  event.dataTransfer!.effectAllowed = 'copy'
}
async function generate() {
  if (!ready.value) return
  try {
    let sources: Blob[] = []
    let mask: Blob | null = null
    if (form.operation === 'edit') sources = await canvas.value!.getEditRefBlobs()
    if (form.operation === 'inpaint') {
      const source = await canvas.value!.getSelectedImageBlob()
      if (source) sources = [source]
      mask = await canvas.value!.getMaskBlob()
    }
    if (await studio.submit(sources, mask)) { historyOpen.value = !narrow.value; settingsOpen.value = false }
    else submissionFlight.value = null
  } catch (cause) { error.value = cause instanceof Error ? cause.message : String(cause) }
}
onMounted(studio.init)
</script>

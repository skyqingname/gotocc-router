<template>
  <AppLayout full-viewport>
    <div class="gotocc-creative relative h-full min-h-0 overflow-hidden" :class="historyOpen && 'history-open'">
      <CreativeCanvas ref="canvas" class="absolute inset-0" :operation="form.operation" :resolve-output="studio.resolveOutput" :allowed-mimes="config.allowed_mime_types" @error="error = $event" />

      <div class="studio-account canvas-island absolute left-3 top-3 z-20 flex items-center gap-1 rounded-full p-1">
        <button class="canvas-tool-btn" type="button" :title="t('creative.canvas.reset')" @click="resetConfirm = true"><Icon name="trash" size="sm" /></button>
        <Select v-model="keyID" :options="keyOptions" :disabled="submitting || loadingModels" searchable="auto" class="studio-key-select" :aria-label="t('asyncImage.filters.apiKey')" @change="studio.selectKey" />
      </div>

      <button v-if="!historyOpen" class="studio-history-toggle canvas-island absolute right-3 top-3 z-20 flex h-10 items-center gap-2 rounded-full px-3 text-sm" type="button" :aria-expanded="historyOpen" :aria-label="t('studioGotocc.history')" @click="historyOpen = true">
        <Icon name="clock" size="sm" /><span class="hidden sm:inline">{{ t('studioGotocc.history') }}</span><span v-if="activityCount" class="studio-count">{{ activityCount }}</span>
      </button>

      <div v-if="canvas?.sceneReady && canvas.imageCount === 0" class="studio-empty-guide pointer-events-none absolute inset-x-6 top-[32%] flex flex-col items-center text-center">
        <span class="mb-3 text-[10px] font-semibold uppercase tracking-[0.3em] text-gray-400">GOTOCC / CREATIVE STUDIO</span>
        <h1 class="text-2xl font-semibold tracking-tight text-gray-800 dark:text-gray-100">{{ t('creative.empty.title') }}</h1>
        <p class="mt-3 max-w-md text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('creative.empty.description') }}</p>
        <div class="pointer-events-auto mt-5 flex max-w-xl flex-wrap justify-center gap-2">
          <button type="button" class="canvas-island studio-example" @click="canvas?.openFilePicker()"><Icon name="upload" size="sm" />{{ t('creative.empty.upload') }}</button>
          <button v-for="example in examples" :key="example" type="button" class="canvas-island studio-example" @click="form.prompt = t(`creative.empty.examples.${example}.prompt`)">{{ t(`creative.empty.examples.${example}.label`) }}</button>
        </div>
      </div>

      <form v-support-readonly class="studio-composer canvas-island absolute bottom-5 z-20 rounded-2xl" @submit.prevent="generate">
        <div v-if="error || guide || activityCount" class="studio-status flex items-center gap-2 border-b border-gray-200/60 px-4 py-2 text-xs dark:border-dark-600" :class="error ? 'text-red-600 dark:text-red-300' : 'text-gray-500 dark:text-gray-300'" role="status">
          <Icon :name="error ? 'exclamationCircle' : activityCount ? 'refresh' : 'infoCircle'" size="sm" :class="!error && activityCount ? 'animate-spin' : ''" />
          <span class="min-w-0 flex-1 break-words">{{ error || guide || t('studioGotocc.active', { count: activityCount }) }}</span>
          <button v-if="error" type="button" :aria-label="t('common.close')" @click="error = ''"><Icon name="x" size="sm" /></button>
        </div>
        <textarea v-model="form.prompt" rows="2" class="studio-prompt block w-full resize-none bg-transparent px-4 pb-2 pt-4 text-sm outline-none" :placeholder="t('creative.panel.promptPlaceholder')" :aria-label="t('asyncImage.create.prompt')" :disabled="submitting" @keydown.ctrl.enter.prevent="generate" @keydown.meta.enter.prevent="generate" />
        <div class="flex min-w-0 items-center gap-1 px-2 pb-2">
          <Select v-model="form.model" :options="modelOptions" :disabled="loadingModels || !models.length || submitting" searchable="auto" class="studio-model-select" :aria-label="t('asyncImage.create.model')" @change="studio.onModelChanged" />
          <div ref="parameterRoot" class="relative">
            <button type="button" class="studio-chip" :class="parametersOpen && 'studio-chip-active'" :aria-expanded="parametersOpen" :title="t('creative.composer.params')" @click="parametersOpen = !parametersOpen"><Icon name="cog" size="sm" /><span>{{ t('creative.composer.params') }}</span></button>
            <div v-if="parametersOpen" class="studio-parameters absolute bottom-11 left-1/2 w-72 -translate-x-1/2 rounded-xl border border-gray-200 bg-white p-4 shadow-xl dark:border-dark-600 dark:bg-dark-800">
              <label class="block"><span class="input-label">{{ t('asyncImage.create.size') }}</span><Select v-model="form.size" :options="sizeOptions" :searchable="false" :aria-label="t('asyncImage.create.size')" /></label>
              <div v-if="form.size === 'custom'" class="mt-3 grid grid-cols-2 gap-2">
                <label><span class="input-label">{{ t('asyncImage.create.width') }}</span><input v-model.number="form.customWidth" type="number" class="input w-full" /></label>
                <label><span class="input-label">{{ t('asyncImage.create.height') }}</span><input v-model.number="form.customHeight" type="number" class="input w-full" /></label>
              </div>
              <p v-if="customSizeError" class="mt-2 text-xs text-red-500">{{ customSizeError }}</p>
              <label v-if="isGPTImage2(form.model)" class="mt-3 block"><span class="input-label">{{ t('creative.panel.quality') }}</span><Select v-model="form.quality" :options="qualityOptions" :searchable="false" :aria-label="t('creative.panel.quality')" /></label>
              <label v-if="isGPTImage2(form.model)" class="mt-3 block"><span class="input-label">{{ t('creative.panel.background') }}</span><Select v-model="form.background" :options="backgroundOptions" :searchable="false" :aria-label="t('creative.panel.background')" /></label>
              <label class="mt-3 block"><span class="input-label">{{ t('asyncImage.create.count') }}</span><input v-model.number="form.n" type="number" min="1" :max="config.max_image_count" class="input w-full" /></label>
            </div>
          </div>
          <Select v-model="form.operation" :options="operationOptions" :disabled="submitting" :searchable="false" class="studio-operation-select" :aria-label="t('creative.composer.operation')" />
          <button type="submit" class="studio-send ml-auto flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl bg-primary-600 text-white transition hover:bg-primary-500 disabled:opacity-40" :disabled="!ready" :title="t('creative.composer.send')" :aria-label="t('creative.composer.send')"><Icon :name="submitting ? 'refresh' : 'arrowUp'" size="md" :class="submitting && 'animate-spin'" /></button>
        </div>
        <div v-if="!keyID || (keyID && !canSubmit && !loadingModels)" class="border-t border-gray-100 px-4 py-2 text-xs text-gray-500 dark:border-dark-700">{{ !keyID ? t('studioGotocc.selectKey') : t('studioGotocc.readOnlyKey') }}</div>
      </form>

      <button v-if="historyOpen" type="button" class="absolute inset-0 z-30 bg-black/15 md:hidden" :aria-label="t('common.close')" @click="historyOpen = false" />
      <aside v-if="historyOpen" class="studio-history absolute bottom-3 right-3 top-3 z-40 flex flex-col overflow-hidden rounded-2xl border border-gray-200/80 bg-white/95 shadow-lg backdrop-blur-xl dark:border-dark-700 dark:bg-dark-900/95">
        <div class="flex items-center justify-between gap-2 border-b border-gray-100 px-4 py-3 dark:border-dark-700"><h2 class="font-medium">{{ t('studioGotocc.history') }}</h2><div class="flex gap-1"><button type="button" class="canvas-tool-btn" :disabled="loading" :title="t('asyncImage.actions.refresh')" @click="studio.refreshHistory"><Icon name="refresh" size="sm" :class="loading && 'animate-spin'" /></button><button type="button" class="canvas-tool-btn" :title="t('common.close')" @click="historyOpen = false"><Icon name="x" size="sm" /></button></div></div>
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
            <div v-if="expandedTask === task.task_id" class="space-y-2 border-t border-gray-100 px-3 py-3 dark:border-dark-700">
              <p class="whitespace-pre-wrap break-words text-xs leading-5">{{ params.get(task.task_id)?.prompt || task.prompt_preview }}</p>
              <p v-if="params.has(task.task_id)" class="text-[10px] text-gray-400">{{ params.get(task.task_id)?.imageSize }} · {{ params.get(task.task_id)?.quality }} · {{ t('creative.operations.' + params.get(task.task_id)?.operation) }}</p>
              <p class="break-all font-mono text-[10px] text-gray-400">{{ task.task_id }}</p>
              <p v-if="task.status === 'completed' && task.actual_images != null && task.actual_images < (task.requested_images || 1)" class="text-xs text-amber-600">{{ t('studioGotocc.partial', { actual: task.actual_images, requested: task.requested_images }) }}</p>
              <button type="button" class="studio-link" @click="studio.reuseTask(task)">{{ t('studioGotocc.reuse') }}</button>
            </div>
            <div v-if="task.status === 'completed'" class="flex items-center gap-3 px-3 py-2.5"><button type="button" class="studio-link" :disabled="importingTask === task.task_id" @click="importResult(task)"><Icon name="plus" size="xs" />{{ importingTask === task.task_id ? t('common.loading') : t('studioGotocc.toCanvas') }}</button><button type="button" class="studio-link ml-auto" :title="t('asyncImage.actions.downloadAll')" @click="studio.downloadTask(task)"><Icon name="download" size="sm" /></button></div>
            <div v-if="task.status === 'failed'" class="flex justify-end px-3 pb-2"><button v-support-readonly type="button" class="studio-link text-red-500" @click="deleteTarget = task">{{ t('asyncImage.actions.delete') }}</button></div>
          </article>
        </div>
        <div v-if="tasks.length || offset" class="flex items-center justify-between border-t border-gray-100 px-3 py-2 text-xs dark:border-dark-700"><button type="button" class="studio-link" :disabled="loading || !offset" @click="studio.page(-1)">{{ t('asyncImage.list.previous') }}</button><span class="text-gray-400">{{ Math.floor(offset / config.history_page_size) + 1 }}</span><button type="button" class="studio-link" :disabled="loading || !hasMore" @click="studio.page(1)">{{ t('asyncImage.list.next') }}</button></div>
        <p class="border-t border-gray-100 px-4 py-2 text-[10px] leading-4 text-gray-400 dark:border-dark-700">{{ t('studioGotocc.storageHint') }}</p>
      </aside>
    </div>
    <ConfirmDialog :show="resetConfirm" :title="t('creative.canvas.reset')" :message="t('studioGotocc.resetHint')" @confirm="resetCanvas" @cancel="resetConfirm = false" />
    <ConfirmDialog :show="!!deleteTarget" :title="t('asyncImage.actions.delete')" :message="t('asyncImage.delete.confirm', { taskId: deleteTarget?.task_id })" danger @confirm="deleteFailed" @cancel="deleteTarget = null" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { onClickOutside, useMediaQuery } from '@vueuse/core'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
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
const { config, keys, keyID, loading, loadingModels, submitting, tasks, models, form, status, offset, hasMore, error, activityCount, params, canSubmit, canInpaint, ready, sizes, customSizeError } = studio
const historyOpen = ref(false)
const parametersOpen = ref(false)
const parameterRoot = ref<HTMLElement | null>(null)
const resetConfirm = ref(false)
const deleteTarget = ref<AsyncImageTask | null>(null)
const expandedTask = ref('')
const importingTask = ref('')
const narrow = useMediaQuery('(max-width: 767px)')
const examples = ['poster', 'product', 'illustration'] as const
onClickOutside(parameterRoot, () => { parametersOpen.value = false })

const keyOptions = computed(() => [{ value: 0, label: t('asyncImage.filters.selectKey') }, ...keys.value.map(key => ({ value: key.id, label: key.name }))])
const modelOptions = computed(() => models.value.length ? models.value.map(model => ({ value: model, label: model })) : [{ value: '', label: loadingModels.value ? t('asyncImage.create.modelsLoading') : t('asyncImage.create.model') }])
const operationOptions = computed(() => ['generate', 'edit', ...(canInpaint.value ? ['inpaint'] : [])].map(value => ({ value, label: t(`creative.operations.${value}`) })))
const sizeOptions = computed(() => sizes.value.map(value => ({ value, label: value === 'auto' ? t('asyncImage.create.sizeAuto') : value === 'custom' ? t('asyncImage.create.sizeCustom') : value.replace('x', ' × ') })))
const qualityOptions = computed(() => config.qualities.map(value => ({ value, label: t(`studioGotocc.quality.${value}`) })))
const backgroundOptions = computed(() => config.backgrounds.map(value => ({ value, label: t(`studioGotocc.background.${value}`) })))
const statusOptions = computed(() => [{ value: '', label: t('asyncImage.filters.allStatuses') }, ...['processing', 'completed', 'failed'].map(value => ({ value, label: statusText(value) }))])
const guide = computed(() => canvas.value?.guideKey === 'maskPaint' ? t('creative.panel.maskHint') : canvas.value?.guideKey ? t('creative.panel.selectImageHint') : '')
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
    if (await studio.submit(sources, mask)) { historyOpen.value = !narrow.value; parametersOpen.value = false }
  } catch (cause) { error.value = cause instanceof Error ? cause.message : String(cause) }
}
onMounted(studio.init)
</script>

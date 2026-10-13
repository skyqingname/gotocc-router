import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { unzip } from 'fflate'
import { keysAPI } from '@/api/keys'
import {
  deleteAsyncImageTask, downloadAsyncImageZip, getAsyncImageObjectURL, getAsyncImageTask,
  listAsyncImageModels, listAsyncImageTasks, preferredAsyncImageModel, saveAsyncImageBlob,
  submitAsyncImageEdit, submitAsyncImageGeneration,
  type AsyncImageTask, type AsyncImageTaskResultItem,
} from '@/api/asyncImage'
import { getAutoRoutingCapabilities, isAutoRoutingKey, keyAllowsAsyncImage, keyCanManageAsyncImage } from '@/composables/useAsyncImageAccess'
import { useUserView } from '@/composables/useUserView'
import { isGPTImage2, validateGPTImage2CustomSize } from '@/utils/asyncImageSize'
import type { ApiKey } from '@/types'
import config from '../../../../../creative-studio.json'
import { createCreativeLocalStore, type LocalRunParams } from './localStore'
import type { CreativeOperation } from './types'

export interface StudioOutput { blob: Blob; runId: string; outputIndex: number }

export function useStudio(placeOutput: (output: StudioOutput) => Promise<void>) {
  const { t } = useI18n()
  const local = createCreativeLocalStore(useUserView().user!.id)
  const keys = ref<ApiKey[]>([])
  const keyID = ref(0)
  const loading = ref(false)
  const loadingModels = ref(false)
  const submitting = ref(false)
  const tasks = ref<AsyncImageTask[]>([])
  const models = ref<string[]>([])
  const status = ref('')
  const offset = ref(0)
  const hasMore = ref(false)
  const error = ref('')
  const activityCount = ref(0)
  const params = ref(new Map<string, LocalRunParams>())
  const [initialWidth, initialHeight] = config.default_size.split('x').map(Number)
  const form = reactive({
    model: '', prompt: '', operation: 'generate' as CreativeOperation,
    size: config.default_size, customWidth: initialWidth, customHeight: initialHeight,
    quality: config.default_quality, background: config.default_background, n: config.default_image_count,
  })
  const selectedKey = computed(() => keys.value.find(key => key.id === keyID.value))
  const canSubmit = ref(false)
  const sizes = computed(() => isGPTImage2(form.model) ? config.gpt_image_2_sizes : config.standard_sizes)
  const customSizeError = computed(() => {
    if (form.size !== 'custom') return ''
    const code = validateGPTImage2CustomSize(form.customWidth, form.customHeight)
    return code ? t(`asyncImage.create.customSizeErrors.${code}`) : ''
  })
  const canInpaint = computed(() => selectedKey.value?.group?.platform === 'openai' && !isAutoRoutingKey(selectedKey.value))
  const ready = computed(() => canSubmit.value && !!form.model && !!form.prompt.trim() && !submitting.value && !customSizeError.value && Number.isInteger(form.n) && form.n >= 1 && form.n <= config.max_image_count)
  const active = new Map<string, { key: ApiKey; task: AsyncImageTask }>()
  const importing = new Map<string, Promise<Blob[]>>()
  let pollTimer: ReturnType<typeof setTimeout> | undefined
  let draftTimer: ReturnType<typeof setTimeout> | undefined
  let selectionVersion = 0
  let historyVersion = 0
  let disposed = false
  let hydrated = false

  function showError(cause: unknown) { error.value = cause instanceof Error ? cause.message : String(cause) }

  async function init() {
    loading.value = true
    try {
      const all: ApiKey[] = []
      for (let page = 1; ; page++) {
        const result = await keysAPI.list(page, config.keys_page_size, { sort_by: 'created_at', sort_order: 'desc' })
        all.push(...result.items)
        if (page >= result.pages) break
      }
      if (disposed) return
      keys.value = all.filter(keyCanManageAsyncImage)
      const draft = await local.loadSetting<{ keyID: number; form: typeof form }>('draft')
      params.value = await local.loadRunParamsMap()
      if (draft) Object.assign(form, draft.form)
      keyID.value = draft && keys.value.some(key => key.id === draft.keyID) ? draft.keyID : 0
      if (keyID.value) {
        await selectKey()
        if (draft) {
          Object.assign(form, draft.form)
          if (!models.value.includes(form.model)) form.model = ''
        }
        if (!canInpaint.value && form.operation === 'inpaint') form.operation = 'edit'
        onModelChanged()
      }
      hydrated = true
    } catch (cause) { showError(cause) }
    finally { loading.value = false }
  }

  async function selectKey() {
    const version = ++selectionVersion
    const key = selectedKey.value
    models.value = []
    form.model = ''
    canSubmit.value = false
    status.value = ''
    offset.value = 0
    error.value = ''
    tasks.value = []
    if (!key) return
    const history = refreshHistory()
    const activeTasks = loadActive(key).catch(cause => {
      if (version === selectionVersion && !disposed) showError(cause)
    })
    loadingModels.value = true
    try {
      const capabilities = isAutoRoutingKey(key) ? await getAutoRoutingCapabilities(key) : undefined
      if (version !== selectionVersion || disposed) return
      canSubmit.value = keyAllowsAsyncImage(key, capabilities || undefined)
      if (key.status === 'active') {
        const available = await listAsyncImageModels(key.key)
        if (version !== selectionVersion || disposed) return
        models.value = available
        form.model = preferredAsyncImageModel(available)
        onModelChanged()
      }
      if (!canInpaint.value && form.operation === 'inpaint') form.operation = 'edit'
    } catch (cause) { if (version === selectionVersion) showError(cause) }
    finally {
      await Promise.all([history, activeTasks])
      if (version === selectionVersion) loadingModels.value = false
    }
  }

  function onModelChanged() {
    if (!sizes.value.includes(form.size)) form.size = config.default_size
  }

  function usable(item: AsyncImageTaskResultItem) {
    return !!item.url && (item.url_expires_at == null || item.url_expires_at > Date.now() / 1000 + config.image_url_refresh_skew_seconds)
  }

  async function renew(task: AsyncImageTask, key: ApiKey): Promise<AsyncImageTask> {
    if (!task.result?.data?.length) return { ...task, result: { ...task.result, data: task.image_url ? [{ url: task.image_url }] : [] } }
    const data = await Promise.all(task.result.data.map(async item => {
      if (usable(item) || !item.object_id) return usable(item) ? item : { ...item, url: undefined }
      try {
        const renewed = await getAsyncImageObjectURL(key.key, item.object_id)
        return { ...item, url: renewed.url, url_expires_at: renewed.url_expires_at }
      } catch (cause) {
        showError(cause)
        return { ...item, url: undefined }
      }
    }))
    return { ...task, result: { ...task.result, data }, image_url: data.find(usable)?.url }
  }

  async function refreshHistory() {
    const key = selectedKey.value
    const version = ++historyVersion
    if (!key) { tasks.value = []; hasMore.value = false; return }
    loading.value = true
    try {
      const page = await listAsyncImageTasks(key.key, { status: status.value, offset: offset.value, limit: config.history_page_size })
      const refreshed = await Promise.all(page.data.map(task => renew(task, key)))
      if (version !== historyVersion || key.id !== keyID.value || disposed) return
      tasks.value = refreshed
      hasMore.value = page.has_more
    } catch (cause) { if (version === historyVersion) showError(cause) }
    finally { if (version === historyVersion) loading.value = false }
  }

  async function loadActive(key: ApiKey) {
    for (let position = 0; ; position += config.history_page_size) {
      const page = await listAsyncImageTasks(key.key, { status: 'processing', offset: position, limit: config.history_page_size })
      if (disposed) return
      for (const task of page.data) active.set(task.task_id, { key, task })
      if (!page.has_more) break
    }
    schedulePoll()
  }

  function schedulePoll() {
    clearTimeout(pollTimer)
    activityCount.value = active.size
    if (!active.size || disposed) return
    pollTimer = setTimeout(poll, config.poll_interval_ms)
  }

  async function poll() {
    await Promise.all([...active].map(async ([id, entry]) => {
      try {
        const task = await renew(await getAsyncImageTask(entry.key.key, id), entry.key)
        if (disposed) return
        entry.task = task
        if (entry.key.id === keyID.value) {
          const index = tasks.value.findIndex(row => row.task_id === id)
          if (index !== -1) tasks.value.splice(index, 1, task)
        }
        if (task.status !== 'processing') {
          active.delete(id)
          if (task.status === 'completed') await importTask(task, entry.key)
          if (task.status === 'failed') error.value = task.error?.message || t('asyncImage.status.failed')
        }
      } catch (cause) { if (!disposed) showError(cause) }
    }))
    schedulePoll()
  }

  async function submit(sourceBlobs: Blob[], maskBlob: Blob | null): Promise<boolean> {
    const key = selectedKey.value
    if (!key || !ready.value) return false
    if (form.operation !== 'generate' && !sourceBlobs.length) { error.value = t('creative.panel.selectImageHint'); return false }
    if (form.operation === 'inpaint' && !maskBlob) { error.value = t('creative.error.maskRequired'); return false }
    if (sourceBlobs.length > config.max_reference_images) { error.value = t('asyncImage.create.referenceImageLimit', { count: config.max_reference_images }); return false }
    if (sourceBlobs.some(blob => blob.size > config.max_upload_file_bytes)) { error.value = t('asyncImage.create.uploadFileTooLarge'); return false }
    if (sourceBlobs.reduce((sum, blob) => sum + blob.size, maskBlob?.size || 0) > config.max_upload_total_bytes) { error.value = t('asyncImage.create.uploadTotalTooLarge'); return false }
    const snapshot = { ...form }
    const size = snapshot.size === 'custom' ? `${snapshot.customWidth}x${snapshot.customHeight}` : snapshot.size
    const payload = {
      model: snapshot.model, prompt: snapshot.prompt.trim(), size, n: snapshot.n,
      ...(isGPTImage2(snapshot.model) ? { quality: snapshot.quality, background: snapshot.background } : {}),
    }
    submitting.value = true
    error.value = ''
    try {
      const created = snapshot.operation === 'generate'
        ? await submitAsyncImageGeneration(key.key, payload)
        : await submitAsyncImageEdit(key.key, {
          ...payload, images: sourceBlobs.map((blob, index) => new File([blob], `reference-${index + 1}.${blob.type.split('/')[1]}`, { type: blob.type })),
          ...(maskBlob ? { mask: new File([maskBlob], 'mask.png', { type: 'image/png' }) } : {}),
        })
      const task = { ...created, model: snapshot.model, prompt_preview: snapshot.prompt, request_type: snapshot.operation === 'generate' ? 'generation' : 'edit' }
      const saved: LocalRunParams = {
        runId: task.task_id, optionKey: String(key.id), groupName: key.name, operation: snapshot.operation,
        prompt: snapshot.prompt, imageSize: size, aspectRatio: '', quality: snapshot.quality,
        background: snapshot.background, thinkingLevel: '', referenceCount: sourceBlobs.length, createdAt: Date.now(),
        model: snapshot.model, imageCount: snapshot.n,
      }
      params.value.set(task.task_id, saved)
      try { await local.saveRunParams(saved) } catch (cause) { showError(cause) }
      if (disposed) return true
      if (key.id === keyID.value) {
        historyVersion++; loading.value = false
        hasMore.value = hasMore.value || tasks.value.length >= config.history_page_size
        status.value = ''; offset.value = 0
        tasks.value = [task, ...tasks.value].slice(0, config.history_page_size)
      }
      active.set(task.task_id, { key, task })
      schedulePoll()
      return true
    } catch (cause) { showError(cause); return false }
    finally { submitting.value = false }
  }

  async function taskBlobs(task: AsyncImageTask, key: ApiKey): Promise<Blob[]> {
    const pending = importing.get(task.task_id)
    if (pending) return pending
    const job = (async () => {
      // The authenticated ZIP endpoint serves stored images without depending on the bucket's browser CORS policy.
      const archive = await downloadAsyncImageZip(key.key, task.task_id)
      const files = await new Promise<Record<string, Uint8Array>>((resolve, reject) => {
        void archive.arrayBuffer().then(buffer => unzip(new Uint8Array(buffer), (cause, data) => cause ? reject(cause) : resolve(data))).catch(reject)
      })
      const entries = Object.entries(files).sort(([a], [b]) => a.localeCompare(b, undefined, { numeric: true }))
      return entries.map(([name, bytes], index) => {
        const mime = task.result?.data?.[index]?.content_type || ({ png: 'image/png', jpg: 'image/jpeg', jpeg: 'image/jpeg', webp: 'image/webp' }[name.split('.').pop()!] || 'application/octet-stream')
        return new Blob([bytes], { type: mime })
      })
    })()
    importing.set(task.task_id, job)
    try { return await job } finally { importing.delete(task.task_id) }
  }

  async function importTask(task: AsyncImageTask, key = selectedKey.value) {
    if (!key) return
    try {
      const blobs = await taskBlobs(task, key)
      for (let index = 0; index < blobs.length; index++) {
        if (disposed) return
        const asset = { blob: blobs[index], runId: task.task_id, outputIndex: index }
        await placeOutput(asset)
      }
    } catch (cause) { showError(cause) }
  }

  async function resolveOutput(runId: string, outputIndex: number): Promise<Blob> {
    const key = selectedKey.value!
    const task = tasks.value.find(task => task.task_id === runId)!
    return (await taskBlobs(task, key))[outputIndex]
  }

  async function downloadTask(task: AsyncImageTask) {
    const key = selectedKey.value
    if (!key) return
    try { saveAsyncImageBlob(await downloadAsyncImageZip(key.key, task.task_id), `${task.task_id}.zip`) }
    catch (cause) { showError(cause) }
  }

  async function deleteTask(task: AsyncImageTask) {
    const key = selectedKey.value
    if (!key) return
    try { await deleteAsyncImageTask(key.key, task.task_id); await refreshHistory() }
    catch (cause) { showError(cause) }
  }

  function reuseTask(task: AsyncImageTask) {
    const saved = params.value.get(task.task_id)
    form.prompt = saved?.prompt || task.prompt_preview || ''
    if (task.model && models.value.includes(task.model)) form.model = task.model
    if (saved) {
      form.quality = saved.quality; form.background = saved.background; form.n = saved.imageCount
      form.operation = saved.operation as CreativeOperation
      if (sizes.value.includes(saved.imageSize)) form.size = saved.imageSize
      else if (isGPTImage2(form.model)) {
        const [width, height] = saved.imageSize.split('x').map(Number)
        form.size = 'custom'; form.customWidth = width; form.customHeight = height
      }
    }
    if (!canInpaint.value && form.operation === 'inpaint') form.operation = 'edit'
  }

  async function page(direction: number) { offset.value += direction * config.history_page_size; await refreshHistory() }
  async function filter() { offset.value = 0; await refreshHistory() }
  function saveDraft() { return local.saveSetting('draft', { keyID: keyID.value, form: { ...form } }).catch(showError) }
  watch([form, keyID], () => {
    if (!hydrated) return
    clearTimeout(draftTimer)
    draftTimer = setTimeout(saveDraft, config.draft_save_debounce_ms)
  }, { deep: true })
  onBeforeUnmount(() => { disposed = true; selectionVersion++; historyVersion++; clearTimeout(pollTimer); clearTimeout(draftTimer); if (hydrated) void saveDraft() })
  return { config, keys, keyID, selectedKey, loading, loadingModels, submitting, tasks, models, form, status, offset, hasMore, error, activityCount, params, canSubmit, canInpaint, ready, sizes, customSizeError, init, selectKey, onModelChanged, refreshHistory, submit, importTask, resolveOutput, downloadTask, deleteTask, reuseTask, page, filter }
}

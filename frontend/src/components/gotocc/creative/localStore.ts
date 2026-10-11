// Adapted from TokenFlux/TokenRouter f383062dd5fd534f462887acb70ea241ce0d8278; LGPL-3.0-or-later.
// Copyright (c) 2026 Wesley Liddick & TokenFlux. GoToCC: bind storage to the current user.
import config from '../../../../../creative-studio.json'

// 素材种类：源图 / mask / AI 输出
export type LocalAssetKind = 'source' | 'mask' | 'output'

// 本地素材记录，blob 为图片二进制本体
export interface LocalAsset {
  key: string
  kind: LocalAssetKind
  blob: Blob
  runId?: string
  outputIndex?: number
  createdAt: number
}

// 场景 JSON 快照（fabric canvas toJSON 结果）
export interface LocalSceneRecord {
  key: string
  json: string
  updatedAt: number
}

// 设置项（上次选择的模型、尺寸等恢复用）
export interface LocalSettingRecord {
  key: string
  value: unknown
  updatedAt: number
}

// 任务提交时的提示词和参数快照。服务端不保存提示词明文，历史记录从这里读取提示词、画质等参数。
export interface LocalRunParams {
  model: string
  imageCount: number
  runId: string
  // 提交时选中的模型选项 key（group_id::model），填回输入框时用来重新选中模型
  optionKey: string
  groupName: string
  operation: string
  prompt: string
  imageSize: string
  aspectRatio: string
  quality: string
  background: string
  thinkingLevel: string
  // 随任务上传的参考图张数，局部重绘的 mask 不计入
  referenceCount: number
  createdAt: number
}

// 本地存储错误类型
export type LocalStoreErrorType = 'quota' | 'unavailable' | 'unknown'

export class LocalStoreError extends Error {
  readonly type: LocalStoreErrorType

  constructor(type: LocalStoreErrorType, message: string, cause?: unknown) {
    super(message)
    this.name = 'LocalStoreError'
    this.type = type
    // 保留原始错误便于排查
    if (cause !== undefined) {
      const target = this as { cause?: unknown }
      target.cause = cause
    }
  }
}

// 配额不足专用错误，UI 据此提示“空间不足请下载备份”
export class LocalStoreQuotaError extends LocalStoreError {
  constructor(message = 'Local storage quota exceeded', cause?: unknown) {
    super('quota', message, cause)
    this.name = 'LocalStoreQuotaError'
  }
}

// ==================== key 组合规则 ====================

// 输出素材 key：runId + 输出序号（关联服务端 run 元信息）
export function outputAssetKey(runId: string, outputIndex: number): string {
  return `output:${runId}:${outputIndex}`
}

// 本地素材 key（源图 / mask 等用户本地产物）
export function localAssetKey(kind: LocalAssetKind, localId: string): string {
  return `${kind}:local:${localId}`
}


export function createCreativeLocalStore(userID: number) {
const DB_NAME = `${config.local_store_name}:${userID}`
const DB_VERSION = 1

const STORE_ASSETS = 'assets'
const STORE_SCENES = 'scenes'
const STORE_SETTINGS = 'settings'

function idbRequest<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(toLocalStoreError(request.error))
  })
}

// 统一错误归一化：QuotaExceededError 单独识别
function toLocalStoreError(error: unknown): LocalStoreError {
  if (error instanceof LocalStoreError) return error
  const name = (error as DOMException | null)?.name || ''
  if (name === 'QuotaExceededError' || (error as DOMException)?.code === 22) {
    return new LocalStoreQuotaError('Local storage quota exceeded', error)
  }
  if (
    name === 'InvalidStateError' ||
    name === 'SecurityError' ||
    name === 'NotSupportedError' ||
    name === 'UnknownError'
  ) {
    return new LocalStoreError('unavailable', 'Local storage unavailable', error)
  }
  return new LocalStoreError('unknown', 'Local storage operation failed', error)
}

// 打开（并升级）数据库，进程内缓存打开 Promise
let dbPromise: Promise<IDBDatabase> | null = null

function openStore(): Promise<IDBDatabase> {
  if (dbPromise) return dbPromise
  dbPromise = new Promise((resolve, reject) => {
    if (typeof indexedDB === 'undefined') {
      reject(new LocalStoreError('unavailable', 'IndexedDB is not supported in this environment'))
      return
    }
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onupgradeneeded = () => {
      const db = request.result
      // 三个 store 均以 key 为主键；assets 额外建 kind 索引用于按类列举
      if (!db.objectStoreNames.contains(STORE_ASSETS)) {
        const store = db.createObjectStore(STORE_ASSETS, { keyPath: 'key' })
        store.createIndex('kind', 'kind', { unique: false })
      }
      if (!db.objectStoreNames.contains(STORE_SCENES)) {
        db.createObjectStore(STORE_SCENES, { keyPath: 'key' })
      }
      if (!db.objectStoreNames.contains(STORE_SETTINGS)) {
        db.createObjectStore(STORE_SETTINGS, { keyPath: 'key' })
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => {
      dbPromise = null
      reject(toLocalStoreError(request.error))
    }
    request.onblocked = () => {
      dbPromise = null
      reject(new LocalStoreError('unavailable', 'Local storage is blocked by another session'))
    }
  })
  return dbPromise
}

// 便捷封装：在指定 store 上执行一个读写操作
async function withStore<T>(
  storeName: string,
  mode: IDBTransactionMode,
  run: (store: IDBObjectStore) => IDBRequest<T>,
): Promise<T> {
  const db = await openStore()
  try {
    return await idbRequest(run(db.transaction(storeName, mode).objectStore(storeName)))
  } catch (error) {
    throw toLocalStoreError(error)
  }
}

// ==================== assets API ====================

// 保存素材（同 key 覆盖写，天然幂等）
async function saveAsset(asset: LocalAsset): Promise<LocalAsset> {
  await withStore(STORE_ASSETS, 'readwrite', (store) => store.put(asset))
  return asset
}

// 读取单个素材，不存在时返回 null
async function loadAsset(key: string): Promise<LocalAsset | null> {
  const record = await withStore(STORE_ASSETS, 'readonly', (store) =>
    store.get(key) as IDBRequest<LocalAsset | undefined>,
  )
  return record ?? null
}

// 按种类列举素材（按创建时间升序）
async function listAssets(kind: LocalAssetKind): Promise<LocalAsset[]> {
  const db = await openStore()
  try {
    const store = db.transaction(STORE_ASSETS, 'readonly').objectStore(STORE_ASSETS)
    // 优先走 kind 索引；索引不可用时退化为全量过滤
    let records: LocalAsset[]
    if (store.indexNames.contains('kind')) {
      records = await idbRequest(store.index('kind').getAll(kind) as IDBRequest<LocalAsset[]>)
    } else {
      records = await idbRequest(store.getAll() as IDBRequest<LocalAsset[]>)
      records = records.filter((item) => item.kind === kind)
    }
    return records.sort((a, b) => a.createdAt - b.createdAt)
  } catch (error) {
    throw toLocalStoreError(error)
  }
}

// 删除单个素材，key 不存在时视为成功（幂等）
async function deleteAsset(key: string): Promise<void> {
  await withStore(STORE_ASSETS, 'readwrite', (store) => store.delete(key))
}

// 清空某个种类的全部素材
async function clearKind(kind: LocalAssetKind): Promise<void> {
  const assets = await listAssets(kind)
  const db = await openStore()
  try {
    const store = db.transaction(STORE_ASSETS, 'readwrite').objectStore(STORE_ASSETS)
    for (const asset of assets) {
      await idbRequest(store.delete(asset.key))
    }
  } catch (error) {
    throw toLocalStoreError(error)
  }
}

// ==================== scenes API ====================

// 保存画布场景 JSON 快照
async function saveSceneJson(key: string, json: string): Promise<void> {
  const record: LocalSceneRecord = { key, json, updatedAt: Date.now() }
  await withStore(STORE_SCENES, 'readwrite', (store) => store.put(record))
}

// 读取场景 JSON，不存在时返回 null
async function loadSceneJson(key: string): Promise<string | null> {
  const record = await withStore(STORE_SCENES, 'readonly', (store) =>
    store.get(key) as IDBRequest<LocalSceneRecord | undefined>,
  )
  return record?.json ?? null
}

// ==================== settings API ====================

// 保存设置项（上次选择恢复用）
async function saveSetting(key: string, value: unknown): Promise<void> {
  const record: LocalSettingRecord = { key, value, updatedAt: Date.now() }
  await withStore(STORE_SETTINGS, 'readwrite', (store) => store.put(record))
}

// 读取设置项，不存在时返回 null
async function loadSetting<T = unknown>(key: string): Promise<T | null> {
  const record = await withStore(STORE_SETTINGS, 'readonly', (store) =>
    store.get(key) as IDBRequest<LocalSettingRecord | undefined>,
  )
  return record ? (record.value as T) : null
}

// ==================== 任务参数 API ====================

// 任务参数记录和设置项共用 settings store，key 带 run: 前缀；清空本机数据时一起删除。
const RUN_PARAMS_KEY_PREFIX = 'run:'
// 历史列表展示最近 20 条任务和全部进行中任务，本地最多保留最近 200 条参数记录。
const MAX_RUN_PARAMS_RECORDS = config.local_history_limit

function runParamsKeyRange(): IDBKeyRange {
  return IDBKeyRange.bound(RUN_PARAMS_KEY_PREFIX, `${RUN_PARAMS_KEY_PREFIX}\uffff`)
}

// 保存一次任务的参数快照，超过上限时删除最早的记录。
async function saveRunParams(params: LocalRunParams): Promise<void> {
  const record: LocalSettingRecord = {
    key: `${RUN_PARAMS_KEY_PREFIX}${params.runId}`,
    value: params,
    updatedAt: Date.now(),
  }
  const db = await openStore()
  try {
    const store = db.transaction(STORE_SETTINGS, 'readwrite').objectStore(STORE_SETTINGS)
    await idbRequest(store.put(record))
    const records = await idbRequest(store.getAll(runParamsKeyRange()) as IDBRequest<LocalSettingRecord[]>)
    if (records.length <= MAX_RUN_PARAMS_RECORDS) return
    records.sort((a, b) => a.updatedAt - b.updatedAt)
    for (const stale of records.slice(0, records.length - MAX_RUN_PARAMS_RECORDS)) {
      await idbRequest(store.delete(stale.key))
    }
  } catch (error) {
    throw toLocalStoreError(error)
  }
}

// 读取全部任务参数快照，按 runId 建索引。
async function loadRunParamsMap(): Promise<Map<string, LocalRunParams>> {
  const records = await withStore(STORE_SETTINGS, 'readonly', (store) =>
    store.getAll(runParamsKeyRange()) as IDBRequest<LocalSettingRecord[]>,
  )
  const map = new Map<string, LocalRunParams>()
  for (const record of records) {
    const params = record.value as LocalRunParams | null
    if (params && typeof params.runId === 'string') map.set(params.runId, params)
  }
  return map
}

// ==================== 维护 API ====================

// 清空创作台全部本地数据（素材、场景、设置和任务参数）
async function clearAll(): Promise<void> {
  const db = await openStore()
  try {
    for (const storeName of [STORE_ASSETS, STORE_SCENES, STORE_SETTINGS]) {
      await idbRequest(db.transaction(storeName, 'readwrite').objectStore(storeName).clear())
    }
  } catch (error) {
    throw toLocalStoreError(error)
  }
}


return { saveAsset, loadAsset, listAssets, deleteAsset, saveSceneJson, loadSceneJson, saveSetting, loadSetting, saveRunParams, loadRunParamsMap, clearAll, clearKind }
}

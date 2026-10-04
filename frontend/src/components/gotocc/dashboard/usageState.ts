import { computed, inject, provide, ref, type InjectionKey } from 'vue'
import { useI18n } from 'vue-i18n'
import { usageAPI, type TrendParams } from '@/api/usage'
import { keysAPI } from '@/api/keys'
import { userGroupsAPI } from '@/api/groups'
import { teamAPI, type TeamAPIKey } from '@/api/team'
import type { SelectOption } from '@/components/common/Select.vue'
import type { ModelStat, TrendDataPoint } from '@/types'
import {
  bucketKeys,
  fillTrendBuckets,
  formatDayKey,
  previousRange,
  rangeQuery,
  resolveCustomRange,
  resolvePresetRange,
  summarizeTrend,
  type UsageRange,
  type UsageRangePreset,
  type UsageTotals,
} from './usageData'

// 仪表盘用量区的共享状态：工具栏修改条件，指标卡、趋势图、模型排行和热力图展示。结构参照 TokenRouter（LGPL-3.0）。

export type UsageMetric = 'requests' | 'tokens' | 'cost' | 'cacheHitRate'
export type PresetRange = Exclude<UsageRangePreset, 'custom'>

export const USAGE_METRICS: UsageMetric[] = ['requests', 'tokens', 'cost', 'cacheHitRate']
const PRESETS: PresetRange[] = ['7d', '30d']

// 记住上次选择的指标和快捷范围；自定义日期与筛选条件下次打开回到默认值。
const STORAGE_KEY = 'gotocc-dashboard-usage'
// 日期选择器的快捷项能对应快捷范围时按快捷范围处理并记住；其余快捷项（如近 24 小时即昨天和今天）按自定义日期处理。
const PICKER_PRESET_MAP: Record<string, PresetRange> = { '7days': '7d', '30days': '30d' }

// 筛选条件，null 表示不限。
export interface UsageFilters {
  api_key_id: number | null
  model: string | null
  group_id: number | null
}

type NamedOption = { id: number; name: string }

const emptyFilters = (): UsageFilters => ({ api_key_id: null, model: null, group_id: null })

// 没有团队是正常状态，此时只列个人密钥；判断方式与使用记录页一致。
const isNoTeam = (error: any): boolean => (
  error?.reason === 'TEAM_NOT_FOUND'
  || error?.reason === 'TEAM_MEMBERSHIP_REQUIRED'
  || error?.response?.status === 404
)

const readSaved = (): { metric: UsageMetric; range: PresetRange } => {
  const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}')
  return {
    metric: USAGE_METRICS.includes(saved.metric) ? saved.metric : 'tokens',
    range: PRESETS.includes(saved.range) ? saved.range : '7d',
  }
}

function createUsageState() {
  const { t } = useI18n()
  const saved = readSaved()
  const metric = ref<UsageMetric>(saved.metric)
  const rangePreset = ref<UsageRangePreset>(saved.range)
  const customRange = ref<UsageRange | null>(null)
  const activeRange = ref<UsageRange>(resolvePresetRange(saved.range))

  const loading = ref(false)
  const loaded = ref(false)
  const current = ref<TrendDataPoint[]>([])
  // 上一周期按下标与本期对齐，用于对比线和环比。
  const previous = ref<TrendDataPoint[]>([])
  const previousTotals = ref<UsageTotals | null>(null)
  const models = ref<ModelStat[]>([])
  const totals = computed(() => summarizeTrend(current.value))

  const filters = ref<UsageFilters>(emptyFilters())
  const apiKeys = ref<NamedOption[]>([])
  const groups = ref<NamedOption[]>([])
  const modelNames = ref<string[]>([])
  const activeFilterCount = computed(() => Object.values(filters.value).filter((value) => value !== null).length)

  const queryParams = computed<Partial<TrendParams>>(() => {
    const params: Partial<TrendParams> = {}
    if (filters.value.api_key_id !== null) params.api_key_id = filters.value.api_key_id
    if (filters.value.model !== null) params.model = filters.value.model
    if (filters.value.group_id !== null) params.group_id = filters.value.group_id
    return params
  })

  const keyOptions = computed<SelectOption[]>(() => [
    { value: null, label: t('gotocc.dashboard.filters.allKeys') },
    ...apiKeys.value.map((key) => ({ value: key.id, label: key.name })),
  ])
  const modelOptions = computed<SelectOption[]>(() => [
    { value: null, label: t('gotocc.dashboard.filters.allModels') },
    ...modelNames.value.map((name) => ({ value: name, label: name })),
  ])
  const groupOptions = computed<SelectOption[]>(() => [
    { value: null, label: t('gotocc.dashboard.filters.allGroups') },
    ...groups.value.map((group) => ({ value: group.id, label: group.name })),
  ])

  // 日期选择器显示当前窗口覆盖的日期；结束时间是排他边界，回退一毫秒。
  const pickerStart = computed(() => formatDayKey(activeRange.value.startAt))
  const pickerEnd = computed(() => formatDayKey(new Date(activeRange.value.endAt.getTime() - 1)))
  // 当前自定义范围正好覆盖的单日，供热力图高亮。
  const selectedDay = computed(() => (
    rangePreset.value === 'custom' && pickerStart.value === pickerEnd.value ? pickerStart.value : null
  ))

  const saveState = () => {
    const range = rangePreset.value === 'custom' ? readSaved().range : rangePreset.value
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ metric: metric.value, range }))
  }

  const fetchTrend = (range: UsageRange) => usageAPI.getDashboardTrend({
    ...rangeQuery(range),
    granularity: range.granularity,
    ...queryParams.value,
  })

  const fetchModels = (range: UsageRange, params: Partial<TrendParams>) => usageAPI.getDashboardModels({
    ...rangeQuery(range),
    ...params,
  })

  // 模型候选取当前范围内用过的全部模型；已选模型即使不在结果里也保留。
  const setModelOptions = (items: ModelStat[]) => {
    const names = new Set(items.map((item) => item.model))
    if (filters.value.model !== null) names.add(filters.value.model)
    modelNames.value = [...names].sort()
  }

  // 用递增序号丢弃过期响应，快速切换范围或筛选时只保留最后一次结果。
  let requestSeq = 0

  const loadTrend = async (range: UsageRange) => {
    const seq = ++requestSeq
    loading.value = true
    const [currentResult, previousResult, modelsResult] = await Promise.all([
      fetchTrend(range),
      fetchTrend(previousRange(range)),
      fetchModels(range, queryParams.value),
    ]).finally(() => {
      if (seq === requestSeq) loading.value = false
    })
    if (seq !== requestSeq) return
    // 接口没有用量时返回 null 列表。
    current.value = fillTrendBuckets(currentResult.trend ?? [], bucketKeys(range))
    previous.value = fillTrendBuckets(previousResult.trend ?? [], bucketKeys(previousRange(range)))
    previousTotals.value = summarizeTrend(previousResult.trend ?? [])
    models.value = modelsResult.models ?? []
    loaded.value = true
  }

  // 按当前范围重新取数；有筛选时另取一次不带筛选的模型候选，避免候选随筛选变少。
  const load = async () => {
    const range = customRange.value ?? resolvePresetRange(rangePreset.value as PresetRange)
    activeRange.value = range
    if (activeFilterCount.value === 0) {
      await loadTrend(range)
      setModelOptions(models.value)
      return
    }
    const [, candidates] = await Promise.all([loadTrend(range), fetchModels(range, {})])
    setModelOptions(candidates.models ?? [])
  }

  // 密钥候选为个人密钥加团队密钥，分组候选为可用分组加团队密钥绑定的分组，与使用记录页一致。
  const loadFilterOptions = async () => {
    const [personal, available] = await Promise.all([
      keysAPI.list(1, 100, { scope: 'personal' }),
      userGroupsAPI.getAvailable(),
    ])
    let teamKeys: TeamAPIKey[] = []
    try {
      await teamAPI.current()
      teamKeys = await teamAPI.keys()
    } catch (error) {
      if (!isNoTeam(error)) throw error
    }
    const keyMap = new Map<number, NamedOption>()
    for (const key of [...personal.items, ...teamKeys]) keyMap.set(key.id, { id: key.id, name: key.name })
    apiKeys.value = [...keyMap.values()]
    const groupMap = new Map<number, NamedOption>()
    for (const group of available) groupMap.set(group.id, { id: group.id, name: group.name })
    for (const key of teamKeys) {
      if (key.group_id !== null && key.group_name) groupMap.set(key.group_id, { id: key.group_id, name: key.group_name })
    }
    groups.value = [...groupMap.values()]
  }

  // 筛选只影响趋势和模型用量，候选项沿用当前范围的结果。
  const applyFilters = () => {
    void loadTrend(activeRange.value)
  }

  const resetFilters = () => {
    filters.value = emptyFilters()
    applyFilters()
  }

  const selectMetric = (value: UsageMetric) => {
    metric.value = value
    saveState()
  }

  const selectPreset = (value: PresetRange) => {
    rangePreset.value = value
    customRange.value = null
    saveState()
    void load()
  }

  const selectCustomRange = (range: { startDate: string; endDate: string; preset: string | null }) => {
    const mapped = range.preset === null ? undefined : PICKER_PRESET_MAP[range.preset]
    if (mapped) {
      selectPreset(mapped)
      return
    }
    rangePreset.value = 'custom'
    customRange.value = resolveCustomRange(range.startDate, range.endDate)
    void load()
  }

  // 把范围切到某一天，单日窗口按小时聚合。
  const selectDay = (date: string) => {
    selectCustomRange({ startDate: date, endDate: date, preset: null })
  }

  // 按模型筛选整个用量区，传入 null 取消。
  const applyModelFilter = (model: string | null) => {
    filters.value.model = model
    applyFilters()
  }

  return {
    metric,
    activeRange,
    loading,
    loaded,
    current,
    previous,
    previousTotals,
    models,
    totals,
    filters,
    activeFilterCount,
    keyOptions,
    modelOptions,
    groupOptions,
    pickerStart,
    pickerEnd,
    selectedDay,
    load,
    loadFilterOptions,
    applyFilters,
    resetFilters,
    selectMetric,
    selectCustomRange,
    selectDay,
    applyModelFilter,
  }
}

export type UsageState = ReturnType<typeof createUsageState>

const USAGE_STATE_KEY: InjectionKey<UsageState> = Symbol('gotoccUsageState')

// 页面创建共享状态，供工具栏和各区块注入。
export const provideUsageState = (): UsageState => {
  const state = createUsageState()
  provide(USAGE_STATE_KEY, state)
  return state
}

export const injectUsageState = (): UsageState => inject(USAGE_STATE_KEY)!

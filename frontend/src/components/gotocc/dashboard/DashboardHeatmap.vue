<template>
  <div ref="cardRef" class="card relative p-4" :aria-busy="loading">
    <div class="mb-4 flex items-center justify-between gap-2">
      <h3 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
        <Icon name="calendar" size="sm" class="text-primary-600 dark:text-primary-400" />
        {{ t('gotocc.dashboard.heatmap.title') }}
      </h3>
      <!-- 色阶图例 -->
      <div class="flex shrink-0 items-center gap-1 text-xs text-gray-500 dark:text-dark-400">
        <span>{{ t('gotocc.dashboard.heatmap.less') }}</span>
        <span v-for="level in LEVEL_CLASSES" :key="level" class="heatmap-cell h-3 w-3" :class="level" />
        <span>{{ t('gotocc.dashboard.heatmap.more') }}</span>
      </div>
    </div>

    <!-- 格子固定 12px、间距 3px，周列数随容器宽度自适应，宽屏显示更多历史周 -->
    <div ref="gridWrapRef" @mouseleave="hoveredDay = null" @focusout="onGridFocusOut" @keydown="onGridKeydown" @animationend="onWaveEnd">
      <!-- 第 1 列是星期标签，第 1 行是月份标签，其余为日期格子；格子显式指定行列，保证与标签对齐 -->
      <div class="grid justify-between" :style="{ gridTemplateColumns: `auto repeat(${visibleWeeks}, ${CELL_PX}px)`, gap: `${GAP_PX}px` }">
        <template v-if="loading && days.length === 0">
          <div
            v-for="cell in visibleWeeks * 7"
            :key="`loading-${cell}`"
            aria-hidden="true"
            class="heatmap-cell h-3 w-3 animate-pulse bg-gray-200 dark:bg-dark-700 motion-reduce:animate-none"
            :style="{ gridColumn: Math.floor((cell - 1) / 7) + 2, gridRow: ((cell - 1) % 7) + 2 }"
          ></div>
        </template>
        <div
          v-for="m in monthItems"
          :key="`m-${m.weekIndex}`"
          class="overflow-visible whitespace-nowrap text-xs leading-4 text-gray-400 dark:text-dark-400"
          :style="{ gridColumn: m.weekIndex + 2, gridRow: 1 }"
        >{{ m.label }}</div>
        <div
          v-for="w in weekdayLabels"
          :key="`w-${w.row}`"
          class="flex items-center pr-1 text-xs leading-none text-gray-400 dark:text-dark-400"
          :style="{ gridColumn: 1, gridRow: w.row + 2 }"
        >{{ w.label }}</div>
        <!-- 格子可点击查看当天按小时的用量；键盘只有一个格子可 Tab 进入，方向键移动，Enter / Space 选中 -->
        <div
          v-for="day in visibleDays"
          :key="day.date"
          :data-date="day.date"
          :data-wave-last="day.weekIndex === visibleWeeks - 1 ? 'true' : undefined"
          class="heatmap-cell h-3 w-3"
          :class="cellClasses(day)"
          :style="{ gridColumn: day.weekIndex + 2, gridRow: day.dayOfWeek + 2, '--wave-delay': waveDelay(day.weekIndex) }"
          :role="day.future ? undefined : 'button'"
          :tabindex="day.future ? undefined : day.date === focusDate ? 0 : -1"
          :aria-label="day.future ? undefined : cellLabel(day)"
          :aria-pressed="day.future ? undefined : day.date === selectedDay"
          @mouseenter="onCellHover(day, $event)"
          @focus="onCellHover(day, $event)"
          @click="selectCell(day)"
        />
      </div>
    </div>

    <!-- 悬停提示：卡片内绝对定位；前两行格子改为下方弹出 -->
    <div
      v-if="hoveredDay"
      class="pointer-events-none absolute z-20 whitespace-nowrap rounded-lg bg-gray-900 px-2.5 py-1.5 text-xs text-white shadow-lg dark:bg-dark-600"
      :style="tooltipStyle"
    >
      <div class="font-medium">{{ formatDayLabel(hoveredDay.date) }}</div>
      <template v-if="hoveredDay.requests > 0">
        <div>{{ t('gotocc.dashboard.heatmap.requests') }}: {{ formatNumber(hoveredDay.requests) }}</div>
        <div>{{ t('gotocc.dashboard.heatmap.tokens') }}: {{ formatTokensK(hoveredDay.tokens) }}</div>
        <div>{{ t('gotocc.dashboard.heatmap.cost') }}: {{ formatUsd(hoveredDay.actualCost, 4) }}</div>
      </template>
      <div v-else>{{ t('gotocc.dashboard.heatmap.noUsage') }}</div>
      <div class="mt-1 opacity-70">{{ t('gotocc.dashboard.heatmap.selectDay') }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { usageAPI } from '@/api/usage'
import { formatDateLocalInput, formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import { formatUsd } from './format'
import { HEATMAP_WAVE_MAX_MS, HEATMAP_WAVE_STEP_MS } from './motion'
import { injectUsageState } from './usageState'

// 每日活跃热力图，结构与交互参照 TokenRouter（LGPL-3.0），色阶改用品牌主色。

const CELL_PX = 12
const GAP_PX = 3
// 星期标签列的近似宽度，用于计算可见周数。
const LABEL_COL_PX = 24
// 拉取近三年的按日数据，宽屏时可以显示更多历史周。
const FETCH_DAYS = 3 * 364
// 0 为无用量，1-4 按用量分位递增。
const LEVEL_CLASSES = [
  'bg-gray-100 dark:bg-dark-700',
  'bg-primary-200 dark:bg-primary-900',
  'bg-primary-300 dark:bg-primary-700',
  'bg-primary-500 dark:bg-primary-500',
  'bg-primary-700 dark:bg-primary-300',
]
// 方向键步长：上下为前后一天，左右为前后一周。
const KEY_OFFSETS: Record<string, number> = { ArrowUp: -1, ArrowDown: 1, ArrowLeft: -7, ArrowRight: 7 }
// 月份标签之间至少间隔的列数，过近时用新月份替换旧标签。
const MIN_MONTH_LABEL_GAP = 3
// 提示与卡片左右边缘保持的最小距离。
const TOOLTIP_EDGE_PX = 8

interface HeatmapDay {
  date: string
  weekIndex: number
  dayOfWeek: number
  tokens: number
  requests: number
  actualCost: number
  level: number
  // 补齐最后一周的未来占位格，不展示不响应悬停。
  future?: boolean
}

const emit = defineEmits<{ (event: 'select-day', date: string): void }>()

const { t, locale } = useI18n()
const { selectDay, selectedDay } = injectUsageState()

const loading = ref(true)
const days = ref<HeatmapDay[]>([])
const hoveredDay = ref<HeatmapDay | null>(null)
const cardRef = ref<HTMLElement | null>(null)
const gridWrapRef = ref<HTMLElement | null>(null)
const hoverPos = ref({ left: 0, top: 0, above: false })
const gridWidth = ref(0)
const todayKey = ref(formatDateLocalInput(new Date()))
// 首次拿到数据时格子按列错峰入场，最后一列播放完后关闭。
const waveActive = ref(true)
const focusOverride = ref<string | null>(null)
let resizeObserver: ResizeObserver | undefined

// 按非零用量的 25/50/75 分位划分 1-4 档；峰值日固定为最高档。
const computeLevel = (tokens: number, sortedNonZero: number[]): number => {
  if (tokens <= 0) return 0
  if (tokens >= sortedNonZero[sortedNonZero.length - 1]) return 4
  const percentile = (p: number) => sortedNonZero[Math.floor(sortedNonZero.length * p)]
  if (tokens <= percentile(0.25)) return 1
  if (tokens <= percentile(0.5)) return 2
  if (tokens <= percentile(0.75)) return 3
  return 4
}

const load = async () => {
  loading.value = true
  hoveredDay.value = null
  todayKey.value = formatDateLocalInput(new Date())
  // 起点向前对齐到周日，保证整周列。
  const end = new Date()
  const start = new Date(end)
  start.setDate(start.getDate() - FETCH_DAYS)
  start.setDate(start.getDate() - start.getDay())
  const res = await usageAPI.getDashboardTrend({
    start_date: formatDateLocalInput(start),
    end_date: formatDateLocalInput(end),
    granularity: 'day',
  }).finally(() => {
    loading.value = false
  })
  // 趋势接口只返回有用量的日期，按日期建索引后补零；没有用量时返回 null 列表。
  const points = res.trend ?? []
  const byDate = new Map(points.map((p) => [p.date, p]))
  const sortedNonZero = points.map((p) => p.total_tokens).filter((v) => v > 0).sort((a, b) => a - b)
  const result: HeatmapDay[] = []
  const cursor = new Date(start)
  while (cursor <= end) {
    const date = formatDateLocalInput(cursor)
    const point = byDate.get(date)
    const tokens = point?.total_tokens ?? 0
    result.push({
      date,
      weekIndex: Math.floor(result.length / 7),
      dayOfWeek: cursor.getDay(),
      tokens,
      requests: point?.requests ?? 0,
      actualCost: point?.actual_cost ?? 0,
      level: computeLevel(tokens, sortedNonZero),
    })
    cursor.setDate(cursor.getDate() + 1)
  }
  while (cursor.getDay() !== 0) {
    result.push({ date: formatDateLocalInput(cursor), weekIndex: Math.floor(result.length / 7), dayOfWeek: cursor.getDay(), tokens: 0, requests: 0, actualCost: 0, level: 0, future: true })
    cursor.setDate(cursor.getDate() + 1)
  }
  days.value = result
}

const totalWeeks = computed(() => Math.ceil(days.value.length / 7))

// 可见周数随容器宽度自适应，至少 4 周。
const visibleWeeks = computed(() => {
  const fit = Math.floor((gridWidth.value - LABEL_COL_PX + GAP_PX) / (CELL_PX + GAP_PX))
  return Math.max(4, Math.min(fit, totalWeeks.value || fit))
})

// 只渲染最近 visibleWeeks 周，weekIndex 重新从 0 编排。
const visibleDays = computed(() => days.value.slice(-visibleWeeks.value * 7).map((day, i) => ({ ...day, weekIndex: Math.floor(i / 7) })))

const formatMonth = (date: string) => new Date(`${date}T00:00:00`).toLocaleDateString(locale.value, { month: 'short' })
const formatDayLabel = (date: string) => new Date(`${date}T00:00:00`).toLocaleDateString(locale.value, { year: 'numeric', month: 'short', day: 'numeric' })

// 每周首格（周日）月份与上一列不同才显示月份。
const monthItems = computed(() => {
  const items: { weekIndex: number; label: string }[] = []
  let prevMonth = -1
  for (const day of visibleDays.value) {
    if (day.dayOfWeek !== 0) continue
    const month = new Date(`${day.date}T00:00:00`).getMonth()
    if (month !== prevMonth) {
      const prev = items[items.length - 1]
      if (prev && day.weekIndex - prev.weekIndex < MIN_MONTH_LABEL_GAP) items.pop()
      items.push({ weekIndex: day.weekIndex, label: formatMonth(day.date) })
    }
    prevMonth = month
  }
  return items
})

// 只标周一、周三、周五；2024-01-01 是周一。
const weekdayLabels = computed(() => [1, 3, 5].map((dayOfWeek) => ({
  row: dayOfWeek,
  label: new Date(2024, 0, dayOfWeek).toLocaleDateString(locale.value, { weekday: 'narrow' }),
})))

// 漫游焦点：优先手动移动到的日期，其次选中日，最后今天。
const focusDate = computed(() => {
  const visible = new Set(visibleDays.value.filter((day) => !day.future).map((day) => day.date))
  return [focusOverride.value, selectedDay.value, todayKey.value].find((candidate) => candidate !== null && visible.has(candidate)) ?? null
})

// 按列计算入场延迟；列数很多时压缩步长，最后一列不晚于上限。
const waveDelay = (weekIndex: number): string => {
  const step = Math.min(HEATMAP_WAVE_STEP_MS, HEATMAP_WAVE_MAX_MS / visibleWeeks.value)
  return `${Math.round(weekIndex * step)}ms`
}

const cellClasses = (day: HeatmapDay) => {
  if (day.future) return 'invisible'
  if (loading.value) return 'animate-pulse bg-gray-200 dark:bg-dark-700 motion-reduce:animate-none'
  return [
    LEVEL_CLASSES[day.level],
    'heatmap-cell-interactive',
    {
      'heatmap-cell-enter': waveActive.value,
      'heatmap-cell-today': day.date === todayKey.value,
      'heatmap-cell-selected': day.date === selectedDay.value,
    },
  ]
}

const cellLabel = (day: HeatmapDay): string => {
  const usage = day.requests > 0 ? `${t('gotocc.dashboard.heatmap.requests')} ${formatNumber(day.requests)}` : t('gotocc.dashboard.heatmap.noUsage')
  return `${formatDayLabel(day.date)}, ${usage}`
}

// 把趋势图切到这一天，按小时查看。
const selectCell = (day: HeatmapDay) => {
  if (day.future || loading.value) return
  focusOverride.value = day.date
  selectDay(day.date)
  emit('select-day', day.date)
}

const onGridKeydown = (event: KeyboardEvent) => {
  const date = (event.target as HTMLElement).dataset.date
  if (!date || loading.value) return
  const index = visibleDays.value.findIndex((day) => day.date === date)
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    selectCell(visibleDays.value[index])
    return
  }
  const offset = KEY_OFFSETS[event.key]
  if (offset === undefined) return
  event.preventDefault()
  const next = visibleDays.value[index + offset]
  if (next && !next.future) {
    focusOverride.value = next.date
    gridWrapRef.value?.querySelector<HTMLElement>(`[data-date="${next.date}"]`)?.focus()
  }
}

const onGridFocusOut = (event: FocusEvent) => {
  if (!gridWrapRef.value?.contains(event.relatedTarget as Node | null)) hoveredDay.value = null
}

// 最后一列入场结束后去掉入场类，之后新增的格子（如窗口变宽）直接显示。
const onWaveEnd = (event: AnimationEvent) => {
  if ((event.target as HTMLElement).dataset.waveLast) waveActive.value = false
}

// 以格子中心相对卡片的位置定位提示。
const onCellHover = (day: HeatmapDay, event: Event) => {
  if (day.future || loading.value) return
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const cardRect = cardRef.value!.getBoundingClientRect()
  hoverPos.value = { left: rect.left - cardRect.left + rect.width / 2, top: rect.top - cardRect.top, above: day.dayOfWeek >= 2 }
  hoveredDay.value = day
}

// 提示靠近卡片两侧时向内收，避免超出卡片；宽度按内容自适应，用 translate 居中后再夹在边距内。
const tooltipStyle = computed(() => {
  const cardWidth = cardRef.value!.clientWidth
  const left = Math.min(Math.max(hoverPos.value.left, TOOLTIP_EDGE_PX), cardWidth - TOOLTIP_EDGE_PX)
  const shift = left < cardWidth / 3 ? '0%' : left > (cardWidth * 2) / 3 ? '-100%' : '-50%'
  return {
    left: `${left}px`,
    top: hoverPos.value.above ? `${hoverPos.value.top - 6}px` : `${hoverPos.value.top + 18}px`,
    transform: hoverPos.value.above ? `translate(${shift}, -100%)` : `translateX(${shift})`,
  }
})

onMounted(() => {
  gridWidth.value = gridWrapRef.value!.clientWidth
  resizeObserver = new ResizeObserver((entries) => {
    gridWidth.value = entries[0].contentRect.width
  })
  resizeObserver.observe(gridWrapRef.value!)
  void load()
})

onBeforeUnmount(() => resizeObserver?.disconnect())

defineExpose({ reload: load })
</script>

<style scoped>
.heatmap-cell {
  border-radius: 4px;
}

/* 可点击的格子：悬停或键盘聚焦时放大并加外环 */
.heatmap-cell-interactive {
  cursor: pointer;
  transition: transform var(--motion-fast) var(--motion-ease), box-shadow var(--motion-fast) var(--motion-ease);
}

.heatmap-cell-interactive:hover,
.heatmap-cell-interactive:focus-visible {
  transform: scale(1.25);
  outline: none;
  box-shadow: 0 0 0 1px theme('colors.primary.400');
}

/* 今天与选中日的描边；选中日更粗更深 */
.heatmap-cell-today {
  outline: 1px solid theme('colors.gray.400');
  outline-offset: 1px;
}

.heatmap-cell-selected {
  outline: 2px solid theme('colors.primary.700');
  outline-offset: 1px;
}

:global(.dark) .heatmap-cell-today {
  outline-color: theme('colors.dark.400');
}

:global(.dark) .heatmap-cell-selected {
  outline-color: theme('colors.primary.300');
}

/* 首次入场：按列错峰从小到大淡入 */
.heatmap-cell-enter {
  animation: heatmap-cell-enter var(--dash-heatmap-enter-ms) var(--motion-ease) var(--wave-delay, 0ms) both;
}

@keyframes heatmap-cell-enter {
  from {
    opacity: 0;
    transform: scale(0.4);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .heatmap-cell-enter {
    animation: none;
  }

  .heatmap-cell-interactive:hover,
  .heatmap-cell-interactive:focus-visible {
    transform: none;
  }
}
</style>

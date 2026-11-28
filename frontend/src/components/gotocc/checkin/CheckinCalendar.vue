<template>
  <section class="card cal" :aria-label="monthLabel">
    <header class="cal-head">
      <div class="cal-nav">
        <button type="button" class="cal-step" :aria-label="t('checkin.calendar.prev')" :data-tip="t('checkin.tips.prev')" @click="emit('change', shift(-1))">
          <Icon name="chevronLeft" size="sm" />
        </button>
        <h3 :key="month" class="cal-month gc-reveal">{{ monthLabel }}</h3>
        <button type="button" class="cal-step" :disabled="isCurrent" :aria-label="t('checkin.calendar.next')" :data-tip="t('checkin.tips.next')" @click="emit('change', shift(1))">
          <Icon name="chevronRight" size="sm" />
        </button>
      </div>
      <div class="cal-meta">
        <span :key="month" class="cal-summary gc-reveal">{{ t('checkin.calendar.summary', { days: days.length, amount: formatCurrency(monthReward) }) }}</span>
        <button v-if="!isCurrent" type="button" class="cal-current" :data-tip="t('checkin.tips.current')" @click="emit('change', currentMonth)">
          {{ t('checkin.calendar.current') }}
        </button>
      </div>
    </header>

    <div class="cal-grid cal-weekdays" aria-hidden="true">
      <span v-for="name in weekdays" :key="name">{{ name }}</span>
    </div>
    <div :key="month" class="cal-grid cal-days gc-reveal" :aria-busy="loading">
      <span v-for="index in offset" :key="`blank-${index}`" class="cal-blank"></span>
      <div
        v-for="cell in cells"
        :key="cell.date"
        class="cal-cell"
        :data-state="cell.state"
        :data-today="cell.isToday || undefined"
        :class="{ 'cal-cell-new': cell.date === justSigned }"
        :data-tip="cell.tip"
      >
        <span class="cal-day gc-num">{{ cell.day }}</span>
        <span v-if="cell.record" class="cal-stamp">
          <Icon :name="cell.record.milestone_days ? 'gift' : 'check'" size="xs" :stroke-width="2.6" />
        </span>
        <span v-if="cell.record" class="cal-amount gc-num">+{{ formatCurrency(cell.record.reward) }}</span>
        <span v-else-if="cell.isToday" class="cal-today-label">{{ t('checkin.calendar.today') }}</span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { CheckinDay } from '@/api/checkin'
import { formatCurrency } from '@/utils/format'
import { getLocale } from '@/i18n'

// 月历：周一到周日七列。已签的日子盖一枚渐变勾章（满里程碑的那天是礼物章），今天有呼吸光圈，
// 没到的日子淡出。月份和日期都是站点时区里的日历日，只拿来排版，不做时区换算。
const props = defineProps<{
  month: string
  currentMonth: string
  today: string
  days: CheckinDay[]
  monthReward: number
  loading: boolean
  justSigned: string
}>()
const emit = defineEmits<{ change: [month: string] }>()
const { t } = useI18n()

const isCurrent = computed(() => props.month === props.currentMonth)
const parts = computed(() => props.month.split('-').map(Number) as [number, number])
const monthLabel = computed(() => new Intl.DateTimeFormat(getLocale(), { year: 'numeric', month: 'long' })
  .format(new Date(parts.value[0], parts.value[1] - 1, 1)))
// 2024-01-01 是周一，从它起连取七天得到本地化的周一到周日。
const weekdays = computed(() => Array.from({ length: 7 }, (_, index) =>
  new Intl.DateTimeFormat(getLocale(), { weekday: 'narrow' }).format(new Date(2024, 0, 1 + index))))
const offset = computed(() => (new Date(parts.value[0], parts.value[1] - 1, 1).getDay() + 6) % 7)

function shift(step: number): string {
  const date = new Date(parts.value[0], parts.value[1] - 1 + step, 1)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
}

const cells = computed(() => {
  const [year, month] = parts.value
  const count = new Date(year, month, 0).getDate()
  const records = new Map(props.days.map(day => [day.date, day]))
  const dayLabel = new Intl.DateTimeFormat(getLocale(), { month: 'short', day: 'numeric' })
  return Array.from({ length: count }, (_, index) => {
    const day = index + 1
    const date = `${props.month}-${String(day).padStart(2, '0')}`
    const record = records.get(date)
    const label = dayLabel.format(new Date(year, month - 1, day))
    const isToday = date === props.today
    let state: 'signed' | 'milestone' | 'missed' | 'pending' | 'future' = 'missed'
    let tip = t('checkin.tips.missed', { date: label })
    if (record?.milestone_days) {
      state = 'milestone'
      tip = t('checkin.tips.signedMilestone', { date: label, daily: formatCurrency(record.reward - record.milestone_reward), days: record.milestone_days, bonus: formatCurrency(record.milestone_reward) })
    } else if (record) {
      state = 'signed'
      tip = t('checkin.tips.signed', { date: label, amount: formatCurrency(record.reward) })
    } else if (isToday) {
      state = 'pending'
      tip = t('checkin.tips.todayPending')
    } else if (date > props.today) {
      state = 'future'
      tip = t('checkin.tips.future', { date: label })
    }
    return { date, day, record, isToday, state, tip }
  })
})
</script>

<style scoped>
.cal { display: flex; min-width: 0; flex-direction: column; gap: 12px; padding: 18px 20px 20px; }
.cal-head { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px 16px; }
.cal-nav { display: flex; align-items: center; gap: 6px; }
.cal-month { min-width: 8.5rem; text-align: center; font-size: 16px; font-weight: 700; letter-spacing: -.01em; color: var(--surface-text); }
.cal-step {
  display: grid; height: 32px; width: 32px; place-items: center; border-radius: 10px;
  color: var(--surface-muted); transition: background-color var(--motion-hover) var(--motion-ease), color var(--motion-hover) var(--motion-ease), transform var(--motion-press) var(--motion-ease);
}
.cal-step:hover:not(:disabled) { background: var(--surface-hover); color: var(--surface-text); }
.cal-step:active:not(:disabled) { transform: scale(var(--press-scale)); }
.cal-step:disabled { opacity: .35; cursor: default; }
.cal-meta { display: flex; align-items: center; gap: 10px; font-size: 12px; color: var(--surface-muted); }
.cal-current {
  border-radius: 9999px; padding: 3px 10px; font-weight: 600; color: theme('colors.primary.600');
  background: rgb(99 102 241 / 10%); transition: background-color var(--motion-hover) var(--motion-ease);
}
.cal-current:hover { background: rgb(99 102 241 / 18%); }
.dark .cal-current { color: theme('colors.primary.300'); }
.cal-grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 6px; }
.cal-weekdays > span { padding-bottom: 2px; text-align: center; font-size: 11px; font-weight: 600; letter-spacing: .04em; color: var(--surface-muted); }
.cal-cell {
  position: relative; display: flex; min-width: 0; aspect-ratio: 1 / .82; flex-direction: column; align-items: center; justify-content: center; gap: 3px;
  border-radius: 12px; background: var(--surface-soft);
  transition: transform var(--motion-hover) var(--motion-ease), box-shadow var(--motion-hover) var(--motion-ease), background-color var(--motion-hover) var(--motion-ease);
}
.cal-cell:hover { transform: translateY(var(--hover-lift)); box-shadow: 0 0 0 1px var(--card-hover-edge), 0 8px 18px -12px rgb(49 46 129 / 40%); }
.cal-day { position: absolute; left: 8px; top: 6px; font-size: 12px; font-weight: 600; color: var(--surface-muted); }
.cal-cell[data-state='future'] { background: transparent; box-shadow: inset 0 0 0 1px var(--surface-soft); }
.cal-cell[data-state='future'] .cal-day { opacity: .45; }
.cal-cell[data-state='missed'] .cal-day { opacity: .75; }
.cal-cell[data-state='signed'] {
  background: linear-gradient(145deg, rgb(99 102 241 / 14%), rgb(139 92 246 / 10%));
  box-shadow: inset 0 0 0 1px rgb(99 102 241 / 18%);
}
.cal-cell[data-state='milestone'] {
  background: linear-gradient(145deg, rgb(251 191 36 / 22%), rgb(249 115 22 / 12%));
  box-shadow: inset 0 0 0 1px rgb(251 191 36 / 40%);
}
.cal-cell[data-state='signed'] .cal-day, .cal-cell[data-state='milestone'] .cal-day { color: var(--surface-text); }
.cal-stamp {
  display: grid; height: 26px; width: 26px; place-items: center; border-radius: 9999px; color: white;
  background: linear-gradient(135deg, theme('colors.primary.500'), theme('colors.accent.500'));
  box-shadow: 0 6px 14px -6px rgb(99 102 241 / 70%), inset 0 1px 0 rgb(255 255 255 / 35%);
}
.cal-cell[data-state='milestone'] .cal-stamp {
  background: linear-gradient(135deg, theme('colors.amber.400'), theme('colors.orange.500'));
  box-shadow: 0 6px 14px -6px rgb(234 88 12 / 70%), inset 0 1px 0 rgb(255 255 255 / 40%);
}
.cal-amount { font-size: 11px; font-weight: 650; color: theme('colors.primary.700'); }
.dark .cal-amount { color: theme('colors.primary.200'); }
.cal-cell[data-state='milestone'] .cal-amount { color: theme('colors.amber.700'); }
.dark .cal-cell[data-state='milestone'] .cal-amount { color: theme('colors.amber.200'); }
.cal-today-label { margin-top: 10px; font-size: 11px; font-weight: 650; color: theme('colors.primary.600'); }
.dark .cal-today-label { color: theme('colors.primary.300'); }
/* 今天：一圈品牌色描边，外面一层光慢慢呼吸。 */
.cal-cell[data-today]::after {
  content: ''; position: absolute; inset: -1px; border-radius: inherit; pointer-events: none;
  box-shadow: 0 0 0 2px rgb(99 102 241 / 70%), 0 0 0 0 rgb(99 102 241 / 35%);
  animation: cal-breathe 2.4s var(--motion-ease-fluid) infinite;
}
@keyframes cal-breathe {
  0%, 100% { box-shadow: 0 0 0 2px rgb(99 102 241 / 70%), 0 0 0 0 rgb(99 102 241 / 30%); }
  50% { box-shadow: 0 0 0 2px rgb(99 102 241 / 90%), 0 0 0 7px rgb(99 102 241 / 0%); }
}
.cal-cell-new .cal-stamp { animation: cal-stamp 640ms var(--motion-ease-fluid) both; }
@keyframes cal-stamp {
  0% { transform: scale(2.2) rotate(-18deg); opacity: 0; }
  55% { transform: scale(.9) rotate(4deg); opacity: 1; }
  100% { transform: scale(1) rotate(0); }
}
@media (max-width: 640px) {
  .cal { padding: 14px 12px 14px; }
  .cal-grid { gap: 4px; }
  .cal-cell { aspect-ratio: 1 / 1.05; border-radius: 10px; }
  .cal-day { left: 5px; top: 3px; font-size: 10px; }
  .cal-stamp { height: 20px; width: 20px; margin-top: 6px; }
  .cal-amount, .cal-today-label { display: none; }
}
@media (prefers-reduced-motion: reduce) {
  .cal-cell[data-today]::after, .cal-cell-new .cal-stamp { animation: none; }
}
</style>

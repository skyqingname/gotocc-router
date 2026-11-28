<template>
  <section class="milestones" :aria-label="t('checkin.milestones.title')">
    <div class="mile-head">
      <h3 class="mile-title">{{ t('checkin.milestones.title') }}</h3>
    </div>
    <div class="mile-track" :data-tip="t('checkin.tips.track', { days })">
      <div class="mile-rail"><i :style="{ width: `${fill}%` }"></i></div>
      <span
        v-for="item in milestones"
        :key="item.days"
        class="mile-node"
        :data-state="stateOf(item)"
        :style="{ left: `${(item.days / last) * 100}%` }"
      >
        <Icon v-if="item.reached" name="check" size="xs" :stroke-width="3" />
        <span v-else class="gc-num">{{ item.days }}</span>
      </span>
    </div>
    <div class="mile-cards">
      <div
        v-for="item in milestones"
        :key="item.days"
        class="mile-card"
        :data-state="stateOf(item)"
        :class="{ 'mile-card-pop': item.days === justReached }"
        :data-tip="tipOf(item)"
      >
        <div class="mile-card-top">
          <span class="mile-days">{{ t('checkin.daysUnit', { n: item.days }) }}</span>
          <span class="mile-reward gc-num">+{{ formatCurrency(item.reward) }}</span>
        </div>
        <div class="mile-status">
          <Icon v-if="item.reached" name="checkCircle" size="xs" />
          <span>{{ statusOf(item) }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { CheckinMilestone } from '@/api/checkin'
import { formatCurrency } from '@/utils/format'

// 本月累计奖励：一条按天数等比的进度轨，下面每个节点一张小卡写奖励和还差几天，本月来不及的写明。
const props = defineProps<{ milestones: CheckinMilestone[]; days: number; justReached: number }>()
const { t } = useI18n()
const last = computed(() => props.milestones[props.milestones.length - 1]!.days)
const fill = computed(() => Math.min(props.days / last.value, 1) * 100)

function stateOf(item: CheckinMilestone): 'reached' | 'pending' | 'unreachable' {
  if (item.reached) return 'reached'
  return item.reachable ? 'pending' : 'unreachable'
}

function statusOf(item: CheckinMilestone): string {
  if (item.reached) return t('checkin.milestones.reached')
  return item.reachable ? t('checkin.milestones.remaining', { n: item.remaining }) : t('checkin.milestones.unreachable')
}

function tipOf(item: CheckinMilestone): string {
  const amount = formatCurrency(item.reward)
  if (item.reached) return t('checkin.tips.milestoneReached', { days: item.days, amount })
  return item.reachable ? t('checkin.tips.milestonePending', { n: item.remaining, amount }) : t('checkin.tips.milestoneUnreachable', { days: item.days })
}
</script>

<style scoped>
.milestones { display: flex; flex-direction: column; gap: 14px; }
.mile-head { display: flex; align-items: baseline; justify-content: space-between; }
.mile-title { font-size: 13px; font-weight: 650; color: var(--surface-text); }
.mile-track { position: relative; height: 26px; margin-inline: 13px 13px; }
.mile-rail { position: absolute; left: 0; right: 0; top: 50%; height: 6px; margin-top: -3px; overflow: hidden; border-radius: 9999px; background: var(--surface-soft); }
.mile-rail > i {
  display: block; height: 100%; border-radius: inherit;
  background: linear-gradient(90deg, theme('colors.primary.500'), theme('colors.accent.500'), theme('colors.amber.400'));
  transition: width 900ms var(--motion-ease-fluid);
}
.mile-node {
  position: absolute; top: 50%; display: grid; height: 26px; width: 26px; place-items: center;
  border-radius: 9999px; transform: translate(-50%, -50%);
  font-size: 11px; font-weight: 700; color: var(--surface-muted);
  background: var(--surface-solid); box-shadow: 0 0 0 2px var(--surface-edge);
  transition: box-shadow var(--motion-hover) var(--motion-ease), transform var(--motion-hover) var(--motion-ease);
}
.mile-node[data-state='reached'] {
  color: white; background: linear-gradient(135deg, theme('colors.amber.400'), theme('colors.orange.500'));
  box-shadow: 0 0 0 3px rgb(251 191 36 / 25%), 0 6px 14px -6px rgb(234 88 12 / 70%);
}
.mile-node[data-state='unreachable'] { opacity: .55; }
.mile-track:hover .mile-node { transform: translate(-50%, -50%) scale(1.06); }
.mile-cards { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; }
.mile-card {
  display: flex; min-width: 0; flex-direction: column; gap: 6px; padding: 10px 12px;
  border-radius: 12px; background: var(--surface-soft);
  box-shadow: inset 0 0 0 1px transparent;
  transition: transform var(--motion-hover) var(--motion-ease), box-shadow var(--motion-hover) var(--motion-ease);
}
.mile-card:hover { transform: translateY(var(--hover-lift)); box-shadow: inset 0 0 0 1px var(--card-hover-edge); }
.mile-card[data-state='reached'] {
  background: linear-gradient(135deg, rgb(251 191 36 / 16%), rgb(249 115 22 / 10%));
  box-shadow: inset 0 0 0 1px rgb(251 191 36 / 35%);
}
.mile-card[data-state='unreachable'] { opacity: .6; }
.mile-card-top { display: flex; flex-wrap: wrap; align-items: baseline; justify-content: space-between; gap: 2px 8px; }
.mile-days { font-size: 13px; font-weight: 650; color: var(--surface-text); }
.mile-reward { font-size: 14px; font-weight: 700; color: theme('colors.amber.600'); }
.dark .mile-reward { color: theme('colors.amber.300'); }
.mile-status { display: flex; align-items: center; gap: 4px; font-size: 12px; color: var(--surface-muted); }
.mile-card[data-state='reached'] .mile-status { color: theme('colors.amber.700'); font-weight: 600; }
.dark .mile-card[data-state='reached'] .mile-status { color: theme('colors.amber.300'); }
.mile-card-pop { animation: mile-pop 700ms var(--motion-ease-fluid) both; }
@keyframes mile-pop {
  0% { transform: scale(.92); box-shadow: inset 0 0 0 1px rgb(251 191 36 / 35%), 0 0 0 0 rgb(251 191 36 / 50%); }
  60% { transform: scale(1.04); box-shadow: inset 0 0 0 1px rgb(251 191 36 / 60%), 0 0 0 10px rgb(251 191 36 / 0%); }
  100% { transform: scale(1); }
}
@media (max-width: 640px) {
  .mile-cards { gap: 6px; }
  .mile-card { padding: 8px 9px; }
  .mile-reward { font-size: 13px; }
}
@media (prefers-reduced-motion: reduce) {
  .mile-card-pop { animation: none; }
}
</style>

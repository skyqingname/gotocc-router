<template>
  <AppLayout appearance="user">
    <div class="ck-page" data-test="checkin">
      <div v-if="loadFailed" class="card ck-failed">
        <span>{{ t('checkin.loadFailed') }}</span>
        <button type="button" class="btn btn-secondary btn-sm" @click="load">{{ t('checkin.retry') }}</button>
      </div>

      <template v-else-if="current">
        <section class="card ck-hero">
          <div class="ck-hero-main">
            <div class="ck-intro">
              <h2 class="ck-title">{{ t('checkin.title') }}</h2>
              <p class="ck-lead">
                {{ t('checkin.lead', { amount: formatCurrency(current.daily_reward) }) }}
                {{ t('checkin.leadMilestones', { days: milestoneDaysText, rewards: milestoneRewardsText }) }}
              </p>
            </div>

            <div class="ck-action">
              <CheckinRing
                :days="current.month_days"
                :total="daysInCurrentMonth"
                :label="t('checkin.ring.label')"
                :unit="t('checkin.ring.unit')"
                :tip="t('checkin.tips.ring', { days: current.month_days, total: daysInCurrentMonth })"
              />
              <div class="ck-cta">
                <div class="ck-button-wrap">
                  <RouterLink
                    v-if="!current.unlocked"
                    to="/wallet"
                    class="ck-button"
                    data-state="locked"
                    data-test="checkin-locked"
                    :data-tip="t('checkin.tips.locked')"
                  >
                    <Icon name="lock" size="sm" />
                    <span>{{ t('checkin.button.locked') }}</span>
                    <Icon name="arrowRight" size="sm" class="ck-button-arrow" />
                  </RouterLink>
                  <button
                    v-else-if="!current.checked_in_today"
                    v-support-readonly
                    type="button"
                    class="ck-button"
                    data-state="ready"
                    data-test="checkin-button"
                    :disabled="submitting"
                    :data-tip="t('checkin.tips.checkIn', { amount: formatCurrency(current.today_reward) })"
                    @click="checkIn"
                  >
                    <span class="ck-button-sheen" aria-hidden="true"></span>
                    <Icon :name="current.today_milestone_days ? 'gift' : 'sparkles'" size="sm" />
                    <span>{{ submitting ? t('checkin.button.checking') : t('checkin.button.checkIn', { amount: formatCurrency(current.today_reward) }) }}</span>
                  </button>
                  <div
                    v-else
                    :key="'done'"
                    class="ck-button gc-reveal"
                    data-state="done"
                    data-test="checkin-done"
                    :data-tip="t('checkin.tips.done')"
                  >
                    <span class="ck-done-medal"><Icon name="check" size="xs" :stroke-width="3" /></span>
                    <span>{{ t('checkin.button.done') }}</span>
                  </div>
                  <div v-if="gained" class="ck-burst" aria-hidden="true"><CelebrationBurst /></div>
                  <span v-if="gained" :key="gained.date" class="ck-gained gc-num" data-test="checkin-gained">{{ t('checkin.gained', { amount: formatCurrency(gainedShown) }) }}</span>
                </div>
                <p
                  :key="hintKey"
                  class="ck-hint gc-reveal"
                  :data-state="!current.unlocked ? 'locked' : undefined"
                >{{ hintText }}</p>
              </div>
            </div>
          </div>

          <p v-if="gained?.milestone_days" :key="`ms-${gained.milestone_days}`" class="ck-milestone-banner gc-reveal" data-test="checkin-milestone">
            <Icon name="trophy" size="sm" />
            {{ t('checkin.gainedMilestone', { days: gained.milestone_days, amount: formatCurrency(gained.milestone_reward) }) }}
          </p>

          <CheckinMilestones
            :milestones="current.milestones"
            :days="current.month_days"
            :just-reached="gained?.milestone_days ?? 0"
          />
        </section>

        <div class="ck-body">
          <CheckinCalendar
            :month="viewed!.month"
            :current-month="current.current_month"
            :today="current.today"
            :days="viewed!.days"
            :month-reward="viewed!.month_reward"
            :loading="monthLoading"
            :just-signed="gained?.date ?? ''"
            @change="selectMonth"
          />

          <aside class="ck-side">
            <section class="card ck-stats">
              <div class="ck-stat" :data-tip="t('checkin.tips.monthDays')">
                <span class="ck-stat-icon" data-tone="indigo"><Icon name="calendar" size="sm" /></span>
                <span class="ck-stat-label">{{ t('checkin.stats.monthDays') }}</span>
                <strong class="ck-stat-value gc-num">{{ t('checkin.stats.days', { n: Math.round(monthDaysShown) }) }}</strong>
              </div>
              <div class="ck-stat" :data-tip="t('checkin.tips.monthReward')">
                <span class="ck-stat-icon" data-tone="violet"><Icon name="sparkles" size="sm" /></span>
                <span class="ck-stat-label">{{ t('checkin.stats.monthReward') }}</span>
                <strong class="ck-stat-value gc-num">{{ formatCurrency(monthRewardShown) }}</strong>
              </div>
              <div class="ck-stat" :data-tip="t('checkin.tips.totalReward')">
                <span class="ck-stat-icon" data-tone="amber"><Icon name="trophy" size="sm" /></span>
                <span class="ck-stat-label">{{ t('checkin.stats.totalReward') }}</span>
                <strong class="ck-stat-value gc-num">{{ formatCurrency(totalRewardShown) }}</strong>
              </div>
            </section>

            <section class="card ck-rules">
              <h3 class="ck-rules-title">{{ t('checkin.rules.title') }}</h3>
              <ul>
                <li>{{ t('checkin.rules.daily', { amount: formatCurrency(current.daily_reward) }) }}</li>
                <li v-for="item in current.milestones" :key="item.days">{{ t('checkin.rules.milestone', { days: item.days, amount: formatCurrency(item.reward) }) }}</li>
                <li>{{ t('checkin.rules.month') }}</li>
                <li>{{ t('checkin.rules.unlock') }}</li>
                <li>{{ t('checkin.rules.timezone', { tz: current.timezone }) }}</li>
              </ul>
            </section>
          </aside>
        </div>
      </template>

      <template v-else>
        <div class="gc-skeleton h-[260px] w-full"></div>
        <div class="gc-skeleton h-[420px] w-full"></div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import CelebrationBurst from '@/components/gotocc/onboarding/CelebrationBurst.vue'
import CheckinRing from '@/components/gotocc/checkin/CheckinRing.vue'
import CheckinMilestones from '@/components/gotocc/checkin/CheckinMilestones.vue'
import CheckinCalendar from '@/components/gotocc/checkin/CheckinCalendar.vue'
import { useCountUp } from '@/components/gotocc/dashboard/useCountUp'
import { checkinAPI, type CheckinOverview, type CheckinResult } from '@/api/checkin'
import { supportReadonly as vSupportReadonly } from '@/directives/supportReadonly'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatCurrency } from '@/utils/format'

// 签到中心：上面一张主卡是本月（进度环、签到按钮、每月累计奖励），下面左边月历可切月，
// 右边是本月签到、本月获得、累计获得与规则。规则和金额全部来自接口（根目录 daily-checkin.json）。
const { t } = useI18n()
const app = useAppStore()
const auth = useAuthStore()

const current = shallowRef<CheckinOverview | null>(null)
const viewed = shallowRef<CheckinOverview | null>(null)
const loadFailed = ref(false)
const monthLoading = ref(false)
const submitting = ref(false)
const gained = shallowRef<CheckinResult | null>(null)

const daysInCurrentMonth = computed(() => {
  const [year, month] = current.value!.current_month.split('-').map(Number) as [number, number]
  return new Date(year, month, 0).getDate()
})
const milestoneDaysText = computed(() => current.value!.milestones.map(item => t('checkin.daysUnit', { n: item.days })).join(t('checkin.listSeparator')))
const milestoneRewardsText = computed(() => current.value!.milestones.map(item => formatCurrency(item.reward)).join(t('checkin.listSeparator')))

const hintKey = computed(() => !current.value!.unlocked ? 'locked' : current.value!.checked_in_today ? 'done' : 'ready')
const hintText = computed(() => {
  const view = current.value!
  if (!view.unlocked) return t('checkin.hint.locked')
  if (view.checked_in_today) return t('checkin.hint.done')
  if (view.today_milestone_days) return t('checkin.hint.milestoneToday', { days: view.today_milestone_days, amount: formatCurrency(view.today_reward) })
  return t('checkin.hint.ready')
})

const monthDaysShown = useCountUp(() => current.value?.month_days ?? 0, 700)
const monthRewardShown = useCountUp(() => current.value?.month_reward ?? 0, 900)
const totalRewardShown = useCountUp(() => current.value?.total_reward ?? 0, 900)
const gainedShown = useCountUp(() => gained.value?.total ?? 0, 900)

async function load() {
  loadFailed.value = false
  try {
    const overview = await checkinAPI.getOverview()
    current.value = overview
    if (!viewed.value || viewed.value.month === overview.month) viewed.value = overview
    else viewed.value = await checkinAPI.getOverview(viewed.value.month)
  } catch (err) {
    loadFailed.value = true
    app.showError(extractApiErrorMessage(err, t('checkin.loadFailed')))
  }
}

async function selectMonth(month: string) {
  if (month === current.value!.month) {
    viewed.value = current.value
    return
  }
  monthLoading.value = true
  try {
    viewed.value = await checkinAPI.getOverview(month)
  } catch (err) {
    app.showError(extractApiErrorMessage(err, t('checkin.loadFailed')))
  } finally {
    monthLoading.value = false
  }
}

async function checkIn() {
  submitting.value = true
  try {
    gained.value = await checkinAPI.checkIn()
    if (auth.user) auth.user = { ...auth.user, balance: gained.value.balance }
    await Promise.all([load(), auth.refreshUser()])
  } catch (err) {
    app.showError(extractApiErrorMessage(err, t('checkin.loadFailed')))
    await load()
  } finally {
    submitting.value = false
  }
}

onMounted(() => { void load() })
</script>

<style scoped>
.ck-page { display: flex; width: 100%; max-width: 72rem; flex-direction: column; gap: 14px; margin-inline: auto; }
.ck-failed { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 10px; padding: 14px 16px; font-size: 13px; color: var(--surface-muted); }

.ck-hero { position: relative; display: flex; flex-direction: column; gap: 20px; overflow: hidden; padding: 24px 26px 22px; }
.ck-hero::before {
  content: ''; position: absolute; inset: 0; pointer-events: none; border-radius: inherit;
  background:
    radial-gradient(30rem 14rem at 0% 0%, rgb(99 102 241 / 14%), transparent 70%),
    radial-gradient(22rem 12rem at 100% 0%, rgb(251 191 36 / 12%), transparent 70%),
    radial-gradient(18rem 10rem at 70% 100%, rgb(139 92 246 / 10%), transparent 70%);
}
.ck-hero > * { position: relative; }
.ck-hero-main { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 20px 32px; }
.ck-intro { min-width: 0; max-width: 34rem; flex: 1 1 18rem; }
.ck-title { font-size: 24px; font-weight: 750; letter-spacing: -.015em; color: var(--surface-text); }
.ck-lead { margin-top: 6px; font-size: 13px; line-height: 1.75; color: var(--surface-muted); }
.ck-action { display: flex; align-items: center; gap: 22px; }
.ck-cta { display: flex; min-width: 14rem; flex-direction: column; align-items: flex-start; gap: 8px; }
.ck-button-wrap { position: relative; }

.ck-button {
  position: relative; display: inline-flex; height: 48px; align-items: center; gap: 8px; overflow: hidden;
  border-radius: 14px; padding-inline: 20px; font-size: 15px; font-weight: 700; white-space: nowrap;
  transition: transform var(--motion-hover) var(--motion-ease), box-shadow var(--motion-hover) var(--motion-ease), background-color var(--motion-hover) var(--motion-ease);
}
.ck-button[data-state='ready'] {
  color: white;
  background: linear-gradient(135deg, theme('colors.primary.500'), theme('colors.accent.500'));
  box-shadow: 0 14px 30px -14px rgb(99 102 241 / 80%), inset 0 1px 0 rgb(255 255 255 / 25%);
}
.ck-button[data-state='ready']:hover:not(:disabled) { transform: translateY(-2px); box-shadow: 0 18px 34px -14px rgb(99 102 241 / 90%), inset 0 1px 0 rgb(255 255 255 / 25%); }
.ck-button[data-state='ready']:active:not(:disabled) { transform: scale(var(--press-scale)); }
.ck-button[data-state='ready']:disabled { opacity: .8; cursor: progress; }
.ck-button-sheen {
  position: absolute; inset: 0; pointer-events: none;
  background: linear-gradient(105deg, transparent 30%, rgb(255 255 255 / 35%) 48%, transparent 66%);
  transform: translateX(-120%); animation: ck-sheen 3.2s var(--motion-ease-fluid) 1.2s infinite;
}
@keyframes ck-sheen { 0%, 60% { transform: translateX(-120%); } 100% { transform: translateX(120%); } }
.ck-button[data-state='done'] {
  color: theme('colors.emerald.700'); background: rgb(16 185 129 / 12%);
  box-shadow: inset 0 0 0 1px rgb(16 185 129 / 30%); cursor: default;
}
.dark .ck-button[data-state='done'] { color: theme('colors.emerald.300'); background: rgb(16 185 129 / 14%); }
.ck-done-medal {
  display: grid; height: 22px; width: 22px; place-items: center; border-radius: 9999px; color: white;
  background: linear-gradient(135deg, theme('colors.emerald.400'), theme('colors.teal.500'));
  animation: ck-medal 520ms var(--motion-ease-fluid) both;
}
@keyframes ck-medal { 0% { transform: scale(.2) rotate(-40deg); } 70% { transform: scale(1.15) rotate(6deg); } 100% { transform: scale(1) rotate(0); } }
.ck-button[data-state='locked'] {
  color: var(--surface-text); background: var(--surface-solid);
  box-shadow: inset 0 0 0 1px var(--surface-edge), 0 8px 20px -14px rgb(49 46 129 / 40%);
}
.ck-button[data-state='locked']:hover { transform: translateY(-1px); box-shadow: inset 0 0 0 1px theme('colors.amber.400'), 0 12px 24px -14px rgb(217 119 6 / 50%); }
.ck-button-arrow { transition: transform var(--motion-hover) var(--motion-ease); }
.ck-button[data-state='locked']:hover .ck-button-arrow { transform: translateX(var(--icon-nudge)); }
/* 撒花只留光圈和光点，从按钮中心散开；勋章由按钮自己的对勾承担。 */
.ck-burst { position: absolute; left: 50%; top: 50%; pointer-events: none; transform: translate(-50%, -50%) scale(1.9); animation: ck-burst-out 1.6s ease forwards; }
.ck-burst :deep(.burst-medal) { visibility: hidden; }
@keyframes ck-burst-out { 0%, 70% { opacity: 1; } 100% { opacity: 0; } }
.ck-gained {
  position: absolute; left: calc(100% + 10px); top: 50%; margin-top: -12px; border-radius: 9999px; padding: 2px 9px;
  font-size: 13px; font-weight: 750; color: white; pointer-events: none;
  background: linear-gradient(135deg, theme('colors.amber.400'), theme('colors.orange.500'));
  box-shadow: 0 8px 18px -8px rgb(234 88 12 / 70%);
  animation: ck-gained 2.6s var(--motion-ease-fluid) forwards;
}
@keyframes ck-gained {
  0% { opacity: 0; transform: translateY(8px) scale(.8); }
  12% { opacity: 1; transform: translateY(-4px) scale(1.05); }
  20% { transform: translateY(0) scale(1); }
  80% { opacity: 1; }
  100% { opacity: 0; transform: translateY(-10px); }
}
.ck-hint { font-size: 12px; line-height: 1.6; color: var(--surface-muted); }
.ck-hint[data-state='locked'] { font-weight: 600; color: theme('colors.amber.600'); }
.dark .ck-hint[data-state='locked'] { color: theme('colors.amber.300'); }
.ck-milestone-banner {
  display: flex; align-items: center; gap: 8px; border-radius: 12px; padding: 10px 14px;
  font-size: 13px; font-weight: 650; color: theme('colors.amber.800');
  background: linear-gradient(90deg, rgb(251 191 36 / 20%), rgb(249 115 22 / 8%));
  box-shadow: inset 0 0 0 1px rgb(251 191 36 / 35%);
}
.dark .ck-milestone-banner { color: theme('colors.amber.200'); }

.ck-body { display: grid; grid-template-columns: minmax(0, 1fr) 19rem; align-items: start; gap: 14px; }
.ck-side { display: flex; min-width: 0; flex-direction: column; gap: 14px; }
.ck-stats { display: flex; flex-direction: column; gap: 4px; padding: 10px; }
.ck-stat {
  display: grid; grid-template-columns: auto 1fr auto; align-items: center; gap: 10px; border-radius: 12px; padding: 10px;
  transition: background-color var(--motion-hover) var(--motion-ease);
}
.ck-stat:hover { background: var(--surface-hover); }
.ck-stat-icon { display: grid; height: 32px; width: 32px; place-items: center; border-radius: 10px; }
.ck-stat-icon[data-tone='indigo'] { color: theme('colors.primary.600'); background: rgb(99 102 241 / 12%); }
.ck-stat-icon[data-tone='violet'] { color: theme('colors.accent.600'); background: rgb(139 92 246 / 12%); }
.ck-stat-icon[data-tone='amber'] { color: theme('colors.amber.600'); background: rgb(251 191 36 / 16%); }
.dark .ck-stat-icon[data-tone='indigo'] { color: theme('colors.primary.300'); }
.dark .ck-stat-icon[data-tone='violet'] { color: theme('colors.accent.300'); }
.dark .ck-stat-icon[data-tone='amber'] { color: theme('colors.amber.300'); }
.ck-stat-label { font-size: 13px; color: var(--surface-muted); }
.ck-stat-value { font-size: 17px; font-weight: 750; color: var(--surface-text); }
.ck-rules { padding: 16px 18px; }
.ck-rules-title { font-size: 13px; font-weight: 650; color: var(--surface-text); }
.ck-rules ul { margin-top: 8px; display: flex; flex-direction: column; gap: 6px; }
.ck-rules li { position: relative; padding-left: 14px; font-size: 12px; line-height: 1.6; color: var(--surface-muted); }
.ck-rules li::before { content: ''; position: absolute; left: 2px; top: .62em; height: 5px; width: 5px; border-radius: 9999px; background: linear-gradient(135deg, theme('colors.primary.400'), theme('colors.accent.400')); }

@media (max-width: 1024px) {
  .ck-body { grid-template-columns: minmax(0, 1fr); }
  .ck-side { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 640px) {
  .ck-hero { padding: 18px 16px; gap: 16px; }
  .ck-title { font-size: 20px; }
  .ck-action { width: 100%; gap: 14px; }
  .ck-cta { min-width: 0; flex: 1; }
  .ck-button { height: 44px; padding-inline: 14px; font-size: 14px; }
  .ck-side { grid-template-columns: minmax(0, 1fr); }
}
@media (prefers-reduced-motion: reduce) {
  .ck-button-sheen, .ck-burst, .ck-done-medal { animation: none; }
  .ck-burst { display: none; }
}
</style>

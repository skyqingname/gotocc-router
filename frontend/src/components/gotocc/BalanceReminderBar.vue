<template>
  <div v-if="visible" class="balance-reminder gc-reveal" role="status" data-test="balance-reminder">
    <span class="balance-reminder-icon" aria-hidden="true"><Icon name="exclamationTriangle" size="sm" /></span>
    <p class="balance-reminder-text">
      {{ t('consoleGuide.balance.message', { balance: withCreditUnit(balance.toFixed(2)), threshold: withCreditUnit(threshold.toFixed(2)) }) }}
    </p>
    <div class="balance-reminder-actions">
      <RouterLink :to="rechargePath" class="btn btn-primary btn-sm" data-test="balance-recharge">{{ t('consoleGuide.balance.recharge') }}</RouterLink>
      <RouterLink v-if="emailNotifyAvailable" :to="{ path: '/profile', hash: '#balance-notify' }" class="btn btn-secondary btn-sm" data-test="balance-email">
        {{ t('consoleGuide.balance.email') }}
      </RouterLink>
      <button type="button" class="balance-reminder-close" :aria-label="t('consoleGuide.balance.dismiss')" data-test="balance-dismiss" @click="dismiss">
        <Icon name="x" size="sm" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { withCreditUnit } from '@/utils/credits'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import consoleGuide from '../../../../console-guide.json'

// 控制台顶部的余额提醒：余额低于提醒线时出现。提醒线用用户自己设的，没设就用站点设置
// （站点关闭余额提醒时不用站点默认值），百分比阈值按累计充值折算，与邮件提醒同一规则。
// 关闭时记下当时的余额；之后余额高过这个数（充过值）才再次提醒。记录只在本浏览器。

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const user = computed(() => authStore.user!)
const balance = computed(() => user.value.balance)
const settings = computed(() => appStore.cachedPublicSettings)
const emailNotifyAvailable = computed(() => settings.value?.balance_low_notify_enabled === true)

const threshold = computed(() => {
  const own = user.value.balance_notify_threshold
  const base = own !== null ? own : emailNotifyAvailable.value ? settings.value!.balance_low_notify_threshold : 0
  return user.value.balance_notify_threshold_type === 'percentage' && user.value.total_recharged > 0
    ? (user.value.total_recharged * base) / 100
    : base
})

const storageKey = computed(() => `${consoleGuide.balance_reminder.storage_prefix}${user.value.id}`)
const dismissedAt = ref<number | null>(null)
watch(storageKey, (key) => {
  const saved = localStorage.getItem(key)
  dismissedAt.value = saved === null ? null : Number(saved)
}, { immediate: true })

// 余额比关闭时高，说明充过值：清掉记录，下次低于提醒线时重新提醒。
watch([balance, dismissedAt], ([value, at]) => {
  if (at !== null && value > at) {
    localStorage.removeItem(storageKey.value)
    dismissedAt.value = null
  }
}, { immediate: true })

const visible = computed(() => threshold.value > 0 && balance.value < threshold.value && dismissedAt.value === null)

// 去充值：开着在线支付就去购买页，否则去兑换（与头像菜单的入口条件相同）。
const rechargePath = computed(() => (isFeatureFlagEnabled(FeatureFlags.payment) ? '/purchase' : '/redeem'))

function dismiss() {
  localStorage.setItem(storageKey.value, String(balance.value))
  dismissedAt.value = balance.value
}
</script>

<style scoped>
.balance-reminder {
  display: flex; flex-wrap: wrap; align-items: center; gap: 8px 12px;
  padding: 10px 12px 10px 14px; border: 1px solid rgb(245 158 11 / 35%); border-radius: var(--surface-radius-card);
  background: linear-gradient(90deg, rgb(245 158 11 / 14%), rgb(245 158 11 / 6%)), var(--surface-solid);
}
.balance-reminder-icon { display: grid; height: 28px; width: 28px; flex: none; place-items: center; border-radius: 9999px; background: rgb(245 158 11 / 18%); color: theme('colors.amber.600'); }
.dark .balance-reminder-icon { color: theme('colors.amber.300'); }
.balance-reminder-text { min-width: 12rem; flex: 1 1 18rem; font-size: 13px; font-weight: 500; color: var(--surface-text); font-variant-numeric: tabular-nums; }
.balance-reminder-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; }
.balance-reminder-close { display: grid; height: 28px; width: 28px; place-items: center; border-radius: var(--surface-radius-tag); color: var(--surface-muted); }
.balance-reminder-close:hover { background: var(--surface-hover); color: var(--surface-text); }
</style>

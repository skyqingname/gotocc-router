<template>
  <AppLayout appearance="user" workspace>
    <div class="workspace-page wallet-page">
      <!-- 顶部：可用余额和四个选项卡。选项卡记在地址栏 ?tab=，刷新、分享、支付回跳都回到同一处。 -->
      <section class="card workspace-fixed wallet-hero">
        <div class="wallet-hero-main">
          <div class="wallet-balance" :data-tip="balanceTip">
            <span class="wallet-balance-label">{{ t('walletCenter.balance') }}</span>
            <strong class="wallet-balance-value gc-num">
              <span aria-hidden="true">{{ money(animatedBalance) }}</span>
              <span class="sr-only">{{ money(available) }}</span>
            </strong>
            <span v-if="frozen > 0" class="wallet-balance-frozen" :data-tip="t('walletCenter.tips.frozen', { amount: exactCredit(frozen) })">
              {{ t('walletCenter.frozen', { amount: money(frozen) }) }}
            </span>
          </div>
          <p class="wallet-hero-lead">{{ paymentEnabled ? t('walletCenter.leadPayment') : t('walletCenter.leadRedeem') }}</p>
        </div>
        <WorkspaceTabs v-model="tab" embedded class="wallet-tabs" :tabs="tabs" :label="t('nav.wallet')" />
      </section>

      <div :key="tab" class="workspace-body wallet-body gc-reveal">
        <PaymentView v-if="tab === 'recharge'" tab="recharge" @update:tab="tab = $event" />
        <PaymentView v-else-if="tab === 'subscription' && paymentEnabled" tab="subscription" @update:tab="tab = $event">
          <template #lead><SubscriptionsView /></template>
        </PaymentView>
        <div v-else-if="tab === 'subscription'" class="workspace-scroll wallet-scroll"><SubscriptionsView /></div>
        <UserOrdersView v-else-if="tab === 'orders'" />
        <div v-else class="workspace-scroll wallet-scroll"><RedeemView /></div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import '@/styles/wallet.css'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import WorkspaceTabs from '@/components/common/WorkspaceTabs.vue'
import PaymentView from './PaymentView.vue'
import SubscriptionsView from './SubscriptionsView.vue'
import UserOrdersView from './UserOrdersView.vue'
import RedeemView from './RedeemView.vue'
import { useUserView } from '@/composables/useUserView'
import { useCountUp } from '@/components/gotocc/dashboard/useCountUp'
import { COUNT_UP_MS } from '@/components/gotocc/dashboard/motion'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { exactCredit } from '@/utils/consoleTips'
import { withCreditUnit } from '@/utils/credits'
import type { WalletTab } from '@/router/walletRedirect'

// 充值与订阅：原「充值/订阅」「我的订阅」「我的订单」「兑换」四页合成一个入口。
// 充值、订单要开着在线支付；订阅要开着订阅功能，其中可购买的套餐也要开着在线支付；兑换始终有。
// 默认选项卡：支付开着是充值，关着是兑换。
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const view = useUserView()

const paymentEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.payment))
const subscriptionEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.subscription))

const tabs = computed(() => [
  ...(paymentEnabled.value ? [{ value: 'recharge' as WalletTab, label: t('walletCenter.tabs.recharge'), tip: t('walletCenter.tips.recharge') }] : []),
  ...(subscriptionEnabled.value ? [{ value: 'subscription' as WalletTab, label: t('walletCenter.tabs.subscription'), tip: t('walletCenter.tips.subscription') }] : []),
  ...(paymentEnabled.value ? [{ value: 'orders' as WalletTab, label: t('walletCenter.tabs.orders'), tip: t('walletCenter.tips.orders') }] : []),
  { value: 'redeem' as WalletTab, label: t('walletCenter.tabs.redeem'), tip: t('walletCenter.tips.redeem') },
])

// 换选项卡时只留 tab，支付回跳、续费分组这类一次性参数不跟到别的选项卡。
const tab = computed<WalletTab>({
  get: () => tabs.value.find(item => item.value === route.query.tab)?.value ?? (paymentEnabled.value ? 'recharge' : 'redeem'),
  set: value => { void router.replace({ path: route.path, query: { tab: value } }) },
})

const available = computed(() => Number(view.user?.balance ?? 0))
const frozen = computed(() => Number(view.user?.frozen_balance ?? 0))
const animatedBalance = useCountUp(available, COUNT_UP_MS)
const money = (value: number) => withCreditUnit(value.toFixed(2))
const balanceTip = computed(() => t('consoleTips.header.balance', {
  available: exactCredit(available.value),
  frozen: exactCredit(frozen.value),
  total: exactCredit(available.value + frozen.value),
}))

onMounted(() => { void view.refreshUser() })
</script>

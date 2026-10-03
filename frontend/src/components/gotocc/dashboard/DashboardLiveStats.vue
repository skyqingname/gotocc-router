<template>
  <div class="flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-gray-500 dark:text-dark-400" :aria-busy="!loaded">
    <span v-for="item in items" :key="item.key" class="inline-flex items-center gap-2" :title="item.hint">
      <span class="flex h-6 w-6 items-center justify-center rounded-md" :class="[METRIC_TONES[item.tone].tile, METRIC_TONES[item.tone].icon]">
        <Icon :name="item.icon" size="xs" />
      </span>
      <span>{{ item.label }}</span>
      <Skeleton v-if="!loaded" :width="40" :height="12" />
      <span v-else class="font-semibold tabular-nums text-gray-900 dark:text-white">
        <span aria-hidden="true">{{ item.animated }}</span>
        <span class="sr-only">{{ item.value }}</span>
      </span>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import { formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import { COUNT_UP_MS } from './motion'
import { METRIC_TONES, type MetricTone } from './tones'
import { useCountUp } from './useCountUp'
import { formatUsd } from './format'

// 实时状态条：RPM/TPM 是接口给出的近 5 分钟均值，页面可见时每分钟刷新一次。结构参照 TokenRouter（LGPL-3.0）。
const POLL_INTERVAL_MS = 60 * 1000

const { t } = useI18n()

const stats = ref<UserDashboardStats | null>(null)
const loaded = ref(false)
let timer: ReturnType<typeof setInterval> | undefined

const animatedRpm = useCountUp(() => stats.value?.rpm ?? 0, COUNT_UP_MS)
const animatedTpm = useCountUp(() => stats.value?.tpm ?? 0, COUNT_UP_MS)
const animatedLatency = useCountUp(() => stats.value?.average_duration_ms ?? 0, COUNT_UP_MS)
const animatedTodayCost = useCountUp(() => stats.value?.today_actual_cost ?? 0, COUNT_UP_MS)

const formatRpm = (value: number) => formatNumber(Math.round(value * 10) / 10)
const formatLatency = (ms: number) => (ms < 1000 ? `${Math.round(ms)}ms` : `${(ms / 1000).toFixed(2)}s`)
const formatCost = (value: number) => formatUsd(value, value >= 1 ? 2 : 4)

type LiveIcon = 'swap' | 'cube' | 'clock' | 'dollar'

const items = computed(() => {
  const current = stats.value
  return [
    { key: 'rpm', icon: 'swap' as LiveIcon, tone: 'requests' as MetricTone, label: t('gotocc.dashboard.live.rpm'), hint: t('gotocc.dashboard.live.rpmHint'), value: formatRpm(current?.rpm ?? 0), animated: formatRpm(animatedRpm.value) },
    { key: 'tpm', icon: 'cube' as LiveIcon, tone: 'tokens' as MetricTone, label: t('gotocc.dashboard.live.tpm'), hint: t('gotocc.dashboard.live.tpmHint'), value: formatTokensK(current?.tpm ?? 0), animated: formatTokensK(animatedTpm.value) },
    { key: 'latency', icon: 'clock' as LiveIcon, tone: 'latency' as MetricTone, label: t('gotocc.dashboard.live.latency'), hint: undefined, value: formatLatency(current?.average_duration_ms ?? 0), animated: formatLatency(animatedLatency.value) },
    { key: 'todayCost', icon: 'dollar' as LiveIcon, tone: 'cost' as MetricTone, label: t('gotocc.dashboard.live.todayCost'), hint: undefined, value: formatCost(current?.today_actual_cost ?? 0), animated: formatCost(animatedTodayCost.value) },
  ]
})

const load = async () => {
  stats.value = await usageAPI.getDashboardStats().finally(() => {
    loaded.value = true
  })
}

// 页面隐藏时暂停轮询，回到页面时立即刷新再恢复。
const startPolling = () => {
  clearInterval(timer)
  timer = setInterval(() => void load(), POLL_INTERVAL_MS)
}

const onVisibilityChange = () => {
  if (document.visibilityState === 'hidden') {
    clearInterval(timer)
    return
  }
  void load()
  startPolling()
}

onMounted(() => {
  void load()
  startPolling()
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onBeforeUnmount(() => {
  clearInterval(timer)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})

defineExpose({ reload: load })
</script>

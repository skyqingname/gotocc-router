<template>
  <AppLayout>
    <!-- 首次进入时各区块按 --rise-i 依次上移淡入，最后一块播放完后移除入场类 -->
    <div class="space-y-4" :class="{ 'dash-rise': entering }" :style="dashboardMotionVars" @animationend="onRiseEnd">
      <div class="dash-rise-item flex flex-wrap items-center justify-between gap-3" :style="{ '--rise-i': 0 }">
        <DashboardLiveStats ref="liveStatsRef" />
        <DashboardToolbar :refreshing="refreshing" @refresh="refreshAll" />
      </div>
      <DashboardUsageChart ref="usageChartRef" />
      <div class="dash-rise-item" :style="{ '--rise-i': 3 }">
        <DashboardHeatmap ref="heatmapRef" @select-day="revealChart" />
      </div>
      <div v-if="!view.isSimpleMode" class="dash-rise-item" :style="{ '--rise-i': 4 }">
        <DashboardPlatformUsage ref="platformRef" />
      </div>
      <div class="dash-rise-item grid grid-cols-1 gap-4 lg:grid-cols-3" :style="{ '--rise-i': 5 }" data-rise-last="true">
        <div class="space-y-4 lg:col-span-2">
          <UserDashboardRecentUsage :data="recentUsage" :loading="recentLoading" />
          <DashboardAnnouncements />
        </div>
        <div>
          <UserDashboardQuickActions />
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import DashboardAnnouncements from '@/components/gotocc/dashboard/DashboardAnnouncements.vue'
import DashboardHeatmap from '@/components/gotocc/dashboard/DashboardHeatmap.vue'
import DashboardLiveStats from '@/components/gotocc/dashboard/DashboardLiveStats.vue'
import DashboardPlatformUsage from '@/components/gotocc/dashboard/DashboardPlatformUsage.vue'
import DashboardToolbar from '@/components/gotocc/dashboard/DashboardToolbar.vue'
import DashboardUsageChart from '@/components/gotocc/dashboard/DashboardUsageChart.vue'
import { dashboardMotionVars } from '@/components/gotocc/dashboard/motion'
import { provideUsageState } from '@/components/gotocc/dashboard/usageState'
import { usageAPI } from '@/api/usage'
import { useUserView } from '@/composables/useUserView'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatDateLocalInput } from '@/utils/format'
import type { UsageLog } from '@/types'

// GoToCC 用户仪表盘：布局与动效参照 TokenRouter（LGPL-3.0），保留平台额度、最近使用和快捷操作。
// 最近使用固定展示近 7 天，与卡片标题一致。
const RECENT_DAYS = 7
const RECENT_COUNT = 5

const view = useUserView()
const announcementStore = useAnnouncementStore()
const usageState = provideUsageState()

const liveStatsRef = ref<InstanceType<typeof DashboardLiveStats> | null>(null)
const usageChartRef = ref<InstanceType<typeof DashboardUsageChart> | null>(null)
const heatmapRef = ref<InstanceType<typeof DashboardHeatmap> | null>(null)
const platformRef = ref<InstanceType<typeof DashboardPlatformUsage> | null>(null)
const refreshing = ref(false)
// 入场动画只在首次挂载时播放。
const entering = ref(true)
const recentUsage = ref<UsageLog[]>([])
const recentLoading = ref(false)

const loadRecent = async () => {
  recentLoading.value = true
  const start = new Date()
  start.setDate(start.getDate() - (RECENT_DAYS - 1))
  const result = await usageAPI.getByDateRange(formatDateLocalInput(start), formatDateLocalInput(new Date())).finally(() => {
    recentLoading.value = false
  })
  recentUsage.value = result.items.slice(0, RECENT_COUNT)
}

// 刷新整个仪表盘，顶栏余额随账户信息一起更新；平台区在简易模式下不挂载。
const refreshAll = async () => {
  refreshing.value = true
  await Promise.all([
    view.refreshUser(),
    usageState.load(),
    liveStatsRef.value!.reload(),
    heatmapRef.value!.reload(),
    platformRef.value?.reload(),
    loadRecent(),
    announcementStore.fetchAnnouncements(true),
  ]).finally(() => {
    refreshing.value = false
  })
}

// 热力图选中某天后，把趋势图滚动到可见区域。
const revealChart = () => {
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  usageChartRef.value!.chartCardRef!.scrollIntoView({ behavior: reduced ? 'auto' : 'smooth', block: 'center' })
}

const onRiseEnd = (event: AnimationEvent) => {
  if ((event.target as HTMLElement).dataset.riseLast) entering.value = false
}

onMounted(() => {
  void view.refreshUser()
  void usageState.load()
  void usageState.loadFilterOptions()
  void loadRecent()
  void announcementStore.fetchAnnouncements()
})
</script>

<style scoped>
/* 区块入场：自下方 8px 上移并淡入，相邻区块错开一个步长；结束后不保留 transform，避免影响内部浮层定位。 */
.dash-rise :deep(.dash-rise-item) {
  animation: dash-rise var(--dash-rise-ms) var(--motion-ease) backwards;
  animation-delay: calc(var(--rise-i, 0) * var(--dash-rise-step-ms));
}

@keyframes dash-rise {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .dash-rise :deep(.dash-rise-item) {
    animation: none;
  }
}
</style>

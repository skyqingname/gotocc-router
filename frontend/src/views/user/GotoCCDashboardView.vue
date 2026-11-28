<template>
  <AppLayout>
    <!-- 首次进入时各区块按 --rise-i 依次上移淡入，最后一块播放完后移除入场类 -->
    <div class="space-y-4" :class="{ 'dash-rise': entering }" :style="dashboardMotionVars" @animationend="onRiseEnd">
      <SetupGuideCard />
      <!-- 重点：余额、今天、服务质量 -->
      <div class="dash-rise-item" :style="{ '--rise-i': 0 }">
        <DashboardFocus ref="focusRef" />
      </div>
      <!-- 区间：随右侧时间范围和筛选变化 -->
      <div class="dash-rise-item flex flex-wrap items-center justify-between gap-3 pt-2" :style="{ '--rise-i': 1 }">
        <div class="min-w-0">
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboardFocus.range.title') }}</h2>
          <p class="text-xs surface-muted">{{ t('dashboardFocus.range.lead') }}</p>
        </div>
        <DashboardToolbar :refreshing="refreshing" @refresh="refreshAll" />
      </div>
      <DashboardUsageChart ref="usageChartRef" />
      <div class="dash-rise-item" :style="{ '--rise-i': 4 }">
        <DashboardHeatmap ref="heatmapRef" @select-day="revealChart" />
      </div>
      <div class="dash-rise-item grid grid-cols-1 gap-4 lg:grid-cols-3" :style="{ '--rise-i': 5 }" :data-rise-last="view.isSimpleMode || undefined">
        <div class="space-y-4 lg:col-span-2">
          <DashboardRecentRequests ref="recentRef" />
          <DashboardAnnouncements />
        </div>
        <div class="space-y-4">
          <DashboardQuickActions />
          <DashboardContact />
        </div>
      </div>
      <div v-if="!view.isSimpleMode" class="dash-rise-item" :style="{ '--rise-i': 6 }" data-rise-last="true">
        <DashboardPlatformUsage ref="platformRef" />
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DashboardAnnouncements from '@/components/gotocc/dashboard/DashboardAnnouncements.vue'
import DashboardContact from '@/components/gotocc/dashboard/DashboardContact.vue'
import DashboardFocus from '@/components/gotocc/dashboard/DashboardFocus.vue'
import DashboardHeatmap from '@/components/gotocc/dashboard/DashboardHeatmap.vue'
import DashboardPlatformUsage from '@/components/gotocc/dashboard/DashboardPlatformUsage.vue'
import DashboardQuickActions from '@/components/gotocc/dashboard/DashboardQuickActions.vue'
import DashboardRecentRequests from '@/components/gotocc/dashboard/DashboardRecentRequests.vue'
import DashboardToolbar from '@/components/gotocc/dashboard/DashboardToolbar.vue'
import DashboardUsageChart from '@/components/gotocc/dashboard/DashboardUsageChart.vue'
import SetupGuideCard from '@/components/gotocc/onboarding/SetupGuideCard.vue'
import { dashboardMotionVars } from '@/components/gotocc/dashboard/motion'
import { provideUsageState } from '@/components/gotocc/dashboard/usageState'
import { useUserView } from '@/composables/useUserView'
import { useAnnouncementStore } from '@/stores/announcements'

// GoToCC 用户仪表盘：布局与动效参照 TokenRouter（LGPL-3.0）。
// 第一行是重点（余额、今天、服务质量），第二行起是随时间范围变化的区间数据、趋势、常用模型和热力图，
// 然后是最近请求、公告、快捷入口和联系我们，最后是各平台的用量与额度。
const { t } = useI18n()
const view = useUserView()
const announcementStore = useAnnouncementStore()
const usageState = provideUsageState()

const focusRef = ref<InstanceType<typeof DashboardFocus> | null>(null)
const usageChartRef = ref<InstanceType<typeof DashboardUsageChart> | null>(null)
const heatmapRef = ref<InstanceType<typeof DashboardHeatmap> | null>(null)
const platformRef = ref<InstanceType<typeof DashboardPlatformUsage> | null>(null)
const recentRef = ref<InstanceType<typeof DashboardRecentRequests> | null>(null)
const refreshing = ref(false)
// 入场动画只在首次挂载时播放；简易模式没有平台区，最后一块是最近请求那一行。
const entering = ref(true)

// 刷新整个仪表盘，顶栏余额随账户信息一起更新；平台区在简易模式下不挂载。
const refreshAll = async () => {
  refreshing.value = true
  await Promise.all([
    view.refreshUser(),
    usageState.load(),
    focusRef.value!.reload(),
    heatmapRef.value!.reload(),
    platformRef.value?.reload(),
    recentRef.value!.reload(),
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

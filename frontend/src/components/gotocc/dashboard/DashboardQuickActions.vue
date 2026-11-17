<template>
  <div class="card overflow-hidden">
    <div class="flex min-h-[61px] items-center gap-2 border-b border-gray-100 px-5 py-3 dark:border-dark-700">
      <Icon name="bolt" size="md" class="shrink-0 text-primary-600 dark:text-primary-400" />
      <h2 class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ t('dashboard.quickActions') }}</h2>
    </div>
    <div class="space-y-2 p-4">
      <button
        v-for="action in actions"
        :key="action.path"
        type="button"
        class="group flex w-full items-center gap-3 rounded-xl p-3 text-left transition-colors surface-inset hover:surface-hover"
        @click="router.push(action.path)"
      >
        <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg transition-transform group-hover:scale-105" :class="action.iconClass">
          <Icon :name="action.icon" size="md" />
        </span>
        <span class="min-w-0 flex-1">
          <span class="block truncate text-sm font-medium text-gray-900 dark:text-white">{{ action.title }}</span>
          <span class="block truncate text-xs text-gray-500 dark:text-dark-400">{{ action.description }}</span>
        </span>
        <Icon name="chevronRight" size="sm" class="quick-action-arrow shrink-0 text-gray-400 group-hover:translate-x-0.5 dark:text-dark-500" :class="action.arrowClass" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useResellerCustomer } from '@/composables/useResellerCustomer'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

// 仪表盘快捷入口：入口与显示条件同上游 UserDashboardQuickActions（批量生图按权限显示，站长客户不显示兑换），
// 紧凑样式参照 TokenRouter（LGPL-3.0）：36px 图标块、各自的图标底色，悬停时右侧箭头右移并换成该行的颜色。
const router = useRouter()
const { t } = useI18n()
const resellerCustomer = useResellerCustomer()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

const actions = computed(() => [
  {
    path: '/keys',
    icon: 'key' as const,
    title: t('dashboard.createApiKey'),
    description: t('dashboard.generateNewKey'),
    iconClass: 'bg-primary-100 text-primary-600 dark:bg-primary-500/15 dark:text-primary-300',
    arrowClass: 'group-hover:text-primary-500 dark:group-hover:text-primary-400',
    visible: true,
  },
  {
    path: '/usage',
    icon: 'chart' as const,
    title: t('dashboard.viewUsage'),
    description: t('dashboard.checkDetailedLogs'),
    iconClass: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-500/15 dark:text-emerald-300',
    arrowClass: 'group-hover:text-emerald-500 dark:group-hover:text-emerald-400',
    visible: true,
  },
  {
    path: '/batch-image',
    icon: 'sparkles' as const,
    title: t('dashboard.batchImageAgent'),
    description: t('dashboard.batchImageAgentDesc'),
    iconClass: 'bg-sky-100 text-sky-600 dark:bg-sky-500/15 dark:text-sky-300',
    arrowClass: 'group-hover:text-sky-500 dark:group-hover:text-sky-400',
    visible: canUseBatchImage.value,
  },
  {
    path: '/redeem',
    icon: 'gift' as const,
    title: t('dashboard.redeemCode'),
    description: t('dashboard.addBalanceWithCode'),
    iconClass: 'bg-amber-100 text-amber-600 dark:bg-amber-500/15 dark:text-amber-300',
    arrowClass: 'group-hover:text-amber-500 dark:group-hover:text-amber-400',
    visible: !resellerCustomer.isCustomer.value,
  },
].filter((action) => action.visible))

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>

<style scoped>
.quick-action-arrow {
  transition:
    transform var(--motion-fast) var(--motion-ease),
    color var(--motion-fast) var(--motion-ease);
}
</style>

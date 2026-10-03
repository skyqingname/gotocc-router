<template>
  <div ref="rootRef" class="relative">
    <button
      type="button"
      class="flex h-9 w-9 items-center justify-center rounded-full ring-1 ring-primary-200 transition-shadow hover:ring-2 hover:ring-primary-300 dark:ring-dark-600 dark:hover:ring-dark-400"
      :aria-label="t('common.userMenu')"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span class="gc-avatar h-8 w-8 text-xs">
        <img v-if="avatarUrl" :src="avatarUrl" :alt="displayName" class="h-full w-full object-cover" />
        <template v-else>{{ initials }}</template>
      </span>
    </button>

    <!-- 用户菜单：账户卡、导航、管理员入口、联系方式、退出和主题按分区排列，结构参照 TokenRouter。 -->
    <Transition name="dropdown-fade">
      <div v-if="open" class="dropdown right-0 mt-2 w-72 origin-top-right py-0">
        <div class="menu-section">
          <!-- 账户卡同时是个人资料入口，右侧齿轮提示可进入设置。 -->
          <RouterLink :to="personalPath('/profile')" class="flex items-center gap-3 rounded-lg bg-gray-50 px-2.5 py-2 transition-colors hover:bg-gray-100 dark:bg-dark-900 dark:hover:bg-dark-700" @click="open = false">
            <span class="gc-avatar h-9 w-9 shrink-0 text-sm">
              <img v-if="avatarUrl" :src="avatarUrl" :alt="displayName" class="h-full w-full object-cover" />
              <template v-else>{{ initials }}</template>
            </span>
            <span class="min-w-0 flex-1">
              <span class="block truncate text-sm font-medium text-gray-900 dark:text-white">{{ displayName }}</span>
              <span class="block truncate text-xs text-gray-500 dark:text-dark-400">{{ user.email }}</span>
            </span>
            <Icon name="cog" size="sm" class="shrink-0 text-gray-400 dark:text-dark-400" />
          </RouterLink>

          <!-- 窄屏顶栏不显示余额，放在账户卡下方。 -->
          <div class="flex items-baseline justify-between gap-3 px-2.5 pb-1 pt-2 sm:hidden">
            <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('common.availableBalance') }}</span>
            <span class="text-right">
              <span class="block text-sm font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatMoney(availableBalance) }}</span>
              <span v-if="frozenBalance > 0" class="block text-xs text-amber-600 dark:text-amber-300">
                {{ t('common.frozenBalance') }} {{ formatMoney(frozenBalance) }}
              </span>
            </span>
          </div>
        </div>

        <!-- 导航分组的显示条件与侧栏一致：功能开关、简易模式、管理员只读协助的路径映射。 -->
        <div v-for="group in navGroups" :key="group.key" class="menu-section">
          <RouterLink v-for="item in group.items" :key="item.path" :to="item.to" class="menu-item" @click="open = false">
            <Icon :name="item.icon" size="md" class="shrink-0" />
            {{ item.label }}
          </RouterLink>
        </div>

        <div v-if="view.isAdmin && (appStore.releaseRepository || showOnboarding)" class="menu-section">
          <a
            v-if="appStore.releaseRepository"
            :href="`https://github.com/${appStore.releaseRepository}`"
            target="_blank"
            rel="noopener noreferrer"
            class="menu-item"
            @click="open = false"
          >
            <svg class="h-5 w-5 shrink-0" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path
                fill-rule="evenodd"
                clip-rule="evenodd"
                d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z"
              />
            </svg>
            {{ t('nav.github') }}
          </a>
          <button v-if="showOnboarding" type="button" class="menu-item" @click="replayGuide">
            <Icon name="questionCircle" size="md" class="shrink-0" />
            {{ t('onboarding.restartTour') }}
          </button>
        </div>

        <div v-if="appStore.contactInfo" class="menu-section">
          <div class="flex items-start gap-3 px-2.5 py-2 text-sm">
            <Icon name="chat" size="md" class="shrink-0 text-gray-400 dark:text-dark-400" />
            <span class="min-w-0">
              <span class="block text-xs text-gray-500 dark:text-dark-400">{{ t('common.contactSupport') }}</span>
              <span class="block break-all font-medium text-gray-800 dark:text-dark-100">{{ appStore.contactInfo }}</span>
            </span>
          </div>
        </div>

        <div class="menu-section">
          <button type="button" class="menu-item menu-item-danger" @click="logout">
            <svg class="h-5 w-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15M12 9l-3 3m0 0l3 3m-3-3h12.75" />
            </svg>
            {{ t('nav.logout') }}
          </button>
        </div>

        <!-- 主题三段式切换：浅色 / 深色 / 跟随系统。 -->
        <div class="menu-section">
          <div v-segmented class="segmented grid grid-cols-3 gap-1" role="radiogroup" :aria-label="t('nav.theme')">
            <button
              v-for="option in themeOptions"
              :key="option.mode"
              type="button"
              role="radio"
              :aria-checked="themeMode === option.mode"
              :aria-label="option.label"
              :title="option.label"
              :class="['segmented-item flex items-center justify-center py-1.5', themeMode === option.mode && 'segmented-item-active']"
              @click="setThemeMode(option.mode, $event)"
            >
              <Icon v-if="option.icon" :name="option.icon" size="sm" />
              <svg v-else class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 17.25v1.007a3 3 0 01-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0115 18.257V17.25m6-12V15a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 15V5.25m18 0A2.25 2.25 0 0018.75 3H5.25A2.25 2.25 0 003 5.25m18 0V12a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 12V5.25" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import { useUserView } from '@/composables/useUserView'
import { useTheme, type ThemeMode } from '@/composables/useTheme'
import { adminSupportContext } from '@/utils/adminSupportContext'
import { supportPathForPersonalPath } from '@/utils/adminSupport'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import Icon from '@/components/icons/Icon.vue'
import { vSegmented } from './vSegmented'

const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const onboardingStore = useOnboardingStore()
const view = useUserView()
const { themeMode, setThemeMode } = useTheme()

const open = ref(false)
const rootRef = ref<HTMLElement | null>(null)

// 头部只在已登录时挂载本组件，用户对象始终存在。
const user = computed(() => view.user!)
const avatarUrl = computed(() => user.value.avatar_url || '')
const displayName = computed(() => user.value.username || user.value.email.split('@')[0])
const initials = computed(() => displayName.value.substring(0, 2).toUpperCase())
const availableBalance = computed(() => user.value.balance)
// 没有进行中的异步任务时接口不返回冻结金额。
const frozenBalance = computed(() => user.value.frozen_balance ?? 0)

// 新手引导按钮只给标准模式下的管理员本人，协助模式不显示。
const showOnboarding = computed(() => !authStore.isSimpleMode && !adminSupportContext.value && user.value.role === 'admin')

// 管理员只读协助时，个人页面路径映射到协助路由，与侧栏一致。
function personalPath(path: string): string {
  const id = adminSupportContext.value?.userId
  return id ? supportPathForPersonalPath(id, path) || path : path
}

type MenuIcon = 'home' | 'chart' | 'key' | 'users' | 'creditCard' | 'calendar' | 'document' | 'gift' | 'userPlus' | 'sun' | 'moon'

interface NavEntry {
  path: string
  label: string
  icon: MenuIcon
  visible: boolean
}

const navGroups = computed(() => {
  const settings = appStore.cachedPublicSettings
  const full = !view.isSimpleMode
  const payment = full && isFeatureFlagEnabled(FeatureFlags.payment)
  // 购买入口文案随站点计费模式切换，与侧栏相同。
  const purchaseLabel = {
    recharge_and_subscription: t('nav.buySubscription'),
    recharge_only: t('nav.recharge'),
    subscription_only: t('nav.subscribe'),
  }[resolveSiteBillingMode(settings)]
  const groups: Array<{ key: string; entries: NavEntry[] }> = [
    {
      key: 'workspace',
      entries: [
        { path: '/dashboard', label: t('nav.dashboard'), icon: 'home', visible: true },
        { path: '/usage', label: t('nav.usage'), icon: 'chart', visible: full },
        { path: '/keys', label: t('nav.apiKeys'), icon: 'key', visible: true },
        { path: '/team', label: t('nav.team'), icon: 'users', visible: full && settings?.team_enabled !== false },
      ],
    },
    {
      key: 'billing',
      entries: [
        { path: '/purchase', label: purchaseLabel, icon: 'creditCard', visible: payment },
        { path: '/subscriptions', label: t('nav.mySubscriptions'), icon: 'calendar', visible: full && isFeatureFlagEnabled(FeatureFlags.subscription) },
        { path: '/orders', label: t('nav.myOrders'), icon: 'document', visible: payment },
        { path: '/redeem', label: t('nav.redeem'), icon: 'gift', visible: full },
        { path: '/affiliate', label: t('nav.affiliate'), icon: 'userPlus', visible: full && isFeatureFlagEnabled(FeatureFlags.affiliate) },
      ],
    },
  ]
  return groups
    .map(group => ({
      key: group.key,
      items: group.entries.filter(entry => entry.visible).map(entry => ({ ...entry, to: personalPath(entry.path) })),
    }))
    .filter(group => group.items.length > 0)
})

const themeOptions = computed<Array<{ mode: ThemeMode; icon: MenuIcon | null; label: string }>>(() => [
  { mode: 'light', icon: 'sun', label: t('nav.lightMode') },
  { mode: 'dark', icon: 'moon', label: t('nav.darkMode') },
  { mode: 'system', icon: null, label: t('nav.systemTheme') },
])

function formatMoney(value: number): string {
  return `$${value.toFixed(2)}`
}

function replayGuide() {
  open.value = false
  onboardingStore.replay()
}

async function logout() {
  open.value = false
  await authStore.logout()
  await router.push('/login')
}

function handleClickOutside(event: MouseEvent) {
  if (rootRef.value && !rootRef.value.contains(event.target as Node)) open.value = false
}

onMounted(() => document.addEventListener('click', handleClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', handleClickOutside))
</script>

<style scoped>
/* 头像底色用 logo 的靛 → 紫渐变。 */
.gc-avatar {
  @apply flex items-center justify-center overflow-hidden rounded-full bg-gradient-primary font-semibold text-white;
}
</style>

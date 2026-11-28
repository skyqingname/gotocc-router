<template>
  <aside
    id="app-navigation"
    ref="sidebarRef"
    class="sidebar"
    :class="{ 'sidebar-collapsed': sidebarCollapsed, 'sidebar-mobile-open': mobileVisible }"
    :inert="!isDesktop && !mobileVisible || undefined"
    :aria-hidden="!isDesktop && !mobileVisible || undefined"
    :role="mobileVisible ? 'dialog' : 'complementary'"
    :aria-modal="mobileVisible || undefined"
    :aria-label="t('common.toggleMenu')"
    tabindex="-1"
  >
    <!-- Logo/Brand -->
    <div class="sidebar-header shell-surface" :class="{ 'sidebar-header-collapsed': sidebarCollapsed }">
      <!-- Custom Logo or Default Logo -->
      <router-link
        :to="homePath"
        class="sidebar-logo flex h-9 w-9 items-center justify-center overflow-hidden rounded-lg shadow-glow transition-opacity hover:opacity-80"
        @click="handleMenuItemClick(homePath)"
      >
        <img v-if="settingsLoaded" :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
      </router-link>
      <div class="sidebar-brand" :class="{ 'sidebar-brand-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'" :inert="sidebarCollapsed || undefined">
        <router-link
          :to="homePath"
          class="sidebar-brand-title text-lg font-bold text-gray-900 transition-colors hover:text-primary-600 dark:text-white dark:hover:text-primary-400"
          @click="handleMenuItemClick(homePath)"
        >
          {{ siteName }}
        </router-link>
        <VersionBadge v-if="isAdmin" />
      </div>
    <button v-if="mobileVisible" type="button" class="sidebar-mobile-close btn-ghost btn-icon surface-control" :aria-label="t('common.close')" @click="closeMobile"><Icon name="x" size="md" /></button>
    </div>

    <div class="sidebar-panel shell-surface">
    <!-- Navigation -->
    <nav ref="sidebarNavRef" v-hover-slide="'.sidebar-link'" class="sidebar-nav scrollbar-hide">
      <!-- Admin console: management menu only -->
      <template v-if="isAdmin">
        <!-- Admin Section -->
        <div class="sidebar-section">
          <template v-for="item in adminNavItems" :key="item.path">
            <!-- Collapsible group (has children) -->
            <template v-if="item.children?.length">
              <button
                type="button"
                class="sidebar-link mb-1 w-full surface-control"
                :class="{
                  'sidebar-link-active': isGroupActive(item) && !isGroupExpanded(item),
                  'sidebar-link-collapsed': sidebarCollapsed
                }"
                :data-tip="sidebarCollapsed ? item.label : t('consoleTips.nav.groupToggle')"
                data-tip-side="right"
                @click="handleGroupClick(item)"
              >
                <component :is="item.icon" class="h-5 w-5 flex-shrink-0" />
                <span
                  class="sidebar-label sidebar-label-flex"
                  :class="{ 'sidebar-label-collapsed': sidebarCollapsed }"
                  :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
                >
                  <span class="min-w-0 truncate">{{ item.label }}</span>
                  <ChevronDownIcon
                    class="h-4 w-4 flex-shrink-0 transition-transform duration-200"
                    :class="isGroupExpanded(item) ? 'rotate-180' : ''"
                  />
                </span>
              </button>
              <!-- Children -->
              <div v-if="!sidebarCollapsed && isGroupExpanded(item)" class="mb-1 ml-4 border-l border-gray-200 pl-2 dark:border-dark-600">
                <router-link
                  v-for="child in item.children"
                  :key="child.path"
                  :to="child.path"
                  class="sidebar-link mb-0.5 py-1.5 text-sm"
                  :class="{ 'sidebar-link-active': route.path === child.path }"
                  :data-tip="child.tip"
                  data-tip-side="right"
                  @click="handleMenuItemClick(child.path)"
                >
                  <component :is="child.icon" class="h-4 w-4 flex-shrink-0" />
                  <span>{{ child.label }}</span>
                </router-link>
              </div>
            </template>
            <!-- Normal item (no children) -->
            <router-link
              v-else
              :to="item.path"
              class="sidebar-link mb-1"
              :class="{ 'sidebar-link-active': isActive(item.path), 'sidebar-link-collapsed': sidebarCollapsed }"
              :data-tip="sidebarCollapsed ? item.label : item.tip"
              data-tip-side="right"
              :id="
                item.path === '/admin/accounts'
                  ? 'sidebar-channel-manage'
                  : item.path === '/admin/groups'
                    ? 'sidebar-group-manage'
                    : item.path === '/admin/redeem'
                      ? 'sidebar-wallet'
                      : undefined
              "
              @click="handleMenuItemClick(item.path)"
            >
              <span v-if="item.iconSvg" class="h-5 w-5 flex-shrink-0 sidebar-svg-icon" v-html="sanitizeSvg(item.iconSvg)"></span>
              <component v-else :is="item.icon" class="h-5 w-5 flex-shrink-0" />
              <span class="sidebar-label" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{ item.label }}</span>
            </router-link>
          </template>
        </div>

      </template>

      <!-- Personal / assistance view (also used by administrators in the user portal) -->
      <template v-else>
        <div v-if="isAdminUserPortal" class="mb-2" :class="sidebarCollapsed ? 'px-1' : 'px-2'">
          <AdminSupportUserSelector :collapsed="sidebarCollapsed" />
        </div>
        <!-- 用户端分三段：开始、使用、账户；段标题在侧栏收起时变成一条细分隔线。 -->
        <div v-for="section in userNavSections" :key="section.key" class="sidebar-section user-nav-section">
          <div
            class="sidebar-section-title user-nav-section-title"
            :class="{ 'sidebar-section-title-collapsed': sidebarCollapsed }"
            :aria-hidden="sidebarCollapsed ? 'true' : undefined"
          >
            <span class="sidebar-section-title-text" :class="{ 'sidebar-section-title-text-collapsed': sidebarCollapsed }">{{ section.title }}</span>
          </div>
          <template v-for="item in section.items" :key="item.path">
            <!-- 可展开的组（开始使用）：当前页在组内时自动展开；侧栏收起时点它直接进第一项。 -->
            <template v-if="item.children?.length">
              <button
                type="button"
                class="sidebar-link mb-1 w-full surface-control"
                :class="{
                  'sidebar-link-active': isGroupActive(item) && (sidebarCollapsed || !isGroupExpanded(item)),
                  'sidebar-link-collapsed': sidebarCollapsed
                }"
                :aria-expanded="sidebarCollapsed ? undefined : isGroupExpanded(item)"
                :data-tip="sidebarCollapsed ? item.label : item.tip"
                data-tip-side="right"
                @click="handleGroupClick(item)"
              >
                <component :is="item.icon" class="h-5 w-5 flex-shrink-0" />
                <span
                  class="sidebar-label sidebar-label-flex"
                  :class="{ 'sidebar-label-collapsed': sidebarCollapsed }"
                  :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
                >
                  <span class="min-w-0 truncate">{{ item.label }}</span>
                  <ChevronDownIcon
                    class="h-4 w-4 flex-shrink-0 transition-transform duration-200"
                    :class="isGroupExpanded(item) ? 'rotate-180' : ''"
                  />
                </span>
              </button>
              <div v-if="!sidebarCollapsed && isGroupExpanded(item)" class="user-nav-children mb-1 ml-4 border-l border-gray-200 pl-2 dark:border-dark-600">
                <router-link
                  v-for="child in item.children"
                  :key="child.path"
                  :to="child.path"
                  class="sidebar-link mb-0.5 py-1.5 text-sm"
                  :class="{ 'sidebar-link-active': isActive(child.path) }"
                  :data-tip="child.tip"
                  data-tip-side="right"
                  @click="handleMenuItemClick(child.path)"
                >
                  <component :is="child.icon" class="h-4 w-4 flex-shrink-0" />
                  <span>{{ child.label }}</span>
                </router-link>
              </div>
            </template>
            <component
              v-else
              :is="item.external ? 'a' : RouterLink"
              v-bind="item.external ? { href: item.path, target: '_blank', rel: 'noopener noreferrer' } : { to: item.path }"
              class="sidebar-link mb-1"
              :class="{ 'sidebar-link-active': !item.external && isActive(item.path), 'sidebar-link-collapsed': sidebarCollapsed }"
              :data-tip="sidebarCollapsed ? item.label : item.tip"
              data-tip-side="right"
              :data-tour="item.path === '/keys' ? 'sidebar-my-keys' : item.path === '/usage' ? 'sidebar-usage' : undefined"
              @click="handleMenuItemClick(item.path)"
            >
              <span v-if="item.iconSvg" class="h-5 w-5 flex-shrink-0 sidebar-svg-icon" v-html="sanitizeSvg(item.iconSvg)"></span>
              <component v-else :is="item.icon" class="h-5 w-5 flex-shrink-0" />
              <span class="sidebar-label" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{ item.label }}</span>
            </component>
          </template>
        </div>
      </template>
    </nav>

    <!-- Bottom Section -->
    <div v-hover-slide="'.sidebar-link'" class="sidebar-footer mt-auto border-t border-gray-100 p-3 dark:border-dark-800">
      <!-- The portal switch sits directly above the theme toggle. -->
      <button
        v-if="isAdmin"
        id="sidebar-user-portal"
        type="button"
        @click="enterUserPortal"
        class="sidebar-link mb-2 w-full surface-control"
        :class="{ 'sidebar-link-collapsed': sidebarCollapsed }"
        :data-tip="sidebarCollapsed ? t('nav.userPortal') : t('consoleTips.nav.userPortal')"
        data-tip-side="right"
        :aria-label="t('nav.userPortal')"
      >
        <Icon name="user" size="md" class="flex-shrink-0" />
        <span class="sidebar-label" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{ t('nav.userPortal') }}</span>
      </button>
      <button
        v-else-if="isAdminUserPortal"
        id="sidebar-back-to-admin"
        type="button"
        @click="exitToAdmin"
        class="sidebar-link mb-2 w-full surface-control"
        :class="{ 'sidebar-link-collapsed': sidebarCollapsed }"
        :data-tip="sidebarCollapsed ? t('nav.backToAdmin') : t('consoleTips.nav.backToAdmin')"
        data-tip-side="right"
        :aria-label="t('nav.backToAdmin')"
      >
        <Icon name="arrowLeft" size="md" class="flex-shrink-0" />
        <span class="sidebar-label" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{ t('nav.backToAdmin') }}</span>
      </button>

      <!-- Theme Toggle -->
      <button
        @click="toggleTheme"
        class="sidebar-link theme-toggle mb-2 w-full surface-control"
        :class="{ 'sidebar-link-collapsed': sidebarCollapsed }"
        :data-tip="sidebarCollapsed ? (isDark ? t('nav.lightMode') : t('nav.darkMode')) : t(isDark ? 'consoleTips.nav.lightMode' : 'consoleTips.nav.darkMode')"
        data-tip-side="right"
      >
        <SunIcon v-if="isDark" class="h-5 w-5 flex-shrink-0" />
        <MoonIcon v-else class="h-5 w-5 flex-shrink-0" />
        <span class="sidebar-label" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{
          isDark ? t('nav.lightMode') : t('nav.darkMode')
        }}</span>
      </button>

      <!-- Collapse Button -->
      <button v-if="isDesktop"
        @click="toggleSidebar"
        class="sidebar-link w-full surface-control"
        :class="{ 'sidebar-link-collapsed': sidebarCollapsed }"
        :data-tip="sidebarCollapsed ? t('consoleTips.nav.expand') : t('consoleTips.nav.collapse')"
        data-tip-side="right"
      >
        <ChevronDoubleLeftIcon v-if="!sidebarCollapsed" class="h-5 w-5 flex-shrink-0" />
        <ChevronDoubleRightIcon v-else class="h-5 w-5 flex-shrink-0" />
        <span class="sidebar-label" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{ t('nav.collapse') }}</span>
      </button>
    </div>
    </div>
  </aside>

  <!-- Mobile Overlay -->
  <transition name="fade">
    <div
      v-if="mobileVisible"
      class="sidebar-backdrop fixed inset-0 z-[35] lg:hidden surface-overlay"
      aria-hidden="true"
      @click="closeMobile"
    ></div>
  </transition>
</template>

<script setup lang="ts">
import { computed, h, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useMediaQuery } from '@vueuse/core'
import { useDialogFocus } from '@/composables/useDialogFocus'
import { useI18n } from 'vue-i18n'
import { useResellerAccess } from '@/composables/useResellerAccess'
import { useResellerCustomer } from '@/composables/useResellerCustomer'
import { useTheme } from '@/composables/useTheme'
// 侧栏图标用 GoToCC 的 Lucide 图形，悬停时播放动效。
import {
  BellIcon, CalendarCheckIcon, CanvasIcon, ChannelIcon, ChartIcon, ChevronDoubleLeftIcon, ChevronDoubleRightIcon, CogIcon,
  CompassIcon, CreditCardIcon, DashboardIcon, FolderIcon, GiftIcon, GlobeIcon, KeyIcon, MCPIcon, ModelPlazaIcon, MoonIcon, OrderIcon,
  PluginIcon, PriceTagIcon, RechargeSubscriptionIcon, ResellerIcon, RocketIcon, ServerIcon, ShieldIcon, SignalIcon,
  SunIcon, TaskCenterIcon, TeamIcon, TicketIcon, UserCheckIcon, UserIcon, UsersIcon,
} from '@/components/gotocc/navIcons'
import { useAdminSettingsStore, useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import VersionBadge from '@/components/common/VersionBadge.vue'
import AdminSupportUserSelector from '@/components/layout/AdminSupportUserSelector.vue'
import { sanitizeSvg } from '@/utils/sanitize'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, makeSidebarFlag } from '@/utils/featureFlags'
import Icon from '@/components/icons/Icon.vue'
import { usePortalArea } from '@/composables/usePortalArea'
import { supportPathForPersonalPath } from '@/utils/adminSupport'
import {
  adminSupportContext,
  beginSupportTransition,
  endSupportTransition,
  setAdminSupportContext
} from '@/utils/adminSupportContext'
import { ADMIN_PORTAL_HOME, USER_PORTAL_HOME } from '@/utils/portalArea'
import navigationLinks from '../../../../navigation-links.json'
import { vHoverSlide } from '@/components/gotocc/vHoverSlide'

interface NavItem {
  path: string
  label: string
  icon: unknown
  iconSvg?: string
  external?: boolean
  /** 悬停提示：展开时说明这一页做什么；收起时提示显示名称。 */
  tip?: string
  hideInSimpleMode?: boolean
  children?: NavItem[]
  /**
   * When true, the parent item only toggles the expand/collapse state and
   * does NOT navigate to its `path`. The `path` is purely a stable key.
   */
  expandOnly?: boolean
  /**
   * 可选的功能开关 getter。返回 false 时菜单项被隐藏；返回 undefined/true 时显示。
   * 宽容策略（undefined → 显示）避免 public settings 未加载完成时菜单闪烁消失。
   * Getter 里访问的 reactive 来源（store / composable）会被 computed 自动追踪，
   * 开关切换时菜单自动更新。
   */
  featureFlag?: () => boolean | undefined
}

// applyFeatureFlags 递归过滤掉 featureFlag() === false 的节点（含子节点）。
// 使用 `!== false` 宽容语义：undefined（设置未加载）或 true 都视为显示。
function applyFeatureFlags(items: NavItem[]): NavItem[] {
  const out: NavItem[] = []
  for (const item of items) {
    if (item.featureFlag && item.featureFlag() === false) continue
    if (item.children) {
      const children = applyFeatureFlags(item.children)
      if (item.expandOnly && children.length === 0) continue
      out.push({ ...item, children })
    } else {
      out.push(item)
    }
  }
  return out
}

const { t } = useI18n()

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const resellerAccess = useResellerAccess()
const resellerCustomer = useResellerCustomer()
watch(() => authStore.user?.id, () => { void resellerAccess.load(true).catch(() => { resellerAccess.enabled.value = false }) }, { immediate: true })
const onboardingStore = useOnboardingStore()
const adminSettingsStore = useAdminSettingsStore()

const portalArea = usePortalArea()
const isDesktop = useMediaQuery('(min-width: 1024px)')
const sidebarCollapsed = computed(() => isDesktop.value && appStore.sidebarCollapsed)
const mobileOpen = computed(() => appStore.mobileOpen)
const mobileVisible = computed(() => !isDesktop.value && mobileOpen.value)
const sidebarRef = ref<HTMLElement | null>(null)
useDialogFocus(() => mobileVisible.value, sidebarRef, closeMobile)
watch(isDesktop, desktop => { if (desktop) closeMobile() })
const isAdmin = computed(() => authStore.isAdmin && portalArea.value === 'admin')
const isAdminUserPortal = computed(() => authStore.isAdmin && portalArea.value === 'user')
watch(() => route.fullPath, () => {
  closeMobile()
})
const sidebarNavRef = ref<HTMLElement | null>(null)
const { isDark, toggleTheme } = useTheme()

const homePath = computed(() => (isAdmin.value ? ADMIN_PORTAL_HOME : USER_PORTAL_HOME))

// Per-group expand/collapse overrides. A group with no entry follows the
// automatic behavior (expanded while the active route is one of its children);
// a chevron click records the user's choice, which wins over the automatic
// state so an active group can still be collapsed manually.
const groupExpandOverrides = ref<Map<string, boolean>>(new Map())

// Site settings from appStore (cached, no flicker)
const siteName = computed(() => appStore.siteName)
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

// SVG Icon Components
const ChevronDownIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'm19.5 8.25-7.5 7.5-7.5-7.5'
        })
      ]
    )
}

// Public-settings flags go through the registry in utils/featureFlags.ts,
// which handles the opt-in vs opt-out fallback when settings haven't loaded
// yet. Admin-only flags (not in public settings) stay inline below.
const flagChannelMonitor = makeSidebarFlag(FeatureFlags.channelMonitor)
const flagSubscription = makeSidebarFlag(FeatureFlags.subscription)
const flagModelPlaza = makeSidebarFlag(FeatureFlags.modelPlaza)
const flagAffiliate = makeSidebarFlag(FeatureFlags.affiliate)
const flagRiskControl = makeSidebarFlag(FeatureFlags.riskControl)
const flagGlobalIpAccessControl = makeSidebarFlag(FeatureFlags.globalIpAccessControl)
const flagPluginManagement = makeSidebarFlag(FeatureFlags.pluginManagement)
const flagMCP = makeSidebarFlag(FeatureFlags.mcp)
const flagOpsMonitoring = () => adminSettingsStore.opsMonitoringEnabled
const flagAdminPayment = () => adminSettingsStore.paymentEnabled
const flagTeam = () => appStore.cachedPublicSettings?.team_enabled !== false

interface NavSection {
  key: string
  title: string
  items: NavItem[]
}

// buildSelfNavSections 构造用户端的个人导航，分三段：
// 开始：仪表盘、开始使用（新手指引、签到中心）；
// 使用：模型广场、API 密钥、使用记录、任务中心、服务状态、走进画布；
// 账户：充值与订阅（站长客户是「我的额度」）、站长中心、我的团队、个人资料、MCP、管理员配置的自定义菜单。
function buildSelfNavSections(): NavSection[] {
  return [
    {
      key: 'start',
      title: t('nav.sectionStart'),
      items: [
        { path: '/dashboard', label: t('nav.dashboard'), icon: DashboardIcon, tip: t('consoleTips.nav.dashboard') },
        {
          path: '/getting-started',
          label: t('nav.startGroup'),
          icon: RocketIcon,
          tip: t('consoleTips.nav.startGroup'),
          expandOnly: true,
          children: [
            { path: '/getting-started', label: t('nav.guide'), icon: CompassIcon, tip: t('consoleGuide.start.navDescription') },
            { path: '/checkin', label: t('nav.checkin'), icon: CalendarCheckIcon, tip: t('consoleTips.nav.checkin') },
          ],
        },
      ],
    },
    {
      key: 'use',
      title: t('nav.sectionUse'),
      items: [
        // 带 embedded=1 在后台布局内打开，与页眉入口一致。
        { path: '/model-plaza?embedded=1', label: t('nav.modelPlaza'), icon: ModelPlazaIcon, tip: t('consoleTips.nav.modelPlaza'), featureFlag: flagModelPlaza },
        { path: '/keys', label: t('nav.apiKeys'), icon: KeyIcon, tip: t('keys.description') },
        { path: '/usage', label: t('nav.usage'), icon: ChartIcon, tip: t('usage.description'), hideInSimpleMode: true },
        { path: '/tasks', label: t('gotocc.taskCenter.title'), icon: TaskCenterIcon, tip: t('consoleTips.nav.tasks'), hideInSimpleMode: true },
        { path: '/monitor', label: t('nav.serviceStatus'), icon: SignalIcon, tip: t('consoleTips.nav.serviceStatus') },
        { path: navigationLinks.canvas_url, label: t('nav.enterCanvas'), icon: CanvasIcon, tip: t('consoleTips.nav.canvas'), external: true, hideInSimpleMode: true },
      ],
    },
    {
      key: 'account',
      title: t('nav.sectionAccount'),
      items: [
        resellerCustomer.isCustomer.value
          ? { path: '/credits', label: t('nav.customerCredits'), icon: GiftIcon, tip: t('consoleTips.nav.credits'), hideInSimpleMode: true }
          : { path: '/wallet', label: t('nav.wallet'), icon: RechargeSubscriptionIcon, tip: t('consoleTips.nav.wallet'), hideInSimpleMode: true },
        // 站长中心含发展客户和邀请返利两部分，任一可用就显示。
        // 没有站长资格的人（站长客户、被关闭的站长）在这里只有邀请返利，入口就叫邀请返利。
        ...(resellerAccess.enabled.value || flagAffiliate() ? [{ path: '/reseller', label: resellerAccess.enabled.value ? t('nav.reseller') : t('nav.affiliate'), icon: ResellerIcon, tip: resellerAccess.enabled.value ? t('consoleTips.nav.reseller') : t('affiliate.description'), hideInSimpleMode: true }] : []),
        { path: '/team', label: t('nav.team'), icon: TeamIcon, tip: t('team.description'), hideInSimpleMode: true, featureFlag: flagTeam },
        { path: '/profile', label: t('nav.profile'), icon: UserIcon, tip: t('profile.description') },
        { path: '/settings/mcp', label: t('mcp.title'), icon: MCPIcon, tip: t('mcp.description'), featureFlag: flagMCP },
        ...customMenuItemsForUser.value.map((item): NavItem => ({
          path: `/custom/${item.id}`,
          label: item.label,
          icon: null,
          iconSvg: item.icon_svg,
          tip: t('consoleTips.nav.custom', { name: item.label }),
        })),
      ],
    },
  ]
}

// finalizeNav 合并三重过滤：站长客户可见范围、featureFlag 过滤、简易模式过滤。
function finalizeNav(items: NavItem[]): NavItem[] {
  const visible = applyFeatureFlags(items.filter(item => resellerCustomer.canVisit(item.path)))
  return authStore.isSimpleMode ? visible.filter(item => !item.hideInSimpleMode) : visible
}

// 管理员代看用户时，个人页面换成协助视图的路径；组内子项同样换。
function toSupportPath(id: number, item: NavItem): NavItem {
  if (item.external) return item
  return {
    ...item,
    path: supportPathForPersonalPath(id, item.path) || item.path,
    children: item.children?.map(child => toSupportPath(id, child)),
  }
}

// User navigation sections (regular users and administrators in the user portal).
const userNavSections = computed((): NavSection[] => {
  const id = adminSupportContext.value?.userId
  return buildSelfNavSections()
    .map(section => {
      const items = finalizeNav(section.items)
      return { ...section, items: id ? items.map(item => toSupportPath(id, item)) : items }
    })
    .filter(section => section.items.length > 0)
})

// Custom menu items filtered by visibility
const customMenuItemsForUser = computed(() => {
  const items = appStore.cachedPublicSettings?.custom_menu_items ?? []
  return items
    .filter((item) => item.visibility === 'user')
    .sort((a, b) => a.sort_order - b.sort_order)
})

const customMenuItemsForAdmin = computed(() => {
  return adminSettingsStore.customMenuItems
    .filter((item) => item.visibility === 'admin')
    .sort((a, b) => a.sort_order - b.sort_order)
})

// Admin navigation items
const adminNavItems = computed((): NavItem[] => {
  const baseItems: NavItem[] = [
    { path: '/admin/dashboard', label: t('nav.dashboard'), icon: DashboardIcon, tip: t('admin.dashboard.description') },
    { path: '/admin/ops', label: t('nav.ops'), icon: ChartIcon, tip: t('admin.ops.description'), featureFlag: flagOpsMonitoring },
    { path: '/admin/users', label: t('nav.users'), icon: UsersIcon, tip: t('admin.users.description'), hideInSimpleMode: true },
    { path: '/admin/teams', label: t('nav.teams'), icon: UsersIcon, tip: t('team.adminDescription'), hideInSimpleMode: true, featureFlag: flagTeam },
    { path: '/admin/groups', label: t('nav.groups'), icon: FolderIcon, tip: t('admin.groups.description') },
    {
      path: '/admin/channels',
      label: t('nav.channelManagement'),
      icon: ChannelIcon,
      hideInSimpleMode: true,
      expandOnly: true,
      children: [
        { path: '/admin/channels/pricing', label: t('nav.channelPricing'), icon: PriceTagIcon, tip: t('admin.channels.description') },
        { path: '/admin/channels/monitor', label: t('nav.channelMonitor'), icon: SignalIcon, tip: t('admin.channelMonitor.description'), featureFlag: flagChannelMonitor },
      ],
    },
    // 「仅充值」站点连管理端的「订阅管理」入口也一并收起（路由本身不拦截）。
    { path: '/admin/subscriptions', label: t('nav.subscriptions'), icon: CreditCardIcon, tip: t('admin.subscriptions.description'), hideInSimpleMode: true, featureFlag: flagSubscription },
    { path: '/admin/accounts', label: t('nav.accounts'), icon: GlobeIcon, tip: t('admin.accounts.description') },
    { path: '/admin/plugins', label: t('nav.plugins'), icon: PluginIcon, tip: t('admin.plugins.description'), featureFlag: flagPluginManagement },
    { path: '/admin/announcements', label: t('nav.announcements'), icon: BellIcon, tip: t('admin.announcements.description') },
    { path: '/admin/proxies', label: t('nav.proxies'), icon: ServerIcon, tip: t('admin.proxies.description') },
    {
      path: '/admin/security-audit',
      label: t('nav.securityAudit'),
      icon: ShieldIcon,
      expandOnly: true,
      children: [
        { path: '/admin/risk-control', label: t('nav.contentModeration'), icon: ShieldIcon, tip: t('admin.riskControl.description'), featureFlag: flagRiskControl },
        { path: '/admin/ip-access-control', label: t('nav.ipAccessControl'), icon: ShieldIcon, tip: t('admin.ipAccessControl.description'), featureFlag: flagGlobalIpAccessControl },
      ],
    },
    { path: '/admin/redeem', label: t('nav.redeemCodes'), icon: TicketIcon, tip: t('admin.redeem.description'), hideInSimpleMode: true },
    { path: '/admin/promo-codes', label: t('nav.promoCodes'), icon: GiftIcon, tip: t('admin.promo.description'), hideInSimpleMode: true },
    { path: '/admin/reusable-invitation-codes', label: t('nav.reusableInvitationCodes'), icon: TicketIcon, tip: t('admin.reusableInvitationCodes.description'), hideInSimpleMode: true },
    {
      path: '/admin/affiliates',
      label: t('nav.affiliateManagement'),
      icon: UsersIcon,
      hideInSimpleMode: true,
      expandOnly: true,
      featureFlag: flagAffiliate,
      children: [
        { path: '/admin/agents', label: t('nav.agentApplications'), icon: UserCheckIcon, tip: t('admin.agents.description') },
        { path: '/admin/affiliates/invites', label: t('nav.affiliateInviteRecords'), icon: UsersIcon, tip: t('admin.affiliates.invitesDescription') },
        { path: '/admin/affiliates/rebates', label: t('nav.affiliateRebateRecords'), icon: OrderIcon, tip: t('admin.affiliates.rebatesDescription') },
        { path: '/admin/affiliates/transfers', label: t('nav.affiliateTransferRecords'), icon: CreditCardIcon, tip: t('admin.affiliates.transfersDescription') },
      ],
    },
    {
      path: '/admin/orders',
      label: t('nav.orderManagement'),
      icon: OrderIcon,
      hideInSimpleMode: true,
      expandOnly: true,
      featureFlag: flagAdminPayment,
      children: [
        { path: '/admin/orders/dashboard', label: t('nav.paymentDashboard'), icon: ChartIcon, tip: t('consoleTips.nav.paymentDashboard') },
        { path: '/admin/orders', label: t('nav.orderManagement'), icon: OrderIcon, tip: t('consoleTips.nav.orderManagement') },
        { path: '/admin/orders/plans', label: t('nav.paymentPlans'), icon: CreditCardIcon, tip: t('consoleTips.nav.paymentPlans') },
      ],
    },
    { path: '/admin/usage', label: t('nav.usage'), icon: ChartIcon, tip: t('admin.usage.description') },
    { path: '/admin/audit-logs', label: t('nav.auditLogs'), icon: ShieldIcon, tip: t('admin.audit.description'), hideInSimpleMode: true }
  ]

  const visible = applyFeatureFlags(baseItems)

  // 简易模式只保留可用的后台业务；个人密钥和 MCP 属于用户端。
  if (authStore.isSimpleMode) {
    const filtered = visible.filter(item => !item.hideInSimpleMode)
    filtered.push({ path: '/admin/settings', label: t('nav.settings'), icon: CogIcon, tip: t('admin.settings.description') })
    for (const cm of customMenuItemsForAdmin.value) {
      filtered.push({ path: `/custom/${cm.id}`, label: cm.label, icon: null, iconSvg: cm.icon_svg })
    }
    return filtered
  }

  visible.push({ path: '/admin/settings', label: t('nav.settings'), icon: CogIcon, tip: t('admin.settings.description') })
  for (const cm of customMenuItemsForAdmin.value) {
    visible.push({ path: `/custom/${cm.id}`, label: cm.label, icon: null, iconSvg: cm.icon_svg })
  }
  return visible
})

function toggleSidebar() {
  appStore.toggleSidebar()
}

function closeMobile() {
  appStore.setMobileOpen(false)
}

// The console entry always starts from the administrator's own dashboard, even
// after a previous assistance session. Clearing the target here also drops any
// stale context that could redirect the personal route back to another account.
function enterUserPortal() {
  closeMobile()
  setAdminSupportContext(null)
  void router.push(USER_PORTAL_HOME)
}

async function exitToAdmin() {
  closeMobile()
  beginSupportTransition()
  try {
    await router.push(ADMIN_PORTAL_HOME)
  } finally {
    endSupportTransition()
  }
}

function handleMenuItemClick(itemPath: string) {
  if (mobileOpen.value) {
    closeMobile()
  }

  // Map paths to tour selectors
  const pathToSelector: Record<string, string> = {
    '/admin/groups': '#sidebar-group-manage',
    '/admin/accounts': '#sidebar-channel-manage',
    '/keys': '[data-tour="sidebar-my-keys"]'
  }

  const selector = pathToSelector[itemPath]
  if (selector && onboardingStore.isCurrentStep(selector)) {
    onboardingStore.nextStep(500)
  }
}

// 导航 path 可带查询串（模型广场），选中态只比较路径部分。
function routePathOf(path: string): string {
  return path.split('?')[0]
}

function isActive(path: string): boolean {
  const target = routePathOf(path)
  return route.path === target || route.path.startsWith(target + '/')
}

function isGroupActive(item: NavItem): boolean {
  if (!item.children) return false
  return item.children.some(child => isActive(child.path))
}

function isGroupExpanded(item: NavItem): boolean {
  const override = groupExpandOverrides.value.get(item.path)
  if (override !== undefined) return override
  return isGroupActive(item)
}

function toggleGroup(item: NavItem) {
  groupExpandOverrides.value.set(item.path, !isGroupExpanded(item))
}

/**
 * Click handler for collapsible parent items.
 * - When sidebar is collapsed: do nothing (children are not visible).
 * - When `expandOnly` is true: only toggle expand state.
 * - Otherwise (default, e.g. /admin/orders): navigate to the parent path
 *   (router-link semantics) and ensure the group is expanded.
 */
function handleGroupClick(item: NavItem) {
  if (sidebarCollapsed.value) {
    // 收起时看不到子项：用户端的组直接进第一项，管理端保持原样。
    if (!isAdmin.value) {
      handleMenuItemClick(item.children![0].path)
      void router.push(item.children![0].path)
    }
    return
  }
  if (item.expandOnly) {
    toggleGroup(item)
    return
  }
  // Push to path and ensure expanded
  if (route.path !== item.path) {
    router.push(item.path)
  }
  groupExpandOverrides.value.set(item.path, true)
}

// Fetch admin settings (for feature-gated nav items like Ops).
watch(
  () => authStore.isAdmin,
  (v) => {
    if (v) {
      adminSettingsStore.fetch()
    }
  },
  { immediate: true }
)

onMounted(() => {
  if (authStore.isAdmin) {
    adminSettingsStore.fetch()
  }
  // Restore sidebar scroll position after route change re-mounts the component
  if (appStore.sidebarScrollTop > 0 && sidebarNavRef.value) {
    void nextTick(() => {
      if (sidebarNavRef.value) {
        sidebarNavRef.value.scrollTop = appStore.sidebarScrollTop
      }
    })
  }
})

onBeforeUnmount(() => {
  if (sidebarNavRef.value) {
    appStore.sidebarScrollTop = sidebarNavRef.value.scrollTop
  }
})
</script>

<style scoped>
.sidebar-logo {
  flex: 0 0 2.25rem;
  min-width: 2.25rem;
}

.sidebar-header-collapsed {
  gap: 0;
  padding-left: 1.125rem;
  padding-right: 1.125rem;
}

.sidebar-brand {
  min-width: 0;
  flex: 1 1 auto;
  white-space: nowrap;
  transition:
    max-width 0.22s ease,
    opacity 0.14s ease,
    transform 0.14s ease;
  max-width: 12rem;
}

.sidebar-brand-collapsed {
  max-width: 0;
  overflow: hidden;
  opacity: 0;
  transform: translateX(-4px);
  pointer-events: none;
}

.sidebar-brand-title {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-link-collapsed {
  gap: 0;
  padding-left: 0.875rem;
  padding-right: 0.875rem;
}

.sidebar-section-title {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 1.25rem;
  overflow: hidden;
  white-space: nowrap;
}

.sidebar-section-title-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}

.sidebar-section-title::after {
  content: '';
  position: absolute;
  left: 0.75rem;
  right: 0.75rem;
  top: 50%;
  height: 1px;
  background: rgb(229 231 235);
  opacity: 0;
  transform: translateY(-50%);
  transition: opacity 0.18s ease;
}

.dark .sidebar-section-title::after {
  background: rgb(55 65 81);
}

.sidebar-section-title-text-collapsed {
  opacity: 0;
  transform: translateX(-4px);
}

.sidebar-section-title-collapsed::after {
  opacity: 1;
  transition-delay: 0.08s;
}

/* 用户端段标题：小号字、宽字距；第一段之外上方留出段间距。 */
.user-nav-section + .user-nav-section {
  margin-top: 0.75rem;
}

.user-nav-section-title {
  margin-bottom: 0.25rem;
  padding: 0 0.75rem;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--surface-muted);
}

.sidebar-label {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    max-width 0.2s ease,
    opacity 0.12s ease,
    transform 0.12s ease;
  max-width: 12rem;
}

.sidebar-label-flex {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.sidebar-label-collapsed {
  max-width: 0;
  opacity: 0;
  transform: translateX(-4px);
  pointer-events: none;
}

/* Custom SVG icon in sidebar: constrain size without overriding uploaded SVG colors */
.sidebar-svg-icon {
  color: currentColor;
}

.sidebar-svg-icon :deep(svg) {
  display: block;
  width: 1.25rem;
  height: 1.25rem;
}
</style>

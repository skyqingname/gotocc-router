// Derived from TokenFlux/TokenRouter 38ca040de (v0.7.3) frontend/src/components/layout/search/searchCatalog.ts and recentSearches.ts.
// Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation.
import { computed, type Component, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import {
  BatchImageIcon, BellIcon, CanvasIcon, ChannelIcon, ChartIcon, CogIcon, CreditCardIcon, DashboardIcon, FolderIcon, GiftIcon,
  GlobeIcon, KeyIcon, MCPIcon, ModelPlazaIcon, OrderIcon, OrderListIcon, PluginIcon, PriceTagIcon, RechargeSubscriptionIcon,
  ResellerIcon, ServerIcon, ShieldIcon, SignalIcon, TaskCenterIcon, TeamIcon, TicketIcon, UserCheckIcon, UserIcon, UsersIcon,
} from '@/components/gotocc/navIcons'
import { vendorLabel } from '@/components/gotocc/plaza/vendors'
import { useAsyncImageAccess } from '@/composables/useAsyncImageAccess'
import { useResellerAccess } from '@/composables/useResellerAccess'
import { useResellerCustomer } from '@/composables/useResellerCustomer'
import { useTheme } from '@/composables/useTheme'
import { availableLocales, setLocale } from '@/i18n'
import { resolveRouteMetaKeys } from '@/router/title'
import { useAdminSettingsStore, useAppStore, useAuthStore } from '@/stores'
import { parseAdminSupportTargetId, supportPathForPersonalPath } from '@/utils/adminSupport'
import { adminSupportContext } from '@/utils/adminSupportContext'
import { FeatureFlags, isFeatureFlagEnabled, type FeatureFlagDefinition } from '@/utils/featureFlags'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import { bestMatch, type FieldMatch } from './fuzzyMatch'
import config from '../../../../../global-search.json'
import navigationLinks from '../../../../../navigation-links.json'

// 全局搜索能找到的三类条目：模型广场里的模型、当前账号能进的页面、几个现成的操作。
// 页面目录由路由表的 meta.titleKey 生成标题，哪些页面出现、用哪个图标由下面一份自有规则决定，规则逐条照侧栏 AppSidebar.vue 的显示条件与图标：
// 功能开关、管理员与个人区、简易模式、站长客户不可见的充值类页面、站长中心、创作台权限、自定义菜单，以及管理员只读协助时的路径映射。
// 中英文标题都参与匹配，所以中文界面输入 keys、英文界面输入「密钥」都能找到 API 密钥页。

export type SearchKind = 'model' | 'page' | 'action'
export type SearchIcon = 'document' | 'externalLink' | 'key' | 'creditCard' | 'sun' | 'moon' | 'globe' | 'login'

export interface SearchItem {
  // model:<模型名>、page:<侧栏路径>、action:<名称>；最近访问只存这个键，打开时按当前可见条目重新解析。
  key: string
  kind: SearchKind
  title: string
  subtitle: string
  // 参与匹配的文字，第 0 个是可见标题，其后是各语言标题、模型 ID、路径等。
  fields: string[]
  icon: SearchIcon | null
  // 页面用侧栏同一个动效图标；自定义菜单用管理员上传的 SVG。
  navIcon?: Component
  iconSvg?: string
  model?: string
  run: () => unknown
}

export interface RankedItem {
  item: SearchItem
  match: FieldMatch | null
}

// 一条规则对应侧栏的一个入口。hideInSimpleMode、when 与侧栏 NavItem 的 hideInSimpleMode、featureFlag 同义；
// children 是侧栏里只展开不跳转的分组，分组本身不是页面，展开后的子项才是。
// 画布外链没有路由，用 titleKey 给标题；自定义菜单直接用管理员填写的名称。
interface PageRule {
  path: string
  icon?: Component
  iconSvg?: string
  hideInSimpleMode?: boolean
  when?: () => boolean
  titleKey?: string
  label?: string
  external?: boolean
  children?: PageRule[]
}

const storageKey = (owner: number) => `${config.recent_storage_prefix}${owner}`

export function readRecent(owner: number): string[] {
  return JSON.parse(localStorage.getItem(storageKey(owner)) ?? '[]')
}

export function recordRecent(owner: number, key: string): string[] {
  const next = [key, ...readRecent(owner).filter((value) => value !== key)].slice(0, config.recent_limit)
  localStorage.setItem(storageKey(owner), JSON.stringify(next))
  return next
}

// 命中的条目按分数从高到低；同分时最近访问过的在前，再按标题长短，「gpt-6」排在「gpt-6-luna」前面。
export function rankItems(query: string, items: readonly SearchItem[], recent: readonly string[]): RankedItem[] {
  const recentRank = (key: string) => (recent.includes(key) ? recent.indexOf(key) : recent.length)
  return items
    .flatMap((item) => {
      const match = bestMatch(query, item.fields)
      return match ? [{ item, match }] : []
    })
    .sort((left, right) => right.match.score - left.match.score || recentRank(left.item.key) - recentRank(right.item.key) || left.item.title.length - right.item.title.length)
}

export function useSearchCatalog(plazaGroups: Ref<ModelPlazaGroup[]>) {
  const { t, locale } = useI18n()
  const route = useRoute()
  const router = useRouter()
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const adminSettingsStore = useAdminSettingsStore()
  const resellerCustomer = useResellerCustomer()
  const resellerAccess = useResellerAccess()
  const { canUseAsyncImage } = useAsyncImageAccess()
  const { isDark, toggleTheme } = useTheme()
  const routes = new Map(router.getRoutes().map((record) => [record.path, record]))

  const flag = (definition: FeatureFlagDefinition) => () => isFeatureFlagEnabled(definition)
  const teamEnabled = () => appStore.cachedPublicSettings?.team_enabled !== false

  // 当前语言的文字作标题，所有语言的文字都进匹配字段。
  const texts = (render: (code: string) => string) => {
    const title = render(locale.value)
    return { title, fields: [title, ...availableLocales.map(({ code }) => render(code))] }
  }

  // 同 AppSidebar buildSelfNavItems：用户端主菜单与管理员个人区共用，管理员个人区的仪表盘标为「个人仪表盘」。
  function selfRules(personal: boolean): PageRule[] {
    const custom = (appStore.cachedPublicSettings?.custom_menu_items ?? [])
      .filter((item) => item.visibility === 'user')
      .sort((a, b) => a.sort_order - b.sort_order)
    return [
      { path: '/dashboard', icon: DashboardIcon, titleKey: personal ? 'nav.personalDashboard' : undefined },
      { path: '/model-plaza?embedded=1', icon: ModelPlazaIcon, when: flag(FeatureFlags.modelPlaza) },
      { path: '/keys', icon: KeyIcon },
      { path: '/team', icon: TeamIcon, hideInSimpleMode: true, when: teamEnabled },
      { path: navigationLinks.canvas_url, icon: CanvasIcon, titleKey: 'nav.enterCanvas', external: true, hideInSimpleMode: true },
      { path: '/async-image', icon: BatchImageIcon, hideInSimpleMode: true, when: () => canUseAsyncImage.value },
      { path: '/tasks', icon: TaskCenterIcon, hideInSimpleMode: true },
      { path: '/usage', icon: ChartIcon, hideInSimpleMode: true },
      { path: '/available-channels', icon: ChannelIcon, hideInSimpleMode: true, when: flag(FeatureFlags.availableChannels) },
      { path: '/monitor', icon: SignalIcon, when: flag(FeatureFlags.channelMonitor) },
      { path: '/subscriptions', icon: CreditCardIcon, hideInSimpleMode: true, when: flag(FeatureFlags.subscription) },
      { path: '/purchase', icon: RechargeSubscriptionIcon, hideInSimpleMode: true, when: flag(FeatureFlags.payment) },
      { path: '/orders', icon: OrderListIcon, hideInSimpleMode: true, when: flag(FeatureFlags.payment) },
      { path: '/redeem', icon: GiftIcon, hideInSimpleMode: true },
      { path: '/credits', icon: GiftIcon, hideInSimpleMode: true, when: () => resellerCustomer.isCustomer.value },
      { path: '/affiliate', icon: UsersIcon, hideInSimpleMode: true, when: flag(FeatureFlags.affiliate) },
      { path: '/reseller', icon: ResellerIcon, hideInSimpleMode: true, when: () => resellerAccess.enabled.value },
      { path: '/profile', icon: UserIcon },
      { path: '/settings/mcp', icon: MCPIcon, when: flag(FeatureFlags.mcp) },
      ...custom.map((item) => ({ path: `/custom/${item.id}`, label: item.label, iconSvg: item.icon_svg })),
    ]
  }

  // 同 AppSidebar adminNavItems。
  function adminRules(): PageRule[] {
    return [
      { path: '/admin/dashboard', icon: DashboardIcon },
      { path: '/admin/ops', icon: ChartIcon, when: () => adminSettingsStore.opsMonitoringEnabled },
      { path: '/admin/users', icon: UsersIcon, hideInSimpleMode: true },
      { path: '/admin/teams', icon: UsersIcon, hideInSimpleMode: true, when: teamEnabled },
      { path: '/admin/groups', icon: FolderIcon },
      {
        path: '/admin/channels',
        hideInSimpleMode: true,
        children: [{ path: '/admin/channels/pricing', icon: PriceTagIcon }, { path: '/admin/channels/monitor', icon: SignalIcon, when: flag(FeatureFlags.channelMonitor) }],
      },
      { path: '/admin/subscriptions', icon: CreditCardIcon, hideInSimpleMode: true, when: flag(FeatureFlags.subscription) },
      { path: '/admin/accounts', icon: GlobeIcon },
      { path: '/admin/plugins', icon: PluginIcon, when: flag(FeatureFlags.pluginManagement) },
      { path: '/admin/announcements', icon: BellIcon },
      { path: '/admin/proxies', icon: ServerIcon },
      {
        path: '/admin/security-audit',
        children: [
          { path: '/admin/risk-control', icon: ShieldIcon, when: flag(FeatureFlags.riskControl) },
          { path: '/admin/ip-access-control', icon: ShieldIcon, when: flag(FeatureFlags.globalIpAccessControl) },
        ],
      },
      { path: '/admin/redeem', icon: TicketIcon, hideInSimpleMode: true },
      { path: '/admin/promo-codes', icon: GiftIcon, hideInSimpleMode: true },
      { path: '/admin/reusable-invitation-codes', icon: TicketIcon, hideInSimpleMode: true },
      {
        path: '/admin/affiliates',
        hideInSimpleMode: true,
        when: flag(FeatureFlags.affiliate),
        children: [{ path: '/admin/agents', icon: UserCheckIcon }, { path: '/admin/affiliates/invites', icon: UsersIcon }, { path: '/admin/affiliates/rebates', icon: OrderIcon }, { path: '/admin/affiliates/transfers', icon: CreditCardIcon }],
      },
      {
        path: '/admin/orders',
        hideInSimpleMode: true,
        when: () => adminSettingsStore.paymentEnabled,
        children: [{ path: '/admin/orders/dashboard', icon: ChartIcon }, { path: '/admin/orders', icon: OrderIcon }, { path: '/admin/orders/plans', icon: CreditCardIcon }],
      },
      { path: '/admin/usage', icon: ChartIcon },
      { path: '/admin/audit-logs', icon: ShieldIcon, hideInSimpleMode: true },
    ]
  }

  // 同侧栏 applyFeatureFlags：条件不满足的入口连同子项一起隐藏，分组展开成子项。
  const flatten = (rules: PageRule[]): PageRule[] =>
    rules.flatMap((rule) => (rule.when && !rule.when() ? [] : rule.children ? flatten(rule.children) : [rule]))

  // 同侧栏 finalizeNav：站长客户不能进的页面、简易模式隐藏的入口先去掉，再按条件展开。
  const selfPages = (personal: boolean) =>
    flatten(selfRules(personal).filter((rule) => resellerCustomer.canVisit(rule.path) && !(authStore.isSimpleMode && rule.hideInSimpleMode)))

  // 简易模式下管理员菜单去掉标记的入口，补上 API 密钥与 MCP；系统设置和管理员自定义菜单排在最后。
  function adminPages(): PageRule[] {
    const simple = authStore.isSimpleMode
    const custom = adminSettingsStore.customMenuItems
      .filter((item) => item.visibility === 'admin')
      .sort((a, b) => a.sort_order - b.sort_order)
      .map((item) => ({ path: `/custom/${item.id}`, label: item.label, iconSvg: item.icon_svg }))
    const simpleExtras = simple ? [{ path: '/keys', icon: KeyIcon }, { path: '/settings/mcp', icon: MCPIcon, when: flag(FeatureFlags.mcp) }] : []
    return flatten([...adminRules().filter((rule) => !(simple && rule.hideInSimpleMode)), ...simpleExtras, { path: '/admin/settings', icon: CogIcon }, ...custom])
  }

  // 侧栏的分支：管理员本人看管理菜单加个人区（简易模式没有个人区）；普通用户在未开启后端模式或管理员只读协助时看用户菜单。
  const supportTargetId = computed(() => parseAdminSupportTargetId(route.params.user_id))
  const isAdminSupportMode = computed(() => supportTargetId.value !== null && supportTargetId.value !== authStore.user?.id)
  const visibleRules = computed<PageRule[]>(() => {
    if (authStore.isAdmin && !adminSupportContext.value) return [...adminPages(), ...(authStore.isSimpleMode ? [] : selfPages(true))]
    if (!appStore.backendModeEnabled || isAdminSupportMode.value) return selfPages(false)
    return []
  })

  const pages = computed<SearchItem[]>(() => {
    const billingMode = resolveSiteBillingMode(appStore.cachedPublicSettings)
    const supportUser = adminSupportContext.value?.userId
    return visibleRules.value.map((rule): SearchItem => {
      const to = rule.external || !supportUser ? rule.path : supportPathForPersonalPath(supportUser, rule.path) || rule.path
      const record = routes.get(rule.path.split('?')[0])
      const titleKey = rule.titleKey ?? (record && resolveRouteMetaKeys(record, { billingMode }).titleKey)
      const { title, fields } = titleKey ? texts((code) => t(titleKey, {}, { locale: code })) : { title: rule.label!, fields: [rule.label!] }
      return {
        key: `page:${rule.path}`,
        kind: 'page',
        title,
        subtitle: to,
        fields: [...fields, ...(typeof record?.meta.title === 'string' ? [record.meta.title] : []), rule.path],
        icon: rule.icon || rule.iconSvg ? null : 'document',
        navIcon: rule.icon,
        iconSvg: rule.iconSvg,
        run: () => (rule.external ? window.open(to, '_blank', 'noopener,noreferrer') : router.push(to)),
      }
    })
  })

  // 动作只放站内现成的入口：创建 API 密钥、前往充值与侧栏页面同条件；主题、语言、退出登录与用户菜单相同。
  const actions = computed<SearchItem[]>(() => {
    const visible = new Set(visibleRules.value.map((rule) => rule.path))
    const purchaseKey = resolveRouteMetaKeys(routes.get('/purchase')!, { billingMode: resolveSiteBillingMode(appStore.cachedPublicSettings) }).titleKey!
    const action = (name: string, text: { title: string; fields: string[] }, icon: SearchIcon, run: () => unknown): SearchItem => ({
      key: `action:${name}`,
      kind: 'action',
      title: text.title,
      subtitle: '',
      fields: text.fields,
      icon,
      run,
    })
    return [
      ...(visible.has('/keys') ? [action('create-key', texts((code) => t('dashboard.createApiKey', {}, { locale: code })), 'key', () => router.push('/keys'))] : []),
      ...(visible.has('/purchase')
        ? [action('purchase', texts((code) => t('gotocc.search.actions.goTo', { page: t(purchaseKey, {}, { locale: code }) }, { locale: code })), 'creditCard', () => router.push('/purchase'))]
        : []),
      action(
        'theme',
        texts((code) => t('gotocc.search.actions.switchTheme', { mode: t(isDark.value ? 'nav.lightMode' : 'nav.darkMode', {}, { locale: code }) }, { locale: code })),
        isDark.value ? 'sun' : 'moon',
        () => toggleTheme(),
      ),
      ...availableLocales
        .filter((option) => option.code !== locale.value)
        .map((option) => {
          const text = texts((code) => t('gotocc.search.actions.switchLanguage', { name: option.name }, { locale: code }))
          return action(`locale-${option.code}`, { ...text, fields: [...text.fields, option.code] }, 'globe', () => setLocale(option.code))
        }),
      action('logout', texts((code) => t('nav.logout', {}, { locale: code })), 'login', async () => {
        await authStore.logout()
        await router.push('/login')
      }),
    ]
  })

  // 同一模型可能出现在多个分组里，搜索结果每个模型只出一行；选中后进入模型广场。
  const uniqueModels = computed<PlazaModel[]>(() => {
    const unique = new Map<string, PlazaModel>()
    for (const group of plazaGroups.value) {
      for (const model of group.models) {
        if (!unique.has(model.name)) unique.set(model.name, model)
      }
    }
    return [...unique.values()]
  })

  const modelItem = (model: PlazaModel): SearchItem => {
    const title = model.info?.display_name || model.name
    const vendor = model.info ? vendorLabel(model.info.vendor) : ''
    return {
      key: `model:${model.name}`,
      kind: 'model',
      title,
      subtitle: vendor ? `${model.name} · ${vendor}` : model.name,
      fields: [title, model.name, vendor],
      icon: null,
      model: model.name,
      run: () => router.push('/model-plaza?embedded=1'),
    }
  }

  const models = computed<SearchItem[]>(() => uniqueModels.value.map(modelItem))

  // 没输入时的推荐：按厂商分组，每家按官方上线日期从新到旧，同名展示的变体只留一个；厂商按各自最新一款的上线日期排列。
  // 没有官方上线日期的模型不进推荐，搜索时照常能找到。
  const latestByVendor = computed<SearchItem[][]>(() => {
    const byVendor = new Map<string, PlazaModel[]>()
    for (const model of uniqueModels.value) {
      if (!model.info?.release_date) continue
      byVendor.set(model.info.vendor, [...(byVendor.get(model.info.vendor) ?? []), model])
    }
    const newestFirst = (left: PlazaModel, right: PlazaModel) =>
      right.info!.release_date.localeCompare(left.info!.release_date) || left.name.length - right.name.length
    return [...byVendor.values()]
      .map((list) => {
        const seen = new Set<string>()
        return list.sort(newestFirst).filter((model) => !seen.has(model.info!.display_name) && seen.add(model.info!.display_name))
      })
      .sort((left, right) => newestFirst(left[0], right[0]))
      .map((list) => list.map(modelItem))
  })

  return { models, latestByVendor, pages, actions }
}

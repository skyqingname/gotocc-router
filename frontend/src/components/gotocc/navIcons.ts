import { h, type FunctionalComponent, type VNode } from 'vue'

// 侧栏导航图标。图形来自 Lucide（ISC 许可，https://lucide.dev），悬停动效参照 Lucide Animated（MIT 许可，
// https://github.com/pqoqubbw/icons）及 TokenRouter 的移植，改写为纯 CSS，样式在 styles/gotocc.css 的「侧栏图标动效」一节。
// 需要运动的部件带 class：gc-a 整体或分组运动，gc-d 描线（pathLength=1），gc-dN 表示描线的先后序号。

type Shape = [tag: string, attrs: Record<string, string>, children?: Shape[]]

const render = ([tag, attrs, children]: Shape): VNode => h(tag, attrs, children?.map(render))

const icon = (name: string, shapes: Shape[]): FunctionalComponent => {
  const component: FunctionalComponent = () => h(
    'svg',
    {
      class: `gc-nav-icon gc-i-${name}`,
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '1.75',
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round',
      'aria-hidden': 'true',
    },
    shapes.map(render),
  )
  component.displayName = `GcNavIcon_${name}`
  return component
}

// 描线部件：pathLength 归一为 1，便于用 stroke-dashoffset 从 1 画到 0。
const drawn = (tag: string, attrs: Record<string, string>, order = 0): Shape => [tag, { ...attrs, pathLength: '1', class: `gc-d gc-d${order}` }]

export const DashboardIcon = icon('dashboard', [
  ['rect', { class: 'gc-a gc-a0', width: '7', height: '9', x: '3', y: '3', rx: '1' }],
  ['rect', { class: 'gc-a gc-a1', width: '7', height: '5', x: '14', y: '3', rx: '1' }],
  ['rect', { class: 'gc-a gc-a2', width: '7', height: '9', x: '14', y: '12', rx: '1' }],
  ['rect', { class: 'gc-a gc-a3', width: '7', height: '5', x: '3', y: '16', rx: '1' }],
])

export const KeyIcon = icon('key', [
  ['g', { class: 'gc-a' }, [
    ['path', { d: 'm15.5 7.5 2.3 2.3a1 1 0 0 0 1.4 0l2.1-2.1a1 1 0 0 0 0-1.4L19 4' }],
    ['path', { d: 'm21 2-9.6 9.6' }],
    ['circle', { cx: '7.5', cy: '15.5', r: '5.5' }],
  ]],
])

export const BatchImageIcon = icon('images', [
  ['path', { class: 'gc-back', d: 'M4 8a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2' }],
  ['g', { class: 'gc-a' }, [
    ['path', { d: 'm22 11-1.296-1.296a2.4 2.4 0 0 0-3.408 0L11 16' }],
    ['circle', { cx: '13', cy: '7', r: '1', fill: 'currentColor' }],
    ['rect', { x: '8', y: '2', width: '14', height: '14', rx: '2' }],
  ]],
])

// 画布框与两颗星光沿用侧栏线宽；悬停时星光错峰闪亮。
export const CanvasIcon = icon('canvas', [
  ['g', { class: 'gc-a' }, [
    ['path', { d: 'M13 4H5a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-8' }],
    ['path', { d: 'm3 16 5-5 5 5 3-3 5 5' }],
    ['circle', { cx: '8', cy: '7.5', r: '.75', fill: 'currentColor' }],
  ]],
  ['path', { class: 'gc-spark gc-spark0', d: 'm18.5 1 1.1 3.4L23 5.5l-3.4 1.1-1.1 3.4-1.1-3.4L14 5.5l3.4-1.1Z' }],
  ['path', { class: 'gc-spark gc-spark1', d: 'M2 1v3M.5 2.5h3' }],
])

export const ChartIcon = icon('chart', [
  ['path', { d: 'M3 3v16a2 2 0 0 0 2 2h16' }],
  drawn('path', { d: 'M8 17v-3' }, 0),
  drawn('path', { d: 'M13 17V9' }, 1),
  drawn('path', { d: 'M18 17V5' }, 2),
])

export const GiftIcon = icon('gift', [
  ['path', { d: 'M12 11v10' }],
  ['path', { d: 'M20 11v8a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-8' }],
  ['g', { class: 'gc-a' }, [
    ['path', { d: 'M7.5 7a1 1 0 0 1 0-5A4.8 8 0 0 1 12 7a4.8 8 0 0 1 4.5-5 1 1 0 0 1 0 5' }],
    ['rect', { x: '3', y: '7', width: '18', height: '4', rx: '1' }],
  ]],
])

export const UserIcon = icon('user', [
  ['circle', { cx: '12', cy: '8', r: '5', pathLength: '1', class: 'gc-d gc-d0 gc-a' }],
  drawn('path', { d: 'M20 21a8 8 0 0 0-16 0' }, 1),
])

export const UsersIcon = icon('users', [
  ['path', { d: 'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2' }],
  ['circle', { cx: '9', cy: '7', r: '4' }],
  ['g', { class: 'gc-a' }, [
    ['path', { d: 'M22 21v-2a4 4 0 0 0-3-3.87' }],
    ['path', { d: 'M16 3.13a4 4 0 0 1 0 7.75' }],
  ]],
])

// 团队：前排一人、两侧各一人，按 Lucide 网格与线宽自绘，与代理中心的双人图标区分。
export const TeamIcon = icon('team', [
  ['g', { class: 'gc-a' }, [
    ['circle', { cx: '12', cy: '7.5', r: '3' }],
    ['path', { d: 'M6.5 20v-1a4.5 4.5 0 0 1 4.5-4.5h2a4.5 4.5 0 0 1 4.5 4.5v1' }],
  ]],
  ['g', { class: 'gc-side gc-side0' }, [
    ['circle', { cx: '4.5', cy: '10.5', r: '2' }],
    ['path', { d: 'M1.5 20v-1a3.5 3.5 0 0 1 3.5-3.5' }],
  ]],
  ['g', { class: 'gc-side gc-side1' }, [
    ['circle', { cx: '19.5', cy: '10.5', r: '2' }],
    ['path', { d: 'M22.5 20v-1a3.5 3.5 0 0 0-3.5-3.5' }],
  ]],
])

export const ResellerIcon = icon('store', [
  drawn('path', { d: 'M15 21v-5a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v5' }, 1),
  ['path', { class: 'gc-a', d: 'M17.774 10.31a1.12 1.12 0 0 0-1.549 0 2.5 2.5 0 0 1-3.451 0 1.12 1.12 0 0 0-1.548 0 2.5 2.5 0 0 1-3.452 0 1.12 1.12 0 0 0-1.549 0 2.5 2.5 0 0 1-3.77-3.248l2.889-4.184A2 2 0 0 1 7 2h10a2 2 0 0 1 1.653.873l2.895 4.192a2.5 2.5 0 0 1-3.774 3.244' }],
  ['path', { d: 'M4 10.95V19a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8.05' }],
])

export const ModelPlazaIcon = icon('sparkles', [
  ['path', { class: 'gc-a', d: 'M11.017 2.814a1 1 0 0 1 1.966 0l1.051 5.558a2 2 0 0 0 1.594 1.594l5.558 1.051a1 1 0 0 1 0 1.966l-5.558 1.051a2 2 0 0 0-1.594 1.594l-1.051 5.558a1 1 0 0 1-1.966 0l-1.051-5.558a2 2 0 0 0-1.594-1.594l-5.558-1.051a1 1 0 0 1 0-1.966l5.558-1.051a2 2 0 0 0 1.594-1.594z' }],
  ['path', { class: 'gc-star', d: 'M20 2v4' }],
  ['path', { class: 'gc-star', d: 'M22 4h-4' }],
  ['circle', { class: 'gc-star', cx: '4', cy: '20', r: '2' }],
])

export const FolderIcon = icon('folder', [
  ['g', { class: 'gc-a gc-fallback' }, [
    ['path', { d: 'M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z' }],
  ]],
])

export const ChannelIcon = icon('layers', [
  ['path', { d: 'm12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83Z' }],
  ['path', { class: 'gc-a gc-a0', d: 'm22 12.65-9.17 4.16a2 2 0 0 1-1.66 0L2 12.65' }],
  ['path', { class: 'gc-a gc-a1', d: 'm22 17.65-9.17 4.16a2 2 0 0 1-1.66 0L2 17.65' }],
])

export const CreditCardIcon = icon('credit-card', [
  ['g', { class: 'gc-a' }, [
    ['rect', { width: '20', height: '14', x: '2', y: '5', rx: '2' }],
    ['line', { x1: '2', x2: '22', y1: '10', y2: '10' }],
  ]],
])

export const RechargeSubscriptionIcon = icon('wallet', [
  ['g', { class: 'gc-a' }, [
    ['path', { d: 'M19 7V4a1 1 0 0 0-1-1H5a2 2 0 0 0 0 4h15a1 1 0 0 1 1 1v4h-3a2 2 0 0 0 0 4h3a1 1 0 0 0 1-1v-2a1 1 0 0 0-1-1' }],
    ['path', { d: 'M3 5v14a2 2 0 0 0 2 2h15a1 1 0 0 0 1-1v-4' }],
  ]],
])

export const GlobeIcon = icon('globe', [
  ['circle', { cx: '12', cy: '12', r: '10' }],
  ['path', { class: 'gc-a', d: 'M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20' }],
  ['path', { d: 'M2 12h20' }],
])

export const ServerIcon = icon('server', [
  ['g', { class: 'gc-a gc-a0' }, [
    ['rect', { width: '20', height: '8', x: '2', y: '2', rx: '2', ry: '2' }],
    ['line', { x1: '6', x2: '6.01', y1: '6', y2: '6' }],
  ]],
  ['g', { class: 'gc-a gc-a1' }, [
    ['rect', { width: '20', height: '8', x: '2', y: '14', rx: '2', ry: '2' }],
    ['line', { x1: '6', x2: '6.01', y1: '18', y2: '18' }],
  ]],
])

export const PluginIcon = icon('box', [
  drawn('path', { d: 'M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z' }, 0),
  drawn('path', { d: 'm3.3 7 8.7 5 8.7-5' }, 0),
  drawn('path', { d: 'M12 22V12' }, 0),
])

export const BellIcon = icon('bell', [
  ['g', { class: 'gc-a' }, [
    ['path', { d: 'M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9' }],
    ['path', { d: 'M10.3 21a1.94 1.94 0 0 0 3.4 0' }],
  ]],
])

export const TicketIcon = icon('ticket', [
  ['g', { class: 'gc-a gc-a0' }, [
    ['path', { d: 'M13 5H4a2 2 0 0 0-2 2v2a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h9' }],
    ['path', { d: 'M13 5v2' }],
    ['path', { d: 'M13 11v2' }],
    ['path', { d: 'M13 17v2' }],
  ]],
  ['path', { class: 'gc-a gc-a1', d: 'M13 5h7a2 2 0 0 1 2 2v2a3 3 0 0 0 0 6v2a2 2 0 0 1-2 2h-7' }],
])

export const CogIcon = icon('settings', [
  ['g', { class: 'gc-a' }, [
    ['path', { d: 'M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z' }],
    ['circle', { cx: '12', cy: '12', r: '3' }],
  ]],
])

export const SunIcon = icon('sun', [
  ['circle', { cx: '12', cy: '12', r: '4' }],
  ...['M12 2v2', 'm19.07 4.93-1.41 1.41', 'M20 12h2', 'm17.66 17.66 1.41 1.41', 'M12 20v2', 'm6.34 17.66-1.41 1.41', 'M2 12h2', 'm4.93 4.93 1.41 1.41']
    .map((d, index): Shape => ['path', { class: `gc-ray gc-ray${index}`, d }]),
])

export const MoonIcon = icon('moon', [
  ['g', { class: 'gc-a' }, [['path', { d: 'M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z' }]]],
])

export const ChevronDoubleLeftIcon = icon('chevrons-left', [
  ['g', { class: 'gc-a' }, [['path', { d: 'm11 17-5-5 5-5' }], ['path', { d: 'm18 17-5-5 5-5' }]]],
])

export const ChevronDoubleRightIcon = icon('chevrons-right', [
  ['g', { class: 'gc-a' }, [['path', { d: 'm6 17 5-5-5-5' }], ['path', { d: 'm13 17 5-5-5-5' }]]],
])

export const OrderIcon = icon('receipt', [
  ['path', { d: 'M4 3a1 1 0 0 1 1-1 1.3 1.3 0 0 1 .7.2l.933.6a1.3 1.3 0 0 0 1.4 0l.934-.6a1.3 1.3 0 0 1 1.4 0l.933.6a1.3 1.3 0 0 0 1.4 0l.933-.6a1.3 1.3 0 0 1 1.4 0l.934.6a1.3 1.3 0 0 0 1.4 0l.933-.6A1.3 1.3 0 0 1 19 2a1 1 0 0 1 1 1v18a1 1 0 0 1-1 1 1.3 1.3 0 0 1-.7-.2l-.933-.6a1.3 1.3 0 0 0-1.4 0l-.934.6a1.3 1.3 0 0 1-1.4 0l-.933-.6a1.3 1.3 0 0 0-1.4 0l-.933.6a1.3 1.3 0 0 1-1.4 0l-.934-.6a1.3 1.3 0 0 0-1.4 0l-.933.6a1.3 1.3 0 0 1-.7.2 1 1 0 0 1-1-1z' }],
  drawn('path', { d: 'M16 8h-6a2 2 0 0 0 0 4h4a2 2 0 0 1 0 4H8' }, 0),
  drawn('path', { d: 'M12 17V7' }, 3),
])

export const OrderListIcon = icon('list', [
  ['path', { d: 'M3 5h.01' }],
  ['path', { d: 'M3 12h.01' }],
  ['path', { d: 'M3 19h.01' }],
  drawn('path', { d: 'M8 5h13' }, 0),
  drawn('path', { d: 'M8 12h13' }, 1),
  drawn('path', { d: 'M8 19h13' }, 2),
])

export const SignalIcon = icon('activity', [
  drawn('path', { d: 'M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2' }, 0),
])

export const ShieldIcon = icon('shield-check', [
  ['path', { d: 'M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z' }],
  drawn('path', { d: 'm9 12 2 2 4-4' }, 0),
])

export const UserCheckIcon = icon('user-check', [
  ['path', { d: 'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2' }],
  ['circle', { cx: '9', cy: '7', r: '4' }],
  drawn('path', { d: 'm16 11 2 2 4-4' }, 1),
])

export const PriceTagIcon = icon('tag', [
  ['g', { class: 'gc-a' }, [
    ['path', { d: 'M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42z' }],
    ['circle', { cx: '7.5', cy: '7.5', r: '.5', fill: 'currentColor' }],
  ]],
])

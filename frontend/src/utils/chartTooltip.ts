import { Chart, type Plugin } from 'chart.js'

// 图表的悬停提示与控制台统一：反色小气泡，颜色、圆角、内边距、字号都取 console-glass.css 的
// --tip-* 令牌（按传入的主题取浅色或深色那一套）。另外全局登记一个十字线插件：有坐标轴的
// 图表在悬停数据点时画一横一竖两条虚线，颜色取 --chart-crosshair。

const rootToken = (name: string) => getComputedStyle(document.documentElement).getPropertyValue(name).trim()
const px = (name: string) => Number.parseFloat(rootToken(name))

export function chartTooltipStyle(isDark: boolean) {
  const theme = isDark ? 'dark' : 'light'
  const text = rootToken(`--tip-text-${theme}`)
  const font = { size: px('--tip-font-size') }
  return {
    cornerRadius: px('--tip-radius'),
    caretSize: px('--chart-tip-caret'),
    padding: { top: px('--tip-pad-y'), bottom: px('--tip-pad-y'), left: px('--tip-pad-x'), right: px('--tip-pad-x') },
    backgroundColor: rootToken(`--console-tip-fill-${theme}`),
    titleColor: text,
    bodyColor: text,
    footerColor: text,
    titleFont: { ...font, weight: 600 },
    bodyFont: font,
    footerFont: { ...font, weight: 600 },
    borderColor: rootToken(`--tip-edge-${theme}`),
    borderWidth: 1,
  }
}

const crosshair: Plugin = {
  id: 'gcCrosshair',
  beforeDatasetsDraw(chart) {
    const active = chart.getActiveElements()
    if (!active.length || !chart.scales.x) return
    const { ctx, chartArea } = chart
    const { x, y } = active[0].element
    ctx.save()
    ctx.strokeStyle = getComputedStyle(chart.canvas).getPropertyValue('--chart-crosshair').trim()
    ctx.lineWidth = 1
    ctx.setLineDash([3, 3])
    ctx.beginPath()
    ctx.moveTo(x, chartArea.top)
    ctx.lineTo(x, chartArea.bottom)
    ctx.moveTo(chartArea.left, y)
    ctx.lineTo(chartArea.right, y)
    ctx.stroke()
    ctx.restore()
  },
}

Chart.register(crosshair)

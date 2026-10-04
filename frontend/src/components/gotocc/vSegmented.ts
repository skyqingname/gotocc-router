import type { ObjectDirective } from 'vue'

// 分段控件的滑动底板：按选中项的实际位置设置 .segmented 上的 --segmented-* 变量，
// 样式见 styles/gotocc.css。移植自 TokenRouter（LGPL-3.0）。

interface SegmentedState {
  observer: ResizeObserver
  items: Set<HTMLElement>
}

const states = new WeakMap<HTMLElement, SegmentedState>()
const properties = ['x', 'y', 'width', 'height'] as const

function syncIndicator(element: HTMLElement) {
  const state = states.get(element)
  if (!state) return

  const items = new Set(
    Array.from(element.children).filter(
      (child): child is HTMLElement => child instanceof HTMLElement && child.classList.contains('segmented-item'),
    ),
  )
  for (const item of state.items) {
    if (!items.has(item)) state.observer.unobserve(item)
  }
  for (const item of items) {
    if (!state.items.has(item)) state.observer.observe(item)
  }
  state.items = items

  const active = Array.from(items).find(item => item.classList.contains('segmented-item-active'))
  if (!active || !active.offsetWidth) {
    element.removeAttribute('data-segmented-ready')
    return
  }

  // offset 几何不受面板进场位移影响，坐标相对于轨道内边框。
  const values = [active.offsetLeft, active.offsetTop, active.offsetWidth, active.offsetHeight]
  properties.forEach((property, index) => {
    element.style.setProperty(`--segmented-${property}`, `${values[index]}px`)
  })
  element.setAttribute('data-segmented-ready', '')
}

export const vSegmented: ObjectDirective<HTMLElement> = {
  mounted(element) {
    const observer = new ResizeObserver(() => syncIndicator(element))
    states.set(element, { observer, items: new Set() })
    observer.observe(element)
    syncIndicator(element)
  },
  updated: syncIndicator,
  beforeUnmount(element) {
    states.get(element)?.observer.disconnect()
    states.delete(element)
  },
}

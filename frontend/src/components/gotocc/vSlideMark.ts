import type { ObjectDirective } from 'vue'

// 选项卡的选中指示条：容器底部一条主色细条，切换选项时从旧选项滑到新选项下面。
// 按 [aria-selected="true"] 的子项实际位置写入容器上的 --mark-x / --mark-w；
// 样式见 styles/console-hover.css 的 [data-slide-mark]。

const observers = new WeakMap<HTMLElement, ResizeObserver>()

function sync(container: HTMLElement) {
  const selected = container.querySelector<HTMLElement>(':scope > [aria-selected="true"]')
  if (!selected || !selected.offsetWidth) {
    container.removeAttribute('data-mark-ready')
    return
  }
  container.style.setProperty('--mark-x', `${selected.offsetLeft}px`)
  container.style.setProperty('--mark-w', `${selected.offsetWidth}px`)
  container.setAttribute('data-mark-ready', '')
}

export const vSlideMark: ObjectDirective<HTMLElement> = {
  mounted(container) {
    container.setAttribute('data-slide-mark', '')
    const observer = new ResizeObserver(() => sync(container))
    observer.observe(container)
    observers.set(container, observer)
    sync(container)
  },
  updated: sync,
  beforeUnmount(container) {
    observers.get(container)?.disconnect()
    observers.delete(container)
  },
}

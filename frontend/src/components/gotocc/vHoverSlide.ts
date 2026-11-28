import type { ObjectDirective } from 'vue'

// 悬停底板：容器里放一块底板，指针移到哪一项，底板就平滑滑到哪一项后面，而不是逐项闪烁。
// 绑定值是候选项的选择器（侧栏用 '.sidebar-link'）。指针离开容器时底板淡出；再次进入时
// 先在目标处就位再淡入，不会从上一次的远处滑过来。只在能悬停的设备上启用。
// 样式（颜色、时长、缓动、减少动态效果）见 styles/console-hover.css 的 .hover-slide-plate。

interface SlideState {
  plate: HTMLSpanElement
  current: Element | null
  onOver: (event: PointerEvent) => void
  onLeave: () => void
}

const states = new WeakMap<HTMLElement, SlideState>()
const hoverCapable = window.matchMedia('(hover: hover)')

function place(container: HTMLElement, state: SlideState, item: Element) {
  const box = container.getBoundingClientRect()
  const rect = item.getBoundingClientRect()
  const { plate } = state
  const appearing = state.current === null
  state.current = item
  if (appearing) plate.setAttribute('data-instant', '')
  plate.style.transform = `translate(${rect.left - box.left + container.scrollLeft}px, ${rect.top - box.top + container.scrollTop}px)`
  plate.style.width = `${rect.width}px`
  plate.style.height = `${rect.height}px`
  plate.setAttribute('data-visible', '')
  if (appearing) requestAnimationFrame(() => plate.removeAttribute('data-instant'))
}

export const vHoverSlide: ObjectDirective<HTMLElement, string> = {
  mounted(container, binding) {
    const plate = document.createElement('span')
    plate.className = 'hover-slide-plate'
    plate.setAttribute('aria-hidden', 'true')
    container.prepend(plate)
    container.setAttribute('data-hover-slide', '')
    const state: SlideState = {
      plate,
      current: null,
      onOver(event) {
        if (!hoverCapable.matches || event.pointerType === 'touch' || !(event.target instanceof Element)) return
        const item = event.target.closest(binding.value)
        if (!item || !container.contains(item) || item === state.current) return
        place(container, state, item)
      },
      onLeave() {
        state.current = null
        plate.removeAttribute('data-visible')
      },
    }
    states.set(container, state)
    container.addEventListener('pointerover', state.onOver, { passive: true })
    container.addEventListener('pointerleave', state.onLeave, { passive: true })
  },
  beforeUnmount(container) {
    const state = states.get(container)
    if (!state) return
    container.removeEventListener('pointerover', state.onOver)
    container.removeEventListener('pointerleave', state.onLeave)
    state.plate.remove()
    states.delete(container)
  },
}

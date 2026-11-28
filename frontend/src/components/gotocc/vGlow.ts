import type { ObjectDirective } from 'vue'

// 跟随指针的柔光：挂在控制台根元素上统一委托，指针所在的卡片写入 --glow-x / --glow-y
// （卡片内坐标），光圈的大小、颜色和显隐规则在 styles/console-hover.css。
// 被动监听、rAF 合并为每帧一次；页面隐藏时浏览器不派发指针事件，也不跑 rAF。
// 系统开启「减少动态效果」时不跟随。

const GLOW_TARGET = '[data-glow], .card, .stat-card'

interface GlowState {
  frame: number
  x: number
  y: number
  target: HTMLElement | null
  onMove: (event: PointerEvent) => void
}

const states = new WeakMap<HTMLElement, GlowState>()

function paint(state: GlowState) {
  state.frame = 0
  const card = state.target
  if (!card) return
  const rect = card.getBoundingClientRect()
  card.style.setProperty('--glow-x', `${Math.round(state.x - rect.left)}px`)
  card.style.setProperty('--glow-y', `${Math.round(state.y - rect.top)}px`)
}

export const vGlow: ObjectDirective<HTMLElement> = {
  mounted(element) {
    const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
    const state: GlowState = {
      frame: 0,
      x: 0,
      y: 0,
      target: null,
      onMove(event) {
        if (event.pointerType === 'touch' || reduceMotion.matches) return
        state.x = event.clientX
        state.y = event.clientY
        state.target = event.target instanceof Element ? event.target.closest<HTMLElement>(GLOW_TARGET) : null
        if (state.target && !state.frame) state.frame = requestAnimationFrame(() => paint(state))
      },
    }
    states.set(element, state)
    element.addEventListener('pointermove', state.onMove, { passive: true })
  },
  beforeUnmount(element) {
    const state = states.get(element)
    if (!state) return
    element.removeEventListener('pointermove', state.onMove)
    cancelAnimationFrame(state.frame)
    states.delete(element)
  },
}

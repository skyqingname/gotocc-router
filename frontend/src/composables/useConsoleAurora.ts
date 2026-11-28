import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

// 流动背景的时钟与鼠标牵引（样式见 styles/console-glass.css 的「流动背景」）。
// 时钟从打开控制台算起、页面隐藏时停走：每个页面各自挂一层 AppLayout，换页重挂载后
// 光团按时钟接着原来的位置走，不会跳回起点。鼠标移动经 rAF 合并成每帧一次，只写两个
// -1～1 的比例变量；位移幅度和缓动在令牌里。系统开启「减少动态效果」时不牵引。

const hidden = ref(document.visibilityState === 'hidden')
let runningSince = performance.now()
let elapsed = 0
let listening = false
const lastDrift = { x: 0, y: 0 }

function trackVisibility() {
  if (listening) return
  listening = true
  document.addEventListener('visibilitychange', () => {
    const now = performance.now()
    if (document.visibilityState === 'hidden') {
      if (!hidden.value) elapsed += now - runningSince
      hidden.value = true
    } else {
      runningSince = now
      hidden.value = false
    }
  })
}

function clockSeconds() {
  const running = hidden.value ? 0 : performance.now() - runningSince
  return Number(((elapsed + running) / 1000).toFixed(3))
}

export function useConsoleAurora(drift: Ref<HTMLElement | null>) {
  trackVisibility()
  const clock = clockSeconds()
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
  let frame = 0
  let pointerX = 0
  let pointerY = 0

  function apply() {
    frame = 0
    const element = drift.value
    if (!element) return
    lastDrift.x = (pointerX / window.innerWidth) * 2 - 1
    lastDrift.y = (pointerY / window.innerHeight) * 2 - 1
    element.style.setProperty('--drift-x', lastDrift.x.toFixed(3))
    element.style.setProperty('--drift-y', lastDrift.y.toFixed(3))
  }

  function onPointerMove(event: PointerEvent) {
    pointerX = event.clientX
    pointerY = event.clientY
    if (!frame) frame = requestAnimationFrame(apply)
  }

  onMounted(() => {
    if (reduceMotion.matches) return
    drift.value?.style.setProperty('--drift-x', lastDrift.x.toFixed(3))
    drift.value?.style.setProperty('--drift-y', lastDrift.y.toFixed(3))
    window.addEventListener('pointermove', onPointerMove, { passive: true })
  })

  onBeforeUnmount(() => {
    window.removeEventListener('pointermove', onPointerMove)
    cancelAnimationFrame(frame)
  })

  return { clock, hidden }
}

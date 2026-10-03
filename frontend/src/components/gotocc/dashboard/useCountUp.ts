import { onBeforeUnmount, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { usePreferredReducedMotion } from '@vueuse/core'

// useCountUp 返回一个跟随 source 变化、从当前显示值滚动到新值的数字，四次缓出。
// 系统开启减少动态效果时直接显示新值。移植自 TokenRouter（LGPL-3.0）。
export function useCountUp(source: MaybeRefOrGetter<number>, duration: number) {
  const reducedMotion = usePreferredReducedMotion()
  const display = ref(toValue(source))
  let frame = 0

  const cancel = () => {
    if (frame) cancelAnimationFrame(frame)
    frame = 0
  }

  watch(() => toValue(source), (target) => {
    cancel()
    const from = display.value
    if (from === target || reducedMotion.value === 'reduce') {
      display.value = target
      return
    }
    const startedAt = performance.now()
    const tick = (now: number) => {
      const progress = Math.min((now - startedAt) / duration, 1)
      display.value = from + (target - from) * (1 - Math.pow(1 - progress, 4))
      frame = progress < 1 ? requestAnimationFrame(tick) : 0
    }
    frame = requestAnimationFrame(tick)
  })

  onBeforeUnmount(cancel)

  return display
}

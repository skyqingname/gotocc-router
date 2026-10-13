import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import { useAnimationControls } from 'motion-v'
import type { IconDefinition } from './types'

const INTERACTIVE_SELECTOR = [
  'button',
  'a[href]',
  'summary',
  'label',
  '[data-icon-trigger]',
  '[role="button"]',
  '[role="tab"]',
  '[role="menuitem"]',
  '[role="menuitemcheckbox"]',
  '[role="menuitemradio"]',
  '[role="option"]',
  '[role="checkbox"]',
  '[role="radio"]',
  '[role="switch"]'
].join(', ')

/** 图标动画绑定到最近的控件，悬停控件文字也会触发。 */
export function useIconAnimation(
  svgRef: Ref<SVGSVGElement | null>,
  definition: () => IconDefinition,
  enabled: () => boolean,
  animationActive: () => boolean = () => false
) {
  const controls = useAnimationControls()
  let cleanup: (() => void) | undefined

  onMounted(() => {
    const svg = svgRef.value
    if (!svg) return

    const trigger = svg.closest(INTERACTIVE_SELECTOR) ?? svg
    const labelledControl =
      trigger instanceof HTMLLabelElement ? trigger.control : null
    const media = window.matchMedia('(prefers-reduced-motion: reduce)')
    let hovered = false
    let focused = false
    let active = false
    let generation = 0

    const reset = (immediate: boolean) => {
      controls.stop()
      if (immediate) controls.set(definition().normal)
      else void controls.start(definition().normal)
    }

    const update = () => {
      const suppressed =
        (!enabled() && !animationActive()) ||
        media.matches ||
        trigger.matches(':disabled') ||
        Boolean(
          labelledControl?.matches(':disabled, [aria-disabled="true"]')
        ) ||
        Boolean(
          svg.closest('fieldset:disabled, [aria-disabled="true"], [inert]')
        ) ||
        svg.classList.contains('animate-spin')
      const next =
        !suppressed &&
        (animationActive() || (enabled() && (hovered || focused)))
      if (suppressed && !active) {
        // 复位动画尚未结束时切换系统偏好，也必须立即停在静态图形。
        ++generation
        reset(true)
        return
      }
      if (next === active) return
      active = next
      const current = ++generation
      controls.stop()
      if (!next) {
        reset(suppressed)
        return
      }

      // 序列依靠实际完成事件推进；旧序列不能在离开或卸载后启动下一段。
      void (async () => {
        for (const target of definition().animate) {
          if (current !== generation) return
          await controls.start(target)
        }
      })()
    }

    const enter = (event: Event) => {
      // 触屏不模拟悬停，防止点击后保持装饰动画状态。
      if ('pointerType' in event && event.pointerType === 'touch') return
      hovered = true
      update()
    }
    const leave = () => {
      hovered = false
      update()
    }
    const focus = (event: Event) => {
      // tabindex=-1 仍允许鼠标聚焦 SVG；将装饰图形的焦点交回外层控件。
      // label 内的图标聚焦关联控件，独立图标或不可聚焦容器则清除装饰焦点。
      if (event.target instanceof SVGElement && svg.contains(event.target)) {
        const control = labelledControl ?? trigger
        if (control instanceof HTMLElement) {
          control.focus({ preventScroll: true })
        }
        const activeElement = svg.ownerDocument.activeElement
        if (activeElement instanceof SVGElement && svg.contains(activeElement)) {
          activeElement.blur()
          focused = false
          update()
          return
        }
      }
      focused = true
      update()
    }
    const blur = (event: Event) => {
      if (trigger.contains((event as FocusEvent).relatedTarget as Node | null))
        return
      focused = false
      update()
    }

    trigger.addEventListener('pointerenter', enter)
    trigger.addEventListener('pointerleave', leave)
    trigger.addEventListener('focusin', focus)
    trigger.addEventListener('focusout', blur)
    media.addEventListener('change', update)

    // 异步按钮可能在悬停中变为禁用或加载；同时覆盖父级 fieldset/inert。
    const observer = new MutationObserver(update)
    if (labelledControl) {
      observer.observe(labelledControl, {
        attributes: true,
        attributeFilter: ['disabled', 'aria-disabled']
      })
    }
    let ancestor: Element | null = svg
    while (ancestor) {
      observer.observe(ancestor, {
        attributes: true,
        attributeFilter:
          ancestor === svg
            ? ['disabled', 'aria-disabled', 'inert', 'class']
            : ['disabled', 'aria-disabled', 'inert']
      })
      ancestor = ancestor.parentElement
    }

    const unwatchEnabled = watch(enabled, update)
    // 业务时间轴可独立触发一次动画，仍遵守减少动态效果和禁用状态。
    const unwatchActive = watch(animationActive, update)
    const unwatchDefinition = watch(
      definition,
      () => {
        ++generation
        active = false
        reset(true)
        update()
      },
      { flush: 'post' }
    )

    cleanup = () => {
      ++generation
      controls.stop()
      observer.disconnect()
      unwatchEnabled()
      unwatchActive()
      unwatchDefinition()
      trigger.removeEventListener('pointerenter', enter)
      trigger.removeEventListener('pointerleave', leave)
      trigger.removeEventListener('focusin', focus)
      trigger.removeEventListener('focusout', blur)
      media.removeEventListener('change', update)
    }
    if (animationActive()) update()
  })

  onBeforeUnmount(() => cleanup?.())
  return controls
}

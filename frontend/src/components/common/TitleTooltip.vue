<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'
import consoleGuide from '../../../../console-guide.json'
import { tipFlash } from '@/composables/useTipFlash'

// 控制台统一悬停提示：反色小气泡，挂在 body 上不被裁切，靠边时自动翻到另一侧。
// 内容取自指针所在元素往上第一个命中的：
//   data-tip      专门写给提示的说明（data-tip-side 可指定 top / bottom / left / right，默认 top）；
//   title         原有的原生提示，悬停时转进气泡，原生提示不再出现，离开后原样还回；
//   只有图标的按钮或链接：读它的 aria-label；
//   被省略号截断的文字：显示全文。
// 第一次悬停等 show_delay_ms 才出现；气泡开着或刚关的 skip_delay_ms 内移到相邻触点立即切换。
// 键盘聚焦（:focus-visible）同样出现。只在能悬停的设备上启用。参数见根目录 console-guide.json。

type Side = 'top' | 'bottom' | 'left' | 'right'
interface Candidate { element: Element; text: string }

const config = consoleGuide.tooltip
const OPPOSITE: Record<Side, Side> = { top: 'bottom', bottom: 'top', left: 'right', right: 'left' }

const id = `console-tip-${useId()}`
const bubble = ref<HTMLElement | null>(null)
const text = ref('')
const open = ref(false)
const mode = ref<'delayed' | 'instant'>('delayed')
const sliding = ref(false)
const side = ref<Side>('top')
const position = ref({ x: 0, y: 0, arrow: 0 })

let owner: Element | null = null
let ownerTitle: string | null = null
let addedLabel = ''
let hovered = false
let focused = false
let showTimer = 0
let releaseTimer = 0
let flashTimer = 0
let closedAt = Number.NEGATIVE_INFINITY
let pressed: Element | null = null
let titleObserver: MutationObserver | undefined
let removalObserver: MutationObserver | undefined
let hoverCapable: MediaQueryList

function iconOnly(element: Element) {
  if (!element.matches('button, a[href], [role="button"]')) return false
  return !!element.getAttribute('aria-label')?.trim() && !(element as HTMLElement).innerText.trim()
}

// 先比尺寸再读样式和文字，指针经过大容器时不会去拼整棵子树的文字。
function truncated(element: Element) {
  if (!(element instanceof HTMLElement)) return false
  const clipped = (element.scrollWidth > element.clientWidth + 1 && getComputedStyle(element).textOverflow === 'ellipsis')
    || (element.scrollHeight > element.clientHeight + 1 && getComputedStyle(element).getPropertyValue('-webkit-line-clamp') !== 'none')
  return clipped && !!element.textContent?.trim()
}

function resolve(start: Element): Candidate | null {
  if (start.closest('[role="tooltip"], .help-tooltip-trigger, [data-tip-off]')) return null
  for (let element: Element | null = start; element && element !== document.body; element = element.parentElement) {
    if (element.getAttribute('aria-expanded') === 'true') return null
    const tip = element.getAttribute('data-tip')?.trim()
    if (tip) return { element, text: tip }
    const title = element === owner && ownerTitle !== null ? ownerTitle.trim() : element.getAttribute('title')?.trim()
    if (title) return { element, text: title }
    if (iconOnly(element)) return { element, text: element.getAttribute('aria-label')!.trim() }
    if (truncated(element)) return { element, text: element.textContent!.trim() }
  }
  return null
}

function describe(on: boolean) {
  if (!owner) return
  const ids = new Set((owner.getAttribute('aria-describedby') ?? '').split(/\s+/).filter(Boolean))
  if (on) ids.add(id)
  else ids.delete(id)
  if (ids.size) owner.setAttribute('aria-describedby', [...ids].join(' '))
  else owner.removeAttribute('aria-describedby')
}

function hide() {
  window.clearTimeout(showTimer)
  window.clearTimeout(flashTimer)
  showTimer = 0
  if (open.value) closedAt = performance.now()
  open.value = false
  describe(false)
}

// 点击或按 Esc 收起后，指针还停在这个触点上时不再弹出（同一触点不重复接管），离开后恢复。
const dismiss = hide

function release() {
  window.clearTimeout(releaseTimer)
  hide()
  titleObserver?.disconnect()
  removalObserver?.disconnect()
  if (owner) {
    if (ownerTitle !== null && owner.getAttribute('title') === '') owner.setAttribute('title', ownerTitle)
    if (addedLabel && owner.getAttribute('aria-label') === addedLabel) owner.removeAttribute('aria-label')
  }
  owner = null
  ownerTitle = null
  addedLabel = ''
  hovered = focused = false
}

// 接管一个触点：原生 title 立即清空（否则浏览器会再弹一个），没有可读名称的按钮补上读屏名称。
function claim(candidate: Candidate) {
  owner = candidate.element
  text.value = candidate.text
  const title = owner.getAttribute('title')
  if (title?.trim()) {
    ownerTitle = title
    owner.setAttribute('title', '')
    const named = owner.hasAttribute('aria-label') || owner.hasAttribute('aria-labelledby') || !!(owner as HTMLElement).innerText?.trim()
    if (owner.matches('button, a, input, [role="button"]') && !named) {
      addedLabel = title
      owner.setAttribute('aria-label', title)
    }
    titleObserver?.observe(owner, { attributes: true, attributeFilter: ['title'] })
  }
  removalObserver?.observe(document.body, { childList: true, subtree: true })
}

function present(kind: 'delayed' | 'instant' | 'slide') {
  mode.value = kind === 'delayed' ? 'delayed' : 'instant'
  sliding.value = kind === 'slide'
  open.value = true
  describe(true)
  void nextTick(place)
}

function approach(candidate: Candidate) {
  window.clearTimeout(releaseTimer)
  if (candidate.element === owner) return
  const switching = open.value
  const quick = switching || performance.now() - closedAt < config.skip_delay_ms
  release()
  claim(candidate)
  if (quick) {
    present(switching ? 'slide' : 'instant')
    return
  }
  showTimer = window.setTimeout(() => {
    showTimer = 0
    present('delayed')
  }, config.show_delay_ms)
}

// 离开触点时晚一拍再收：紧接着落到相邻触点就直接滑过去，落到空白处才真正收起。
function leave() {
  window.clearTimeout(releaseTimer)
  releaseTimer = window.setTimeout(release, 0)
}

function place() {
  const element = owner
  const panel = bubble.value
  if (!open.value || !panel) return
  if (!element?.isConnected) return release()
  const anchor = element.getBoundingClientRect()
  if (anchor.bottom <= 0 || anchor.top >= innerHeight || anchor.right <= 0 || anchor.left >= innerWidth) return release()
  const width = panel.offsetWidth
  const height = panel.offsetHeight
  const { gap_px: gap, edge_px: edge, arrow_inset_px: inset } = config
  const room: Record<Side, boolean> = {
    top: anchor.top - gap - height >= edge,
    bottom: anchor.bottom + gap + height <= innerHeight - edge,
    left: anchor.left - gap - width >= edge,
    right: anchor.right + gap + width <= innerWidth - edge,
  }
  const wanted = (element.getAttribute('data-tip-side') ?? 'top') as Side
  const chosen = room[wanted] || !room[OPPOSITE[wanted]] ? wanted : OPPOSITE[wanted]
  const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), max)
  if (chosen === 'top' || chosen === 'bottom') {
    const x = clamp(anchor.left + anchor.width / 2 - width / 2, edge, innerWidth - width - edge)
    const y = chosen === 'top' ? anchor.top - gap - height : anchor.bottom + gap
    position.value = { x, y, arrow: clamp(anchor.left + anchor.width / 2 - x, inset, width - inset) }
  } else {
    const y = clamp(anchor.top + anchor.height / 2 - height / 2, edge, innerHeight - height - edge)
    const x = chosen === 'left' ? anchor.left - gap - width : anchor.right + gap
    position.value = { x, y, arrow: clamp(anchor.top + anchor.height / 2 - y, inset, height - inset) }
  }
  side.value = chosen
}

function onOver(event: PointerEvent) {
  if (event.pointerType === 'touch' || !hoverCapable.matches || !(event.target instanceof Element)) return
  const candidate = resolve(event.target)
  if (!candidate) return
  hovered = true
  approach(candidate)
}
function onOut(event: PointerEvent) {
  if (!owner || (event.relatedTarget instanceof Node && owner.contains(event.relatedTarget))) return
  hovered = false
  if (!focused) leave()
}
function onFocus(event: FocusEvent) {
  if (!hoverCapable.matches || !(event.target instanceof Element) || !event.target.matches(':focus-visible')) return
  const candidate = resolve(event.target)
  if (!candidate) return
  focused = true
  approach(candidate)
}
function onBlur(event: FocusEvent) {
  if (!owner || (event.relatedTarget instanceof Node && owner.contains(event.relatedTarget))) return
  focused = false
  if (!hovered) leave()
}
function onPress(event: PointerEvent) {
  pressed = event.target instanceof Element ? event.target : null
}
function onKey(event: KeyboardEvent) {
  if (event.key === 'Escape') dismiss()
  if (event.key === 'Enter' || event.key === ' ') pressed = document.activeElement
}
function onScroll() {
  sliding.value = false
  place()
}

// 复制成功等反馈：在刚按下的触点上立即显示一句话，过 copied_ms 收起。
watch(tipFlash, (flash) => {
  if (!flash || !pressed?.isConnected || !hoverCapable.matches) return
  const anchor = pressed.closest('button, a, [role="button"], [data-tip], [title]') ?? pressed
  release()
  claim({ element: anchor, text: flash.text })
  hovered = anchor.matches(':hover')
  present('instant')
  flashTimer = window.setTimeout(dismiss, config.copied_ms)
})

onMounted(() => {
  hoverCapable = window.matchMedia('(hover: hover)')
  titleObserver = new MutationObserver(() => {
    if (!owner) return
    const title = owner.getAttribute('title')
    if (title === '') return
    ownerTitle = title
    titleObserver?.disconnect()
    owner.setAttribute('title', '')
    titleObserver?.observe(owner, { attributes: true, attributeFilter: ['title'] })
    if (!title?.trim()) return dismiss()
    text.value = title
    if (addedLabel) {
      addedLabel = title
      owner.setAttribute('aria-label', title)
    }
    void nextTick(place)
  })
  removalObserver = new MutationObserver(() => { if (owner && !owner.isConnected) release() })
  document.addEventListener('pointerover', onOver, true)
  document.addEventListener('pointerout', onOut, true)
  document.addEventListener('pointerdown', onPress, true)
  document.addEventListener('focusin', onFocus, true)
  document.addEventListener('focusout', onBlur, true)
  document.addEventListener('keydown', onKey, true)
  document.addEventListener('click', dismiss, true)
  window.addEventListener('scroll', onScroll, { capture: true, passive: true })
  window.addEventListener('resize', place)
})
onBeforeUnmount(() => {
  release()
  document.removeEventListener('pointerover', onOver, true)
  document.removeEventListener('pointerout', onOut, true)
  document.removeEventListener('pointerdown', onPress, true)
  document.removeEventListener('focusin', onFocus, true)
  document.removeEventListener('focusout', onBlur, true)
  document.removeEventListener('keydown', onKey, true)
  document.removeEventListener('click', dismiss, true)
  window.removeEventListener('scroll', onScroll, true)
  window.removeEventListener('resize', place)
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      :id="id"
      ref="bubble"
      role="tooltip"
      class="console-tip"
      :data-side="side"
      :data-mode="mode"
      :data-sliding="sliding || undefined"
      :style="{ transform: `translate3d(${position.x}px, ${position.y}px, 0)`, '--tip-arrow-at': `${position.arrow}px` }"
    >
      <div class="console-tip-bubble">{{ text }}<span class="console-tip-arrow" aria-hidden="true"></span></div>
    </div>
  </Teleport>
</template>

<style scoped>
.console-tip {
  position: fixed;
  left: 0;
  top: 0;
  z-index: 100000030;
  pointer-events: none;
  width: max-content;
  max-width: min(var(--tip-max-width), calc(100vw - 24px));
}
.console-tip[data-sliding] { transition: transform var(--tip-move) var(--motion-ease-fluid); }
.console-tip-bubble {
  position: relative;
  padding: var(--tip-pad-y) var(--tip-pad-x);
  border: 1px solid var(--tip-edge);
  border-radius: var(--tip-radius);
  background-color: var(--tip-fill);
  box-shadow: var(--tip-shadow);
  color: var(--tip-text);
  font-size: var(--tip-font-size);
  font-weight: 500;
  line-height: var(--tip-line-height);
  white-space: pre-line;
  overflow-wrap: anywhere;
}
.console-tip[data-mode='delayed'] .console-tip-bubble { animation: console-tip-in var(--tip-in) var(--motion-ease-fluid) both; }
.console-tip[data-side='top'] { --tip-from: 0 var(--tip-shift); --tip-origin: var(--tip-arrow-at) 100%; }
.console-tip[data-side='bottom'] { --tip-from: 0 calc(var(--tip-shift) * -1); --tip-origin: var(--tip-arrow-at) 0; }
.console-tip[data-side='left'] { --tip-from: var(--tip-shift) 0; --tip-origin: 100% var(--tip-arrow-at); }
.console-tip[data-side='right'] { --tip-from: calc(var(--tip-shift) * -1) 0; --tip-origin: 0 var(--tip-arrow-at); }
.console-tip-bubble { transform-origin: var(--tip-origin); }
@keyframes console-tip-in {
  from { opacity: 0; translate: var(--tip-from); scale: var(--tip-scale-from); }
}
.console-tip-arrow {
  position: absolute;
  width: var(--tip-arrow-size);
  height: var(--tip-arrow-size);
  background-color: var(--tip-fill);
  border: 1px solid var(--tip-edge);
  rotate: 45deg;
}
.console-tip[data-side='top'] .console-tip-arrow { left: var(--tip-arrow-at); bottom: 0; translate: -50% 50%; border-top: 0; border-left: 0; }
.console-tip[data-side='bottom'] .console-tip-arrow { left: var(--tip-arrow-at); top: 0; translate: -50% -50%; border-bottom: 0; border-right: 0; }
.console-tip[data-side='left'] .console-tip-arrow { top: var(--tip-arrow-at); right: 0; translate: 50% -50%; border-bottom: 0; border-left: 0; }
.console-tip[data-side='right'] .console-tip-arrow { top: var(--tip-arrow-at); left: 0; translate: -50% -50%; border-top: 0; border-right: 0; }
</style>

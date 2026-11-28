<template>
  <Teleport to="body">
    <Transition name="tour-fade">
      <div v-if="open" class="tour" role="dialog" aria-modal="true" :aria-label="t('consoleGuide.tour.label')" data-test="spotlight-tour" @keydown.esc.prevent="finish('skip')">
        <!-- 讲解期间页面不接收点击；聚光框只负责照亮目标。 -->
        <div class="tour-blocker" @click.stop></div>
        <div class="tour-spot" :style="spotStyle"></div>
        <div ref="bubbleRef" class="tour-bubble surface-panel" :style="bubbleStyle" :data-side="placement.side" data-test="tour-bubble">
          <span class="tour-arrow" :style="{ left: `${placement.arrow}px` }"></span>
          <div :key="index" class="gc-reveal">
            <div class="tour-count gc-num">{{ index + 1 }} / {{ steps.length }}</div>
            <h3 class="tour-title">{{ steps[index].title }}</h3>
            <p class="tour-body">{{ steps[index].body }}</p>
          </div>
          <div class="tour-actions">
            <button type="button" class="tour-skip" data-test="tour-skip" @click="finish('skip')">{{ t('consoleGuide.tour.skip') }}</button>
            <button ref="nextRef" type="button" class="btn btn-primary btn-sm" data-test="tour-next" @click="next">
              {{ index === steps.length - 1 ? t('consoleGuide.tour.done') : t('consoleGuide.tour.next') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import consoleGuide from '../../../../../console-guide.json'

// 页面导览：聚光框依次框住目标元素，气泡跟在旁边讲解。位置变化走统一缓动；
// 系统开启「减少动态效果」时直接切换、不做过渡（样式见下）。

export interface TourStep {
  /** 目标元素的选择器；取第一个可见的匹配。 */
  target: string
  title: string
  body: string
  /** 进入这一步前要做的事，例如打开密钥详情。 */
  enter?: () => void
}

const props = defineProps<{ open: boolean; steps: TourStep[] }>()
const emit = defineEmits<{ finish: [result: 'done' | 'skip'] }>()
const { t } = useI18n()
const config = consoleGuide.keys_tour

const index = ref(0)
const rect = ref<DOMRect | null>(null)
const bubbleRef = ref<HTMLElement | null>(null)
const nextRef = ref<HTMLButtonElement | null>(null)
const bubbleHeight = ref(180)
const viewport = ref({ width: window.innerWidth, height: window.innerHeight })
let target: Element | null = null
let token = 0
let frame = 0

const spotStyle = computed(() => {
  const r = rect.value
  if (!r) return { opacity: '0' }
  const pad = config.spotlight_padding
  return { top: `${r.top - pad}px`, left: `${r.left - pad}px`, width: `${r.width + pad * 2}px`, height: `${r.height + pad * 2}px` }
})

const bubbleWidth = computed(() => Math.min(config.bubble_width, viewport.value.width - 32))

const placement = computed(() => {
  const r = rect.value
  const { width: vw, height: vh } = viewport.value
  const w = bubbleWidth.value
  if (!r) return { side: 'none', top: vh / 2 - bubbleHeight.value / 2, left: vw / 2 - w / 2, arrow: w / 2 }
  const gap = config.bubble_gap + config.spotlight_padding
  const below = r.bottom + gap + bubbleHeight.value <= vh - 12
  const above = r.top - gap - bubbleHeight.value >= 12
  const side = below ? 'bottom' : above ? 'top' : 'none'
  const top = side === 'bottom' ? r.bottom + gap : side === 'top' ? r.top - gap - bubbleHeight.value : vh - bubbleHeight.value - 16
  const center = r.left + r.width / 2
  const left = Math.min(Math.max(center - w / 2, 16), vw - w - 16)
  return { side, top, left, arrow: Math.min(Math.max(center - left, 18), w - 18) }
})

const bubbleStyle = computed(() => ({
  top: `${placement.value.top}px`,
  left: `${placement.value.left}px`,
  width: `${bubbleWidth.value}px`
}))

function measure() {
  cancelAnimationFrame(frame)
  frame = requestAnimationFrame(() => {
    viewport.value = { width: window.innerWidth, height: window.innerHeight }
    if (target) rect.value = target.getBoundingClientRect()
    if (bubbleRef.value) bubbleHeight.value = bubbleRef.value.offsetHeight
  })
}

const visibleMatch = (selector: string) =>
  [...document.querySelectorAll(selector)].find(element => element.getBoundingClientRect().height > 0) ?? null

async function show(step: number) {
  const current = ++token
  index.value = step
  const { target: selector, enter } = props.steps[step]
  if (enter) enter()
  // 目标可能随弹窗打开才出现：逐帧等到它可见。
  let element = visibleMatch(selector)
  while (!element) {
    await new Promise(resolve => requestAnimationFrame(resolve))
    if (current !== token) return
    element = visibleMatch(selector)
  }
  target = element
  const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  element.scrollIntoView({ block: 'center', inline: 'nearest', behavior: reduce ? 'auto' : 'smooth' })
  await nextTick()
  measure()
  nextRef.value?.focus({ preventScroll: true })
  await finiteAnimationsSettled()
  if (current === token) measure()
}

function next() {
  if (index.value === props.steps.length - 1) finish('done')
  else void show(index.value + 1)
}

function finish(result: 'done' | 'skip') {
  token++
  target = null
  rect.value = null
  emit('finish', result)
}

function listen(on: boolean) {
  if (on) {
    window.addEventListener('resize', measure)
    window.addEventListener('scroll', measure, true)
  } else {
    window.removeEventListener('resize', measure)
    window.removeEventListener('scroll', measure, true)
  }
}

// 弹窗入场等有限次动画会挪动目标；等它们播完再量一次位置。
const finiteAnimationsSettled = () => Promise.all(document.getAnimations()
  .filter(animation => animation.playState === 'running' && animation.effect?.getComputedTiming().iterations !== Infinity)
  .map(animation => animation.finished.catch(() => undefined)))

watch(() => props.open, (open) => {
  listen(open)
  if (open) void show(0)
})

onBeforeUnmount(() => {
  listen(false)
  cancelAnimationFrame(frame)
  token++
})
</script>

<style scoped>
.tour { position: fixed; inset: 0; z-index: 9000; }
.tour-blocker { position: absolute; inset: 0; }
.tour-spot {
  position: fixed; border-radius: var(--surface-radius-card); pointer-events: none;
  box-shadow: 0 0 0 200vmax rgb(14 12 28 / 58%), 0 0 0 2px rgb(165 180 252 / 85%), 0 0 24px 4px rgb(99 102 241 / 35%);
  transition: top var(--motion-collapse) var(--motion-ease-fluid), left var(--motion-collapse) var(--motion-ease-fluid),
    width var(--motion-collapse) var(--motion-ease-fluid), height var(--motion-collapse) var(--motion-ease-fluid), opacity var(--motion-normal) var(--motion-ease);
}
.tour-bubble {
  position: fixed; display: flex; flex-direction: column; gap: 12px; padding: 16px 16px 12px;
  border-radius: var(--surface-radius-card); color: var(--surface-text);
  transition: top var(--motion-collapse) var(--motion-ease-fluid), left var(--motion-collapse) var(--motion-ease-fluid);
}
.tour-arrow {
  position: absolute; height: 12px; width: 12px; margin-left: -6px; transform: rotate(45deg);
  background: var(--surface-panel); border: 1px solid var(--surface-edge);
  transition: left var(--motion-collapse) var(--motion-ease-fluid);
}
.tour-bubble[data-side='bottom'] .tour-arrow { top: -7px; border-right: 0; border-bottom: 0; }
.tour-bubble[data-side='top'] .tour-arrow { bottom: -7px; border-left: 0; border-top: 0; }
.tour-bubble[data-side='none'] .tour-arrow { display: none; }
.tour-count { font-size: 11px; font-weight: 700; letter-spacing: .04em; color: theme('colors.primary.600'); }
.dark .tour-count { color: theme('colors.primary.300'); }
.tour-title { margin-top: 2px; font-size: 15px; font-weight: 700; }
.tour-body { margin-top: 4px; font-size: 13px; line-height: 1.7; color: var(--surface-muted); }
.tour-actions { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.tour-skip { padding: 4px 2px; font-size: 12px; color: var(--surface-muted); }
.tour-skip:hover { color: var(--surface-text); text-decoration: underline; }
.tour-fade-enter-active, .tour-fade-leave-active { transition: opacity var(--motion-normal) var(--motion-ease); }
.tour-fade-enter-from, .tour-fade-leave-to { opacity: 0; }
@media (prefers-reduced-motion: reduce) {
  .tour-spot, .tour-bubble, .tour-arrow, .tour-fade-enter-active, .tour-fade-leave-active { transition: none; }
}
</style>

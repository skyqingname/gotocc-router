<!-- 兑换成功庆祝：成功图标与彩带覆盖在余额统计上方，保留时长结束后恢复统计。
     移植自 TokenRouter（LGPL-3.0，https://github.com/TokenFlux/TokenRouter）RedeemCelebration.vue。 -->
<template>
  <div class="relative min-w-0">
    <!-- 统计内容保留占位，成功提示收起后恢复显示。 -->
    <div :class="{ invisible: message !== null }" :aria-hidden="message !== null || undefined">
      <slot />
    </div>

    <div class="sr-only" role="status" aria-live="polite" aria-atomic="true">
      <span v-if="announcement" :key="sequence">{{ announcement }}</span>
    </div>

    <Transition name="redeem-celebration" @after-leave="finishFeedback">
      <div
        v-if="visible && message"
        :key="message.sequence"
        :data-sequence="message.sequence"
        class="pointer-events-none absolute inset-0 flex min-w-0 items-center gap-3"
        aria-hidden="true"
      >
        <div
          v-if="confettiVisible"
          :data-sequence="message.sequence"
          class="redeem-confetti pointer-events-none absolute inset-x-0 -top-6 h-28 overflow-hidden"
          @animationend.self="finishConfetti"
        >
          <span
            v-for="particle in particles"
            :key="particle.id"
            class="redeem-confetti-piece absolute h-3 w-1.5 rounded-sm"
            :class="particle.color"
            :style="particle.style"
          />
        </div>

        <div
          class="relative flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400"
          :data-sequence="message.sequence"
          :class="{ 'redeem-success-pop': iconAnimating }"
          @animationend.self="finishIcon"
        >
          <Icon name="check" size="lg" />
        </div>
        <div class="relative min-w-0">
          <p class="truncate text-sm font-medium text-gray-600 dark:text-dark-300">{{ message.title }}</p>
          <p class="truncate text-lg font-semibold tabular-nums text-emerald-700 dark:text-emerald-400">
            {{ message.detail }}
          </p>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { usePreferredReducedMotion } from '@vueuse/core'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  sequence: number
  title: string
  detail: string
}>()

const preference = usePreferredReducedMotion()
const visible = ref(false)
const confettiVisible = ref(false)
const iconAnimating = ref(false)
const announcement = ref('')
const message = ref<{ sequence: number; title: string; detail: string } | null>(null)

// 保留时长只决定提示何时收起，CSS 动画通过完成事件自行清理。
const FEEDBACK_HOLD_MS = 3000
const colors = ['bg-primary-400', 'bg-emerald-400', 'bg-amber-400', 'bg-sky-400', 'bg-rose-400']
// 彩带按固定轨迹从统计区向两侧散开后落下。
const particles = Array.from({ length: 20 }, (_, id) => ({
  id,
  color: colors[id % colors.length],
  style: {
    left: `${4 + (id % 10) * 9.5}%`,
    '--confetti-x': `${(id % 2 === 0 ? -1 : 1) * (14 + (id % 5) * 9)}px`,
    '--confetti-rise': `${-14 - (id % 4) * 9}px`,
    '--confetti-fall': `${40 + (id % 3) * 14}px`,
    '--confetti-turn': `${(id % 2 === 0 ? -1 : 1) * (150 + id * 28)}deg`
  }
}))

// 序号为零表示取消；文案快照保留到退出完成，旧动画不能清除新结果。
watch(() => props.sequence, (sequence, _previous, onCleanup) => {
  visible.value = false
  confettiVisible.value = false
  iconAnimating.value = false
  announcement.value = ''
  if (sequence === 0) return

  message.value = { sequence, title: props.title, detail: props.detail }
  announcement.value = `${props.title} ${props.detail}`
  visible.value = true
  confettiVisible.value = preference.value !== 'reduce'
  iconAnimating.value = preference.value !== 'reduce'

  const timer = window.setTimeout(() => {
    visible.value = false
  }, FEEDBACK_HOLD_MS)
  onCleanup(() => window.clearTimeout(timer))
}, { immediate: true })

// 系统切到减少动态效果时立即取消装饰，切回时不重播本次庆祝。
watch(preference, (value) => {
  if (value !== 'reduce') return
  confettiVisible.value = false
  iconAnimating.value = false
})

function finishConfetti(event: AnimationEvent) {
  const element = event.currentTarget as HTMLElement
  if (Number(element.dataset.sequence) === message.value?.sequence) {
    confettiVisible.value = false
  }
}

function finishIcon(event: AnimationEvent) {
  const element = event.currentTarget as HTMLElement
  if (Number(element.dataset.sequence) === message.value?.sequence) {
    iconAnimating.value = false
  }
}

function finishFeedback(element: Element) {
  if (!visible.value && Number((element as HTMLElement).dataset.sequence) === message.value?.sequence) {
    message.value = null
  }
}
</script>

<style scoped>
.redeem-celebration-enter-active {
  transition: opacity var(--motion-normal) var(--motion-ease);
}

.redeem-celebration-leave-active {
  transition: opacity var(--motion-exit) var(--motion-ease-exit);
}

.redeem-celebration-enter-from,
.redeem-celebration-leave-to {
  opacity: 0;
}

.redeem-success-pop {
  animation: redeem-success-pop var(--motion-layout) var(--motion-ease);
}

.redeem-confetti {
  --celebration-duration: 2000ms;
  animation: redeem-confetti-fade var(--celebration-duration) linear;
}

.redeem-confetti-piece {
  top: 32px;
  animation: redeem-confetti-flight var(--celebration-duration) ease-out both;
}

@keyframes redeem-success-pop {
  0% {
    transform: scale(0.8);
  }
  65% {
    transform: scale(1.08);
  }
  100% {
    transform: scale(1);
  }
}

@keyframes redeem-confetti-fade {
  0%,
  60% {
    opacity: 1;
  }
  100% {
    opacity: 0;
  }
}

@keyframes redeem-confetti-flight {
  0% {
    opacity: 0;
    transform: translateY(8px) rotate(0deg) scale(0.6);
  }
  12% {
    opacity: 0.95;
  }
  35% {
    transform: translate(var(--confetti-x), var(--confetti-rise)) rotate(var(--confetti-turn));
  }
  100% {
    transform: translate(var(--confetti-x), var(--confetti-fall)) rotate(var(--confetti-turn)) scale(0.8);
  }
}

@media (prefers-reduced-motion: reduce) {
  .redeem-success-pop,
  .redeem-confetti,
  .redeem-confetti-piece {
    animation: none;
  }

  .redeem-confetti {
    display: none;
  }
}
</style>

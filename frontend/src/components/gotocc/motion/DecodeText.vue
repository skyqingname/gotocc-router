<template>
  <span class="gc-decode" :aria-label="revealed ? value : maskedValue">
    <span
      v-for="(char, index) in chars"
      :key="index"
      aria-hidden="true"
      :class="{ 'gc-decode-live': !settled[index] }"
    >{{ char }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import keyConsole from '../../../../../key-console.json'

// Secret text that "decodes" into place: every position cycles through random
// characters, then settles left to right. Timing lives in root key-console.json.

const props = withDefaults(defineProps<{
  value: string
  revealed: boolean
  /** Play the arrival from scrambled characters when first shown. */
  arrive?: boolean
}>(), { arrive: false })

const config = keyConsole.key_reveal

const maskedValue = computed(() => {
  const { visible_prefix: prefix, visible_suffix: suffix, mask_char: mask } = config
  const value = props.value
  return value.slice(0, prefix) + mask.repeat(Math.max(value.length - prefix - suffix, 0)) + value.slice(-suffix)
})
const target = computed(() => (props.revealed ? props.value : maskedValue.value))

const chars = ref<string[]>([])
const settled = ref<boolean[]>([])
let frame = 0

const randomChar = () => config.charset[Math.floor(Math.random() * config.charset.length)]
const reducedMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches

function show(text: string) {
  chars.value = [...text]
  settled.value = chars.value.map(() => true)
}

function decode(text: string) {
  cancelAnimationFrame(frame)
  if (reducedMotion()) {
    show(text)
    return
  }
  const goal = [...text]
  const start = performance.now()
  let lastScramble = -Infinity
  const tick = (now: number) => {
    const elapsed = now - start
    const scramble = elapsed - lastScramble >= config.scramble_interval_ms
    if (scramble) lastScramble = elapsed
    const done = goal.map((_, index) => elapsed >= config.settle_start_ms + config.settle_step_ms * index)
    chars.value = goal.map((char, index) => {
      if (done[index]) return char
      return scramble ? randomChar() : chars.value[index] ?? randomChar()
    })
    settled.value = done
    if (done.includes(false)) frame = requestAnimationFrame(tick)
  }
  frame = requestAnimationFrame(tick)
}

watch(target, decode)
if (props.arrive) decode(target.value)
else show(target.value)

onBeforeUnmount(() => cancelAnimationFrame(frame))
</script>

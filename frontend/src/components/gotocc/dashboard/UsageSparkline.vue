<template>
  <div class="relative" aria-hidden="true">
    <!-- 数据变化时按路径重建，从左到右重新描一遍；裁剪放在外层，SVG 上的百分比会按 viewBox 换算 -->
    <div :key="path" class="sparkline-wipe h-full w-full">
      <svg class="h-full w-full overflow-visible" viewBox="0 0 100 32" preserveAspectRatio="none" focusable="false">
        <path
          v-if="path"
          :d="path"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
          vector-effect="non-scaling-stroke"
        />
      </svg>
    </div>
    <!-- 末端圆点用 HTML 绘制，SVG 被拉伸时圆点不会变成椭圆 -->
    <span
      v-if="endPoint"
      :key="`end-${path}`"
      class="sparkline-end absolute h-1.5 w-1.5 rounded-full bg-current"
      :style="{ left: `${endPoint.x}%`, top: `${endPoint.y}%` }"
    ></span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

// 指标卡上的迷你走势线，移植自 TokenRouter（LGPL-3.0）。

const props = defineProps<{
  // 按时间排列的数值，null 表示该时段没有可计算的值，跳过后与前后时段相连。
  values: Array<number | null>
}>()

const WIDTH = 100
const HEIGHT = 32
// 上下各留一点边距，避免线条贴边被裁掉。
const PADDING = 2

// 以 0 为底线按最大值等比缩放，全为 0 时贴底。
const coordinates = computed(() => {
  const numbers = props.values.filter((value): value is number => value !== null)
  const max = Math.max(0, ...numbers)
  const step = props.values.length > 1 ? WIDTH / (props.values.length - 1) : 0
  return props.values
    .map((value, index) => (value === null ? null : {
      x: index * step,
      y: HEIGHT - PADDING - (max > 0 ? value / max : 0) * (HEIGHT - PADDING * 2),
    }))
    .filter((point): point is { x: number; y: number } => point !== null)
})

const path = computed(() => {
  const points = coordinates.value
  if (points.length === 0) return ''
  // 只有一个时段时画成横线。
  if (props.values.length === 1) {
    const y = points[0].y.toFixed(2)
    return `M0 ${y} L${WIDTH} ${y}`
  }
  return `M${points.map((point) => `${point.x.toFixed(2)} ${point.y.toFixed(2)}`).join(' L')}`
})

// 最后一个有值时段的位置，换算成容器百分比。
const endPoint = computed(() => {
  const points = coordinates.value
  if (points.length === 0) return null
  const last = props.values.length === 1 ? { x: WIDTH, y: points[0].y } : points[points.length - 1]
  return { x: (last.x / WIDTH) * 100, y: (last.y / HEIGHT) * 100 }
})
</script>

<style scoped>
/* 描线用裁剪从左向右展开；上下留出余量，不裁掉线帽。 */
.sparkline-wipe {
  animation: sparkline-wipe var(--dash-sparkline-ms) var(--motion-ease) both;
}

.sparkline-end {
  transform: translate(-50%, -50%);
  animation: sparkline-end var(--motion-normal) var(--motion-ease) var(--dash-sparkline-ms) both;
}

@keyframes sparkline-wipe {
  from {
    clip-path: inset(-20% 100% -20% 0);
  }
  to {
    clip-path: inset(-20% 0 -20% 0);
  }
}

@keyframes sparkline-end {
  from {
    opacity: 0;
    transform: translate(-50%, -50%) scale(0);
  }
  to {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .sparkline-wipe,
  .sparkline-end {
    animation: none;
  }
}
</style>

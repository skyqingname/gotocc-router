// 图形与逐元素动效移植自 Lucide Animated；许可与版本见 ../README.md。
// https://github.com/pqoqubbw/icons/blob/072c38b1b04ea738d90a084485ccaad4b890ddca/icons/home.tsx
import { h } from 'vue'
import { motion, type Transition } from 'motion-v'
import type { IconDefinition, IconVariants as Variants } from '../types'

const DEFAULT_TRANSITION: Transition = {
  duration: 0.6,
  opacity: { duration: 0.2 }
}

const PATH_VARIANTS: Variants = {
  normal: {
    pathLength: 1,
    opacity: 1
  },
  animate: {
    opacity: [0, 1],
    pathLength: [0, 1]
  }
}

const icon: IconDefinition = {
  name: 'home',
  normal: 'normal',
  animate: ['animate'],
  render: (controls) =>
    h('g', {}, [
      h(
        'path',
        {
          d: 'M3 10a2 2 0 0 1 .709-1.528l7-5.999a2 2 0 0 1 2.582 0l7 5.999A2 2 0 0 1 21 10v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z'
        },
        []
      ),
      h(
        motion.path,
        {
          animate: controls,
          d: 'M15 21v-8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v8',
          transition: DEFAULT_TRANSITION,
          variants: PATH_VARIANTS,
          initial: 'normal'
        },
        () => []
      )
    ])
}

export default icon

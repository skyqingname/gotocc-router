// 图形与逐元素动效移植自 Lucide Animated；许可与版本见 ../README.md。
// https://github.com/pqoqubbw/icons/blob/072c38b1b04ea738d90a084485ccaad4b890ddca/icons/x.tsx
import { h } from 'vue'
import { motion } from 'motion-v'
import type { IconDefinition, IconVariants as Variants } from '../types'

const PATH_VARIANTS: Variants = {
  normal: {
    opacity: 1,
    pathLength: 1
  },
  animate: {
    opacity: [0, 1],
    pathLength: [0, 1]
  }
}

const icon: IconDefinition = {
  name: 'x',
  normal: 'normal',
  animate: ['animate'],
  render: (controls) =>
    h('g', {}, [
      h(
        motion.path,
        {
          animate: controls,
          d: 'M18 6 6 18',
          variants: PATH_VARIANTS,
          initial: 'normal'
        },
        () => []
      ),
      h(
        motion.path,
        {
          animate: controls,
          d: 'm6 6 12 12',
          transition: { delay: 0.2 },
          variants: PATH_VARIANTS,
          initial: 'normal'
        },
        () => []
      )
    ])
}

export default icon

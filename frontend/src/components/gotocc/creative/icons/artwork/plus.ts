// 图形与逐元素动效移植自 Lucide Animated；许可与版本见 ../README.md。
// https://github.com/pqoqubbw/icons/blob/072c38b1b04ea738d90a084485ccaad4b890ddca/icons/plus.tsx
import { h } from 'vue'
import { motion } from 'motion-v'
import type { IconDefinition } from '../types'

const icon: IconDefinition = {
  name: 'plus',
  normal: 'normal',
  animate: ['animate'],
  render: (controls) =>
    h(
      motion.g,
      {
        animate: controls,
        transition: { type: 'spring', stiffness: 100, damping: 15 },
        variants: {
          normal: {
            rotate: 0
          },
          animate: {
            rotate: 180
          }
        },
        initial: 'normal'
      },
      () => [
        h(
          'path',
          {
            d: 'M5 12h14'
          },
          []
        ),
        h(
          'path',
          {
            d: 'M12 5v14'
          },
          []
        )
      ]
    )
}

export default icon

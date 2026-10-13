// 图形与逐元素动效移植自 Lucide Animated；许可与版本见 ../README.md。
// https://github.com/pqoqubbw/icons/blob/072c38b1b04ea738d90a084485ccaad4b890ddca/icons/refresh-cw.tsx
import { h } from 'vue'
import { motion } from 'motion-v'
import type { IconDefinition } from '../types'

const icon: IconDefinition = {
  name: 'refresh-cw',
  normal: 'normal',
  animate: ['animate'],
  render: (controls) =>
    h(
      motion.g,
      {
        animate: controls,
        transition: { type: 'spring', stiffness: 250, damping: 25 },
        variants: {
          normal: { rotate: '0deg' },
          animate: { rotate: '50deg' }
        },
        initial: 'normal'
      },
      () => [
        h(
          'path',
          {
            d: 'M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8'
          },
          []
        ),
        h(
          'path',
          {
            d: 'M21 3v5h-5'
          },
          []
        ),
        h(
          'path',
          {
            d: 'M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16'
          },
          []
        ),
        h(
          'path',
          {
            d: 'M8 16H3v5'
          },
          []
        )
      ]
    )
}

export default icon

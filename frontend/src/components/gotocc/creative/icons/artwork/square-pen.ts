// 图形与逐元素动效移植自 Lucide Animated；许可与版本见 ../README.md。
// https://github.com/pqoqubbw/icons/blob/072c38b1b04ea738d90a084485ccaad4b890ddca/icons/square-pen.tsx
import { h } from 'vue'
import { motion } from 'motion-v'
import type { IconDefinition, IconVariants as Variants } from '../types'

const PEN_VARIANTS: Variants = {
  normal: {
    rotate: 0,
    x: 0,
    y: 0
  },
  animate: {
    rotate: [-0.5, 0.5, -0.5],
    x: [0, -1, 1.5, 0],
    y: [0, 1.5, -1, 0]
  }
}

const icon: IconDefinition = {
  name: 'square-pen',
  normal: 'normal',
  animate: ['animate'],
  render: (controls) =>
    h(
      'g',
      {
        style: { overflow: 'visible' }
      },
      [
        h(
          'path',
          {
            d: 'M12 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7'
          },
          []
        ),
        h(
          motion.path,
          {
            animate: controls,
            d: 'M18.375 2.625a1 1 0 0 1 3 3l-9.013 9.014a2 2 0 0 1-.853.505l-2.873.84a.5.5 0 0 1-.62-.62l.84-2.873a2 2 0 0 1 .506-.852z',
            variants: PEN_VARIANTS,
            initial: 'normal'
          },
          () => []
        )
      ]
    )
}

export default icon

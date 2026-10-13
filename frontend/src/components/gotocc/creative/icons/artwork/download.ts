// 图形与逐元素动效移植自 Lucide Animated；许可与版本见 ../README.md。
// https://github.com/pqoqubbw/icons/blob/072c38b1b04ea738d90a084485ccaad4b890ddca/icons/download.tsx
import { h } from 'vue'
import { motion } from 'motion-v'
import type { IconDefinition, IconVariants as Variants } from '../types'

const ARROW_VARIANTS: Variants = {
  normal: { y: 0 },
  animate: {
    y: 2,
    transition: {
      type: 'spring',
      stiffness: 200,
      damping: 10,
      mass: 1
    }
  }
}

const icon: IconDefinition = {
  name: 'download',
  normal: 'normal',
  animate: ['animate'],
  render: (controls) =>
    h('g', {}, [
      h(
        'path',
        {
          d: 'M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4'
        },
        []
      ),
      h(
        motion.g,
        {
          animate: controls,
          variants: ARROW_VARIANTS,
          initial: 'normal'
        },
        () => [
          h(
            'polyline',
            {
              points: '7 10 12 15 17 10'
            },
            []
          ),
          h(
            'line',
            {
              x1: '12',
              x2: '12',
              y1: '15',
              y2: '3'
            },
            []
          )
        ]
      )
    ])
}

export default icon

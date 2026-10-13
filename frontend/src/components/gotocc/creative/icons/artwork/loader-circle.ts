// 图形与逐元素动效移植自 Lucide Animated；许可与版本见 ../README.md。
// https://github.com/pqoqubbw/icons/blob/072c38b1b04ea738d90a084485ccaad4b890ddca/icons/loader-circle.tsx
import { h } from 'vue'
import { motion, type Transition } from 'motion-v'
import type { IconDefinition, IconVariants as Variants } from '../types'

const G_VARIANTS: Variants = {
  normal: { rotate: 0 },
  animate: {
    rotate: 360,
    transition: {
      repeat: 0,
      duration: 0.8,
      ease: 'linear'
    }
  }
}

const DEFAULT_TRANSITION: Transition = {
  type: 'spring',
  stiffness: 50,
  damping: 10
}

const icon: IconDefinition = {
  name: 'loader-circle',
  normal: 'normal',
  animate: ['animate'],
  render: (controls) =>
    h('g', {}, [
      h(
        motion.path,
        {
          animate: controls,
          d: 'M21 12a9 9 0 1 1-6.219-8.56',
          style: { transformOrigin: '12px 12px' },
          transition: DEFAULT_TRANSITION,
          variants: G_VARIANTS,
          initial: 'normal'
        },
        () => []
      )
    ])
}

export default icon

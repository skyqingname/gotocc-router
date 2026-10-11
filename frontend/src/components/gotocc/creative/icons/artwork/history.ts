// 图形与逐元素动效移植自 Lucide Animated；许可与版本见 ../README.md。
// https://github.com/pqoqubbw/icons/blob/072c38b1b04ea738d90a084485ccaad4b890ddca/icons/history.tsx
import { h } from 'vue'
import { motion, type Transition } from 'motion-v'
import type { IconDefinition, IconVariants as Variants } from '../types'

const ARROW_TRANSITION: Transition = {
  type: 'spring',
  stiffness: 250,
  damping: 25
}

const ARROW_VARIANTS: Variants = {
  normal: {
    rotate: '0deg'
  },
  animate: {
    rotate: '-50deg'
  }
}

const HAND_TRANSITION: Transition = {
  duration: 0.6,
  ease: [0.4, 0, 0.2, 1]
}

const HAND_VARIANTS: Variants = {
  normal: {
    rotate: 0,
    originX: '0%',
    originY: '100%'
  },
  animate: {
    rotate: -360,
    originX: '0%',
    originY: '100%'
  }
}

const MINUTE_HAND_TRANSITION: Transition = {
  duration: 0.5,
  ease: 'easeInOut'
}

const MINUTE_HAND_VARIANTS: Variants = {
  normal: {
    rotate: 0,
    originX: '0%',
    originY: '0%'
  },
  animate: {
    rotate: -45,
    originX: '0%',
    originY: '0%'
  }
}

const icon: IconDefinition = {
  name: 'history',
  normal: 'normal',
  animate: ['animate'],
  render: (controls) =>
    h('g', {}, [
      h(
        motion.g,
        {
          animate: controls,
          transition: ARROW_TRANSITION,
          variants: ARROW_VARIANTS,
          initial: 'normal'
        },
        () => [
          h(
            'path',
            {
              d: 'M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8'
            },
            []
          ),
          h(
            'path',
            {
              d: 'M3 3v5h5'
            },
            []
          )
        ]
      ),
      h(
        motion.line,
        {
          animate: controls,
          initial: 'normal',
          transition: HAND_TRANSITION,
          variants: HAND_VARIANTS,
          x1: '12',
          x2: '12',
          y1: '12',
          y2: '7'
        },
        () => []
      ),
      h(
        motion.line,
        {
          animate: controls,
          initial: 'normal',
          transition: MINUTE_HAND_TRANSITION,
          variants: MINUTE_HAND_VARIANTS,
          x1: '12',
          x2: '16',
          y1: '12',
          y2: '14'
        },
        () => []
      )
    ])
}

export default icon

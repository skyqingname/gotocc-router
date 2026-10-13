// 图形与逐元素动效移植自 Lucide Animated；许可与版本见 ../README.md。
// https://github.com/pqoqubbw/icons/blob/072c38b1b04ea738d90a084485ccaad4b890ddca/icons/sparkles.tsx
import { h } from 'vue'
import { motion } from 'motion-v'
import type { IconDefinition, IconVariants as Variants } from '../types'

const SPARKLE_VARIANTS: Variants = {
  normal: {
    y: 0,
    fill: 'none'
  },
  animate: {
    y: [0, -1, 0, 0],
    fill: 'currentColor',
    transition: {
      duration: 1,
      bounce: 0.3
    }
  }
}

const STAR_VARIANTS: Variants = {
  normal: {
    opacity: 1,
    x: 0,
    y: 0
  },
  animate: () => ({
    opacity: [0, 1, 0, 0, 0, 0, 1],
    transition: {
      duration: 2,
      delay: 1,
      type: 'tween',
      stiffness: 70,
      damping: 10,
      mass: 0.4
    }
  })
}

const icon: IconDefinition = {
  name: 'sparkles',
  normal: 'normal',
  animate: ['animate'],
  render: (controls) =>
    h('g', {}, [
      h(
        motion.path,
        {
          animate: controls,
          d: 'M9.937 15.5A2 2 0 0 0 8.5 14.063l-6.135-1.582a.5.5 0 0 1 0-.962L8.5 9.936A2 2 0 0 0 9.937 8.5l1.582-6.135a.5.5 0 0 1 .963 0L14.063 8.5A2 2 0 0 0 15.5 9.937l6.135 1.581a.5.5 0 0 1 0 .964L15.5 14.063a2 2 0 0 0-1.437 1.437l-1.582 6.135a.5.5 0 0 1-.963 0z',
          variants: SPARKLE_VARIANTS,
          initial: 'normal'
        },
        () => []
      ),
      h(
        motion.path,
        {
          animate: controls,
          d: 'M20 3v4',
          variants: STAR_VARIANTS,
          initial: 'normal'
        },
        () => []
      ),
      h(
        motion.path,
        {
          animate: controls,
          d: 'M22 5h-4',
          variants: STAR_VARIANTS,
          initial: 'normal'
        },
        () => []
      ),
      h(
        motion.path,
        {
          animate: controls,
          d: 'M4 17v2',
          variants: STAR_VARIANTS,
          initial: 'normal'
        },
        () => []
      ),
      h(
        motion.path,
        {
          animate: controls,
          d: 'M5 18H3',
          variants: STAR_VARIANTS,
          initial: 'normal'
        },
        () => []
      )
    ])
}

export default icon

// 图形与逐元素动效移植自 Lucide Animated；许可与版本见 ../README.md。
// https://github.com/pqoqubbw/icons/blob/072c38b1b04ea738d90a084485ccaad4b890ddca/icons/arrow-up.tsx
import { h } from 'vue'
import { motion } from 'motion-v'
import type { IconDefinition, IconVariants as Variants } from '../types'

const PATH_VARIANTS: Variants = {
  normal: { d: 'm5 12 7-7 7 7', translateY: 0 },
  animate: {
    d: 'm5 12 7-7 7 7',
    translateY: [0, 3, 0],
    transition: {
      duration: 0.4
    }
  }
}

const SECOND_PATH_VARIANTS: Variants = {
  normal: { d: 'M12 19V5' },
  animate: {
    d: ['M12 19V5', 'M12 19V10', 'M12 19V5'],
    transition: {
      duration: 0.4
    }
  }
}

const icon: IconDefinition = {
  name: 'arrow-up',
  normal: 'normal',
  animate: ['animate'],
  render: (controls) =>
    h('g', {}, [
      h(
        motion.path,
        {
          animate: controls,
          d: 'm5 12 7-7 7 7',
          variants: PATH_VARIANTS,
          initial: 'normal'
        },
        () => []
      ),
      h(
        motion.path,
        {
          animate: controls,
          d: 'M12 19V5',
          variants: SECOND_PATH_VARIANTS,
          initial: 'normal'
        },
        () => []
      )
    ])
}

export default icon

import { h } from 'vue'
import { motion } from 'motion-v'
import type { IconDefinition } from './types'

/** 为暂无官方动画的 Lucide 图形提供一次轻微缩放。 */
export function createShapeIcon(
  name: string,
  nodes: Array<[string, Record<string, string>]>
): IconDefinition {
  return {
    name,
    normal: 'normal',
    animate: ['animate'],
    render: (controls) =>
      h(
        motion.g,
        {
          animate: controls,
          initial: 'normal',
          style: { transformOrigin: '12px 12px' },
          variants: {
            normal: { scale: 1 },
            animate: { scale: [1, 1.06, 1] }
          },
          transition: { duration: 0.4, ease: 'easeInOut' }
        },
        () => nodes.map(([tag, attributes]) => h(tag, attributes))
      )
  }
}

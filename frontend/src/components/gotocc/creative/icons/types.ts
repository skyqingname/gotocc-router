import type { VNodeChild } from 'vue'
import type { MotionProps, useAnimationControls } from 'motion-v'

export type IconControls = ReturnType<typeof useAnimationControls>
export type IconAnimation = Parameters<IconControls['start']>[0]
export type IconVariants = NonNullable<MotionProps<'g'>['variants']>

/** 图形定义只描述 SVG 和播放顺序，交互及生命周期由统一入口管理。 */
export interface IconDefinition {
  name: string
  normal: string
  animate: IconAnimation[]
  render: (controls: IconControls, strokeWidth: number) => VNodeChild
}

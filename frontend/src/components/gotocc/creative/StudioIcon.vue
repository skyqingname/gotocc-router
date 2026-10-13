<!-- Adapted from TokenFlux/TokenRouter f383062d. LGPL-3.0-or-later. -->
<script lang="ts">
import { defineComponent, h, normalizeClass, onMounted, onUpdated, ref, type PropType } from 'vue'
import { icons, type IconName } from './icons/registry'
import type { IconControls, IconDefinition } from './icons/types'
import { useIconAnimation } from './icons/useIconAnimation'

type IconSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'
// 导航使用 18px，普通按钮使用 16px，展示图标按场景选择更大档位。
const SIZE_CLASSES: Record<IconSize, [string, string]> = {
  xs: ['h-3', 'w-3'],
  sm: ['h-4', 'w-4'],
  md: ['h-[18px]', 'w-[18px]'],
  lg: ['h-6', 'w-6'],
  xl: ['h-8', 'w-8']
}

// 图形切换时重新挂载内部节点，根 SVG 继续保留焦点、尺寸和外层状态样式。
const Artwork = (props: {
  definition: IconDefinition
  controls: IconControls
  strokeWidth: number
}) => props.definition.render(props.controls, props.strokeWidth)

export default defineComponent({
  name: 'Icon',
  inheritAttrs: false,
  props: {
    name: { type: String as PropType<IconName>, required: true },
    size: { type: String as PropType<IconSize>, default: 'md' },
    strokeWidth: { type: Number, default: 1.75 },
    animateOnHover: { type: Boolean, default: true },
    animationActive: { type: Boolean, default: false }
  },
  setup(props, { attrs, slots }) {
    const svgRef = ref<SVGSVGElement | null>(null)
    const definition = () => icons[props.name]
    const controls = useIconAnimation(
      svgRef,
      definition,
      () => props.animateOnHover,
      () => props.animationActive
    )

    // motion-v 给内部 SVG 节点注册 focus 事件后，浏览器可能将其加入 Tab 顺序。
    // 挂载及换图后显式排除这些装饰节点，焦点和键盘操作继续由外层控件承担。
    const excludeArtworkFromFocus = () => {
      const artwork = svgRef.value?.querySelectorAll(
        'g, path, circle, rect, line, polyline, polygon, ellipse'
      )
      artwork?.forEach((node) => {
        node.setAttribute('tabindex', '-1')
        node.setAttribute('focusable', 'false')
      })
    }
    onMounted(excludeArtworkFromFocus)
    onUpdated(excludeArtworkFromFocus)

    return () => {
      const className = normalizeClass(attrs.class)
      const [height, width] = SIZE_CLASSES[props.size]
      const labelled = Boolean(attrs['aria-label'] || attrs['aria-labelledby'])

      return h(
        'svg',
        {
          xmlns: 'http://www.w3.org/2000/svg',
          fill: 'none',
          viewBox: '0 0 24 24',
          stroke: 'currentColor',
          'stroke-width': props.strokeWidth,
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          'aria-hidden': labelled ? undefined : true,
          role: labelled ? 'img' : undefined,
          focusable: 'false',
          tabindex: -1,
          ...attrs,
          ref: svgRef,
          'data-animated-icon': definition().name,
          // 调用方传入尺寸时使用该尺寸，省略 size 档位的尺寸类。
          class: [
            'shrink-0 select-none',
            !attrs.height &&
              !/(?:^|\s)(?:\S+:)?(?:h-|size-)/.test(className) &&
              height,
            !attrs.width &&
              !/(?:^|\s)(?:\S+:)?(?:w-|size-)/.test(className) &&
              width,
            className
          ]
        },
        [
          h(Artwork, {
            key: props.name,
            definition: definition(),
            controls,
            strokeWidth: props.strokeWidth
          }),
          slots.default?.()
        ]
      )
    }
  }
})
</script>

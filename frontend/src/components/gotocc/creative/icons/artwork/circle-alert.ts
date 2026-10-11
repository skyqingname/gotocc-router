// 图形来自 Lucide；官方暂无对应动画，使用统一轻微缩放。
// https://github.com/lucide-icons/lucide/blob/66d8f9fc394b8530377e5f6112f0b8908ba01280/icons/circle-alert.svg
import { createShapeIcon } from '../createShapeIcon'

export default createShapeIcon('circle-alert', [
  [
    'circle',
    {
      cx: '12',
      cy: '12',
      r: '10'
    }
  ],
  [
    'line',
    {
      x1: '12',
      x2: '12',
      y1: '8',
      y2: '12'
    }
  ],
  [
    'line',
    {
      x1: '12',
      x2: '12.01',
      y1: '16',
      y2: '16'
    }
  ]
])

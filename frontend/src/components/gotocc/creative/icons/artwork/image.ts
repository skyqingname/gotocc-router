// 图形来自 Lucide；官方暂无对应动画，使用统一轻微缩放。
// https://github.com/lucide-icons/lucide/blob/66d8f9fc394b8530377e5f6112f0b8908ba01280/icons/image.svg
import { createShapeIcon } from '../createShapeIcon'

export default createShapeIcon('image', [
  [
    'rect',
    {
      width: '18',
      height: '18',
      x: '3',
      y: '3',
      rx: '2',
      ry: '2'
    }
  ],
  [
    'circle',
    {
      cx: '9',
      cy: '9',
      r: '2'
    }
  ],
  [
    'path',
    {
      d: 'm21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21'
    }
  ]
])

// 图形来自 Lucide；官方暂无对应动画，使用统一轻微缩放。
// https://github.com/lucide-icons/lucide/blob/66d8f9fc394b8530377e5f6112f0b8908ba01280/icons/info.svg
import { createShapeIcon } from '../createShapeIcon'

export default createShapeIcon('info', [
  [
    'circle',
    {
      cx: '12',
      cy: '12',
      r: '10'
    }
  ],
  [
    'path',
    {
      d: 'M12 16v-4'
    }
  ],
  [
    'path',
    {
      d: 'M12 8h.01'
    }
  ]
])

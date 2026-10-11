// Derived from TokenFlux/TokenRouter f383062dd5fd534f462887acb70ea241ce0d8278. Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation.
/**
 * 方头画笔：PencilBrush 子类，用于创作台 mask 涂抹的"方"形状。
 *
 * 实现依据（fabric 7.4 源码，node_modules/fabric/dist/index.mjs 中 PencilBrush）：
 * - 预览阶段 PencilBrush._render 在 contextTop 上 _saveAndTransform(ctx) 应用 viewportTransform 后描边；
 *   其 onMouseMove 的增量分支调用的是「类名限定」的静态方法 PencilBrush.drawSegment，
 *   实例覆写无法拦截，因此方头笔必须走全量重绘路径（clearContext + _render）。
 * - 落笔结束 _finalizeAndAddPath 依次调用可覆写的实例方法 convertPointsToSVGPath / createPath 生成最终 Path。
 *
 * 方头笔迹用「等间距方形印章」实现：沿轨迹以 width/2 为步长插值出一串轴对齐方块，
 * 预览直接 fillRect 这些方块，最终 Path 用同样方块的填充子路径。
 * 这样斜向快速移动不会出现描边折线的斜接尖角 / 圆帽残留，预览与最终笔迹逐像素一致。
 */

import { Path, PencilBrush, Point, type TEvent, type TSimplePathData } from 'fabric'

export class SquarePencilBrush extends PencilBrush {
  /**
   * 鼠标移动：沿用父类的点采集与直线修饰键判断，但每次全量重绘方头预览，
   * 避免父类增量分支调用静态 drawSegment 画出不一致的圆头线段。
   */
  override onMouseMove(pointer: Point, options: TEvent): void {
    if (!this.canvas._isMainEvent(options.e)) return
    if (this.limitedToCanvasSize === true && this._isOutSideCanvas(pointer)) return
    if (this._addPoint(pointer)) {
      this.canvas.clearContext(this.canvas.contextTop)
      this._render()
    }
  }

  /**
   * 沿轨迹插值出等间距（width/2）的印章中心点；
   * 相邻点距离不足步长时至少保留端点，保证快速斜向移动下笔迹连续。
   */
  private stampCenters(points: Point[]): Point[] {
    const step = Math.max(1, this.width / 2)
    const centers: Point[] = []
    const push = (x: number, y: number) => {
      const last = centers[centers.length - 1]
      if (!last || Math.hypot(x - last.x, y - last.y) >= step * 0.9) {
        centers.push(new Point(x, y))
      }
    }
    push(points[0].x, points[0].y)
    for (let i = 1; i < points.length; i++) {
      const from = points[i - 1]
      const to = points[i]
      const dist = Math.hypot(to.x - from.x, to.y - from.y)
      const segments = Math.max(1, Math.ceil(dist / step))
      for (let s = 1; s <= segments; s++) {
        push(from.x + ((to.x - from.x) * s) / segments, from.y + ((to.y - from.y) * s) / segments)
      }
    }
    return centers
  }

  /** 在 contextTop 上绘制方头预览：所有印章轴对齐 fillRect，任意方向粗细一致 */
  override _render(ctx: CanvasRenderingContext2D = this.canvas.contextTop): void {
    const points = this._points
    if (!points.length) return
    this._saveAndTransform(ctx)
    ctx.fillStyle = this.color
    const half = this.width / 2
    for (const center of this.stampCenters(points)) {
      ctx.fillRect(center.x - half, center.y - half, this.width, this.width)
    }
    ctx.restore()
  }

  /**
   * 点集 → 填充方块子路径（M/L 闭合，不用 Z，兼容 TSimplePathData 最简命令集），
   * 与预览印章一一对应，保证最终 Path 与预览视觉一致。
   */
  override convertPointsToSVGPath(points: Point[]): TSimplePathData {
    const path: TSimplePathData = []
    if (!points.length) return path
    const half = this.width / 2
    for (const center of this.stampCenters(points)) {
      const left = center.x - half
      const top = center.y - half
      const right = center.x + half
      const bottom = center.y + half
      path.push(
        ['M', left, top],
        ['L', right, top],
        ['L', right, bottom],
        ['L', left, bottom],
        ['L', left, top],
      )
    }
    return path
  }

  /** 最终 Path 由填充方块组成，斜向笔迹的端点和连接处保持方形。 */
  override createPath(pathData: TSimplePathData): Path {
    const path = new Path(pathData, {
      fill: this.color,
      stroke: '',
      strokeWidth: 0,
    })
    return path
  }
}

import { shallowRef } from 'vue'

/**
 * 让统一悬停提示（components/common/TitleTooltip.vue）在刚按下的触点上短暂换成一句反馈，
 * 例如复制成功后显示「已复制」。显示多久见根目录 console-guide.json 的 tooltip.copied_ms。
 */
export const tipFlash = shallowRef<{ text: string } | null>(null)

export function flashTip(text: string) {
  tipFlash.value = { text }
}

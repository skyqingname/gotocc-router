/**
 * Common component types
 */

export interface Column {
  key: string
  label: string
  sortable?: boolean
  /** Pin this column using its measured width; false opts out of default pinning. */
  sticky?: 'left' | 'right' | false
  class?: string
  formatter?: (value: any, row: any) => string
  /** 表头悬停提示：一句话说明这一列是什么。 */
  tip?: string
}

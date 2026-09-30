import { describe, expect, it } from 'vitest'
import * as XLSX from 'xlsx'

describe('usage workbook export compatibility', () => {
  it('round-trips appended usage rows with Unicode, numbers and literal formula-like text', () => {
    const headers = ['模型', '输入 Token', '备注']
    const rows = [
      ['gpt-4.1', 42, '=1+1'],
      ['中文模型', 0, 'line one\nline two'],
    ]
    const sheet = XLSX.utils.aoa_to_sheet([headers])
    XLSX.utils.sheet_add_aoa(sheet, rows, { origin: -1 })
    const workbook = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(workbook, sheet, 'Usage')
    const bytes = XLSX.write(workbook, { bookType: 'xlsx', type: 'array' })
    const restored = XLSX.read(bytes, { type: 'array' })
    expect(restored.SheetNames).toEqual(['Usage'])
    expect(XLSX.utils.sheet_to_json(restored.Sheets.Usage, { header: 1 })).toEqual([headers, ...rows])
    expect(restored.Sheets.Usage.C2.t).toBe('s')
    expect(restored.Sheets.Usage.C2.f).toBeUndefined()
  })
})

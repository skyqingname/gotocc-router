// Derived from TokenFlux/TokenRouter e74bba115 (v0.7.3) frontend/src/components/announcements/announcementDisplay.ts.
// Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation.
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { formatDate, formatRelativeTime } from '@/utils/format'

// 公告正文的安全渲染、一行摘要和发布日期。顶栏公告与仪表盘公告时间线读同一份管理员写的内容，
// 解析参数和净化只在这里做一次，两处显示保持一致。

// 每次解析都显式传入参数：别的页面会改动共享 marked 实例的默认值。
const MARKDOWN_OPTIONS = { async: false, breaks: true, gfm: true } as const
// 中日文字符之间换行不补空格，其他文字换行处补一个空格。
const CJK_CHARACTER = /[⺀-鿿＀-￯]/
// 发布七天内显示相对时间，更早显示日期。
const RELATIVE_DATE_WINDOW_MS = 7 * 24 * 60 * 60 * 1000

export function renderAnnouncementHtml(content: string): string {
  return DOMPurify.sanitize(marked.parse(content, MARKDOWN_OPTIONS))
}

// 取第一段非空正文作纯文本摘要。表格和代码块先去掉，没有段落时用剩余正文；只有表格或代码时返回空串，由调用方决定是否显示占位。
export function summarizeAnnouncement(content: string): string {
  const parsed = new DOMParser().parseFromString(renderAnnouncementHtml(content), 'text/html')
  parsed.body.querySelectorAll('table, pre').forEach((element) => element.replaceWith(' '))
  parsed.body.querySelectorAll('br').forEach((lineBreak) => {
    const before = lineBreak.previousSibling?.textContent?.trimEnd().slice(-1) ?? ''
    const after = lineBreak.nextSibling?.textContent?.trimStart().charAt(0) ?? ''
    lineBreak.replaceWith(CJK_CHARACTER.test(before) && CJK_CHARACTER.test(after) ? '' : ' ')
  })
  const paragraph = [...parsed.body.querySelectorAll('p')].find((element) => element.textContent?.trim())
  return (paragraph?.textContent || parsed.body.textContent || '').replace(/\s+/g, ' ').trim()
}

// 同年省略年份，跨年补上年份。
export function formatAnnouncementDate(createdAt: string): string {
  const date = new Date(createdAt)
  const options: Intl.DateTimeFormatOptions = { month: 'long', day: 'numeric' }
  if (date.getFullYear() !== new Date().getFullYear()) options.year = 'numeric'
  return formatDate(date, options)
}

export function formatAnnouncementRowDate(createdAt: string): string {
  const age = Date.now() - Date.parse(createdAt)
  return age >= 0 && age < RELATIVE_DATE_WINDOW_MS ? formatRelativeTime(createdAt) : formatAnnouncementDate(createdAt)
}

// Derived from TokenFlux/TokenRouter 38ca040de (v0.7.3) frontend/src/utils/fuzzyMatch.ts.
// Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation.

// 全局搜索的模糊匹配：给候选文字打分并标出命中的字符，用于排序和高亮。
// 比较前统一转小写并去掉空格、连字符、下划线和点，所以「opus55」能找到「claude-opus-5-5」；
// 高亮仍落在原文上：归一化时记下每个字符在原文中的位置。中文按字逐个比较。
// 分数分档且档差大于档内扣分：完全相同 > 前缀 > 词首 > 任意连续子串 > 按顺序散落的子序列。

const TIER = { exact: 1000, prefix: 800, wordStart: 600, substring: 400, subsequence: 200 } as const
// 档内按出现位置或散落跨度扣分，上限低于档差，弱档永远排不到强档前面。
const MAX_PENALTY = 150
const SEPARATORS = new Set([' ', '-', '_', '.'])
const WORD_BREAKS = new Set([' ', '-', '_', '.', '/', ':', '(', ')', '[', ']', '@'])

export interface MatchRange {
  start: number
  end: number
}

export interface FuzzyMatch {
  score: number
  ranges: MatchRange[]
}

// 命中最好的字段序号：调用方把可见标题放在第 0 个字段，只有它命中时才高亮。
export interface FieldMatch extends FuzzyMatch {
  field: number
}

interface NormalizedText {
  value: string
  sourceIndex: number[]
  wordStart: boolean[]
}

function normalizeQuery(query: string): string {
  let result = ''
  for (const char of query.toLowerCase()) {
    if (!SEPARATORS.has(char)) result += char
  }
  return result
}

// 词首：第一个字符、分隔符之后的字符，或小写后面紧跟的大写字母。
function normalizeText(text: string): NormalizedText {
  const value: string[] = []
  const sourceIndex: number[] = []
  const wordStart: boolean[] = []
  for (let index = 0; index < text.length; index++) {
    const char = text[index]
    if (SEPARATORS.has(char)) continue
    const previous = index > 0 ? text[index - 1] : ''
    const camelBoundary = previous !== '' && previous === previous.toLowerCase() && previous !== previous.toUpperCase() && char !== char.toLowerCase()
    value.push(char.toLowerCase())
    sourceIndex.push(index)
    wordStart.push(index === 0 || WORD_BREAKS.has(previous) || camelBoundary)
  }
  return { value: value.join(''), sourceIndex, wordStart }
}

// 归一化后的命中位置换回原文区间，相邻的合并成一段。
function toRanges(normalized: NormalizedText, offsets: number[]): MatchRange[] {
  const ranges: MatchRange[] = []
  for (const offset of offsets) {
    const source = normalized.sourceIndex[offset]
    const last = ranges[ranges.length - 1]
    if (last && last.end === source) last.end = source + 1
    else ranges.push({ start: source, end: source + 1 })
  }
  return ranges
}

const contiguous = (start: number, length: number) => Array.from({ length }, (_, index) => start + index)

function findWordStart(normalized: NormalizedText, query: string): number {
  let offset = normalized.value.indexOf(query)
  while (offset !== -1) {
    if (normalized.wordStart[offset]) return offset
    offset = normalized.value.indexOf(query, offset + 1)
  }
  return -1
}

function findSubsequence(normalized: NormalizedText, query: string): number[] | null {
  const offsets: number[] = []
  let cursor = 0
  for (const char of query) {
    const offset = normalized.value.indexOf(char, cursor)
    if (offset === -1) return null
    offsets.push(offset)
    cursor = offset + 1
  }
  return offsets
}

export function fuzzyMatch(query: string, text: string): FuzzyMatch | null {
  const normalizedQuery = normalizeQuery(query)
  if (normalizedQuery === '' || text === '') return null
  const normalized = normalizeText(text)
  const length = normalizedQuery.length

  if (normalized.value === normalizedQuery) return { score: TIER.exact, ranges: toRanges(normalized, contiguous(0, length)) }
  if (normalized.value.startsWith(normalizedQuery)) return { score: TIER.prefix, ranges: toRanges(normalized, contiguous(0, length)) }
  const wordOffset = findWordStart(normalized, normalizedQuery)
  if (wordOffset !== -1) {
    return { score: TIER.wordStart - Math.min(wordOffset, MAX_PENALTY), ranges: toRanges(normalized, contiguous(wordOffset, length)) }
  }
  const substringOffset = normalized.value.indexOf(normalizedQuery)
  if (substringOffset !== -1) {
    return { score: TIER.substring - Math.min(substringOffset, MAX_PENALTY), ranges: toRanges(normalized, contiguous(substringOffset, length)) }
  }
  const offsets = findSubsequence(normalized, normalizedQuery)
  if (!offsets) return null
  // 跨度 = 首尾命中之间跳过的字符数，越集中排得越前。
  const spread = offsets[offsets.length - 1] - offsets[0] + 1 - length
  return { score: TIER.subsequence - Math.min(spread, MAX_PENALTY), ranges: toRanges(normalized, offsets) }
}

// 多个字段中分数最高的一个；同分时靠前的字段优先。
export function bestMatch(query: string, fields: readonly string[]): FieldMatch | null {
  let best: FieldMatch | null = null
  for (let field = 0; field < fields.length; field++) {
    const match = fuzzyMatch(query, fields[field])
    if (match && (best === null || match.score > best.score)) best = { ...match, field }
  }
  return best
}

export interface TextSegment {
  text: string
  matched: boolean
}

// 按命中区间把文字切成普通段和高亮段。
export function highlightSegments(text: string, ranges: readonly MatchRange[]): TextSegment[] {
  const segments: TextSegment[] = []
  let cursor = 0
  for (const range of ranges) {
    if (range.start > cursor) segments.push({ text: text.slice(cursor, range.start), matched: false })
    segments.push({ text: text.slice(range.start, range.end), matched: true })
    cursor = range.end
  }
  if (cursor < text.length) segments.push({ text: text.slice(cursor), matched: false })
  return segments
}

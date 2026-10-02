/** Preview/validation contract for the recurring-rate engine. The server remains
 * the billing authority; never use this preview to submit an authoritative price.
 */
export const MAX_RATE_SCHEDULE_RULES = 64

export interface RateScheduleRule {
  id: string
  enabled: boolean
  start: string
  end: string
  multiplier: number
}

export interface RateScheduleConfig {
  enabled: boolean
  timezone: string
  rules: RateScheduleRule[]
}

export type RateScheduleErrorCode =
  | 'invalid_timezone'
  | 'invalid_enabled'
  | 'invalid_rules'
  | 'too_many_rules'
  | 'invalid_id'
  | 'duplicate_id'
  | 'invalid_time'
  | 'empty_window'
  | 'invalid_multiplier'
  | 'overlapping_windows'
  | 'invalid_base'
  | 'invalid_result'

export class RateScheduleError extends Error {
  constructor(public readonly code: RateScheduleErrorCode, public readonly field: string) {
    super(`rate schedule ${field}: ${code}`)
    this.name = 'RateScheduleError'
  }
}

export interface RateSchedulePreview {
  readonly timezone: string
  readonly rule_id: string | null
  readonly applied: boolean
  readonly received_at: string
  readonly base_multiplier: number
  readonly schedule_multiplier: number
  readonly effective_multiplier: number
}

export interface CompiledRateSchedule {
  resolve: (receivedAt: Date, baseMultiplier: number) => Readonly<RateSchedulePreview>
}

interface Interval {
  start: number
  end: number
  id: string
  factor: number
}

function parseClock(value: string, allowEndOfDay: boolean): number | null {
  if (typeof value !== 'string' || !/^\d{1,2}:\d{2}$/.test(value)) return null
  const [hour = NaN, minute = NaN] = value.split(':').map(Number)
  if (allowEndOfDay && hour === 24 && minute === 0) return 1440
  if (hour > 23 || minute > 59) return null
  return hour * 60 + minute
}

function finiteNonNegative(value: number): boolean {
  return Number.isFinite(value) && value >= 0
}

export function compileRateSchedule(
  config: RateScheduleConfig,
  serverTimezone: string
): CompiledRateSchedule {
  const zone = config.timezone || serverTimezone
  if (
    typeof zone !== 'string' || zone.length > 128 ||
    (zone !== 'UTC' && (!zone.includes('/') || zone.startsWith('/') || !/^[A-Za-z0-9/_+-]+$/.test(zone)))
  ) {
    throw new RateScheduleError('invalid_timezone', 'timezone')
  }
  let clock: Intl.DateTimeFormat
  try {
    clock = new Intl.DateTimeFormat('en-GB-u-nu-latn', {
      timeZone: zone, hourCycle: 'h23', hour: '2-digit', minute: '2-digit'
    })
  } catch {
    throw new RateScheduleError('invalid_timezone', 'timezone')
  }
  if (typeof config.enabled !== 'boolean') {
    throw new RateScheduleError('invalid_enabled', 'enabled')
  }
  if (!Array.isArray(config.rules)) {
    throw new RateScheduleError('invalid_rules', 'rules')
  }
  if (config.rules.length > MAX_RATE_SCHEDULE_RULES) {
    throw new RateScheduleError('too_many_rules', 'rules')
  }
  const ids = new Set<string>()
  const windows: Interval[] = []
  for (const [index, rule] of config.rules.entries()) {
    const field = `rules[${index}]`
    if (typeof rule.id !== 'string' || !/^[A-Za-z0-9_-]{1,64}$/.test(rule.id)) {
      throw new RateScheduleError('invalid_id', `${field}.id`)
    }
    if (ids.has(rule.id)) throw new RateScheduleError('duplicate_id', `${field}.id`)
    ids.add(rule.id)
    if (typeof rule.enabled !== 'boolean') {
      throw new RateScheduleError('invalid_enabled', `${field}.enabled`)
    }
    const start = parseClock(rule.start, false)
    const end = parseClock(rule.end, true)
    if (start === null) throw new RateScheduleError('invalid_time', `${field}.start`)
    if (end === null) throw new RateScheduleError('invalid_time', `${field}.end`)
    if (start === end) throw new RateScheduleError('empty_window', `${field}.end`)
    if (!finiteNonNegative(rule.multiplier)) {
      throw new RateScheduleError('invalid_multiplier', `${field}.multiplier`)
    }
    if (!rule.enabled) continue
    if (start < end) {
      windows.push({ start, end, id: rule.id, factor: rule.multiplier })
    } else {
      windows.push({ start, end: 1440, id: rule.id, factor: rule.multiplier })
      if (end > 0) windows.push({ start: 0, end, id: rule.id, factor: rule.multiplier })
    }
  }
  windows.sort((a, b) => a.start - b.start)
  for (let index = 1; index < windows.length; index++) {
    if (windows[index].start < windows[index - 1].end) {
      throw new RateScheduleError('overlapping_windows', 'rules')
    }
  }
  // Capture primitives, never a reference to the editable form/configuration.
  const enabled = config.enabled
  return Object.freeze({
    resolve(receivedAt: Date, baseMultiplier: number): Readonly<RateSchedulePreview> {
      if (!Number.isFinite(receivedAt.getTime())) {
        throw new RateScheduleError('invalid_time', 'received_at')
      }
      if (!finiteNonNegative(baseMultiplier)) {
        throw new RateScheduleError('invalid_base', 'base_multiplier')
      }
      const parts = clock.formatToParts(receivedAt)
      const hour = Number(parts.find(part => part.type === 'hour')?.value)
      const minute = Number(parts.find(part => part.type === 'minute')?.value)
      const current = hour * 60 + minute
      if (!Number.isInteger(current) || current < 0 || current >= 1440) {
        throw new RateScheduleError('invalid_time', 'received_at')
      }
      const matched = enabled
        ? windows.find(window => current >= window.start && current < window.end)
        : undefined
      const factor = matched?.factor ?? 1
      const effective = baseMultiplier * factor
      if (!finiteNonNegative(effective) || (baseMultiplier > 0 && factor > 0 && effective === 0)) {
        throw new RateScheduleError('invalid_result', 'effective_multiplier')
      }
      return Object.freeze({
        timezone: zone, rule_id: matched?.id ?? null, applied: matched !== undefined,
        received_at: receivedAt.toISOString(), base_multiplier: baseMultiplier,
        schedule_multiplier: factor, effective_multiplier: effective
      })
    }
  })
}

import { describe, expect, it } from 'vitest'
import {
  compileRateSchedule,
  MAX_RATE_SCHEDULE_RULES,
  RateScheduleError,
  type RateScheduleConfig,
  type RateScheduleRule
} from '../rate-schedule'

const rule = (patch: Partial<RateScheduleRule> = {}): RateScheduleRule => ({
  id: 'night', enabled: true, start: '22:00', end: '06:00', multiplier: 0.5, ...patch
})
const config = (rules = [rule()], patch: Partial<RateScheduleConfig> = {}): RateScheduleConfig => ({
  enabled: true, timezone: 'Asia/Shanghai', rules, ...patch
})
const compile = (value = config()) => compileRateSchedule(value, 'UTC')

function expectCode(run: () => unknown, code: string): void {
  let caught: unknown
  try { run() } catch (error) { caught = error }
  expect(caught).toBeInstanceOf(RateScheduleError)
  expect((caught as RateScheduleError).code).toBe(code)
}

describe('recurring rate schedule', () => {
  it.each([
    ['2026-09-20T21:59:59+08:00', null, 1],
    ['2026-09-20T22:00:00+08:00', 'night', 0.5],
    ['2026-09-21T00:00:00+08:00', 'night', 0.5],
    ['2026-09-21T05:59:59+08:00', 'night', 0.5],
    ['2026-09-21T06:00:00+08:00', null, 1],
    ['2026-09-20T14:00:00Z', 'night', 0.5]
  ] as const)('matches %s using the explicit timezone', (at, id, factor) => {
    const result = compile().resolve(new Date(at), 0.4)
    expect(result.rule_id).toBe(id)
    expect(result.applied).toBe(id !== null)
    expect(result.schedule_multiplier).toBe(factor)
    expect(result.effective_multiplier).toBeCloseTo(0.4 * factor)
  })

  it('preserves user-resolved bases, zero factors, gaps, and adjacent windows', () => {
    const schedule = compile(config([
      rule(), rule({ id: 'peak', start: '18:00', end: '22:00', multiplier: 1.5 }),
      rule({ id: 'free', start: '12:00', end: '13:00', multiplier: 0 })
    ]))
    expect(schedule.resolve(new Date('2026-09-20T20:00:00+08:00'), 0.2).effective_multiplier).toBeCloseTo(0.3)
    expect(schedule.resolve(new Date('2026-09-20T12:00:00+08:00'), 0.2).effective_multiplier).toBe(0)
    expect(schedule.resolve(new Date('2026-09-20T13:00:00+08:00'), 0.2).effective_multiplier).toBe(0.2)
  })

  it('does not turn an empty or disabled schedule into a base multiplier of one', () => {
    for (const value of [config([]), config([rule()], { enabled: false }), config([rule({ enabled: false })])]) {
      const result = compile(value).resolve(new Date('2026-09-20T23:00:00+08:00'), 0.4)
      expect(result.effective_multiplier).toBe(0.4)
      expect(result.applied).toBe(false)
    }
  })

  it('covers every minute of the explicit all-day and midnight-end forms', () => {
    for (const [start, end, from] of [['00:00', '24:00', 0], ['18:00', '00:00', 1080]] as const) {
      const schedule = compile(config([rule({ start, end })], { timezone: 'UTC' }))
      for (let minute = 0; minute < 1440; minute++) {
        const at = new Date(Date.UTC(2026, 8, 20, 0, minute))
        expect(schedule.resolve(at, 0.4).applied).toBe(minute >= from)
      }
    }
  })

  it.each([
    [{ id: '' }, 'invalid_id'], [{ id: 'bad\nid' }, 'invalid_id'],
    [{ start: '24:00' }, 'invalid_time'], [{ start: '1:1' }, 'invalid_time'],
    [{ start: ' 01:00' }, 'invalid_time'], [{ end: '24:01' }, 'invalid_time'],
    [{ start: '06:00' }, 'empty_window'], [{ multiplier: -1 }, 'invalid_multiplier'],
    [{ multiplier: NaN }, 'invalid_multiplier'], [{ multiplier: Infinity }, 'invalid_multiplier']
  ] as const)('rejects invalid rule %j', (patch, code) => {
    expectCode(() => compile(config([rule(patch)])), code)
  })

  it('rejects duplicate IDs and active overlaps, including across midnight', () => {
    expectCode(() => compile(config([rule(), rule()])), 'duplicate_id')
    expectCode(() => compile(config([rule(), rule({ id: 'overlap', start: '05:00', end: '07:00' })])), 'overlapping_windows')
    expectCode(() => compile(config([rule(), rule({ id: 'overlap', start: '21:00', end: '23:00' })])), 'overlapping_windows')
    expect(() => compile(config([rule(), rule({ id: 'off', enabled: false })]))).not.toThrow()
  })

  it('inherits a supplied server zone but never guesses a browser zone', () => {
    expect(compileRateSchedule(config([], { timezone: '' }), 'Asia/Shanghai')
      .resolve(new Date('2026-09-20T00:00:00Z'), 1).timezone).toBe('Asia/Shanghai')
    for (const timezone of ['Local', 'Invalid/Zone', '../etc/passwd']) {
      expectCode(() => compile(config([], { timezone })), 'invalid_timezone')
    }
    expectCode(() => compileRateSchedule(config([], { timezone: '' }), ''), 'invalid_timezone')
    expectCode(() => compile(config(Array.from({ length: MAX_RATE_SCHEDULE_RULES + 1 }, () => rule()))), 'too_many_rules')
  })

  it('handles both repeated DST hours and the skipped spring hour', () => {
    const repeated = compile(config([rule({ start: '01:00', end: '02:00' })], { timezone: 'America/New_York' }))
    for (const at of ['2025-11-02T05:30:00Z', '2025-11-02T06:30:00Z']) {
      expect(repeated.resolve(new Date(at), 0.4).effective_multiplier).toBe(0.2)
    }
    const skipped = compile(config([rule({ start: '02:00', end: '03:00' })], { timezone: 'America/New_York' }))
    for (const at of ['2025-03-09T06:59:59Z', '2025-03-09T07:00:00Z']) {
      expect(skipped.resolve(new Date(at), 0.4).applied).toBe(false)
    }
  })

  it('does not mutate inputs or a snapshot when rules or time change', () => {
    const input = config([rule({ start: '1:30', end: '02:00' })], { timezone: 'UTC' })
    const schedule = compile(input)
    const before = schedule.resolve(new Date('2026-09-20T01:59:59Z'), 0.2)
    input.rules[0].multiplier = 10
    input.enabled = false
    expect(input.rules[0].start).toBe('1:30')
    expect(schedule.resolve(new Date('2026-09-20T01:59:59Z'), 0.2)).toEqual(before)
    expect(schedule.resolve(new Date('2026-09-20T02:00:00Z'), 0.2).applied).toBe(false)
    expect(before.effective_multiplier).toBe(0.1)
    expect(Object.isFrozen(before)).toBe(true)
  })

  it('rejects invalid timestamps, invalid bases, overflow and accidental free underflow', () => {
    const schedule = compile(config([rule({ start: '00:00', end: '24:00', multiplier: 2 })]))
    const at = new Date('2026-09-20T00:00:00Z')
    for (const base of [-1, NaN, Infinity, -Infinity]) {
      expectCode(() => schedule.resolve(at, base), 'invalid_base')
    }
    expectCode(() => schedule.resolve(new Date(NaN), 1), 'invalid_time')
    expectCode(() => schedule.resolve(at, Number.MAX_VALUE), 'invalid_result')
    const tiny = compile(config([rule({ start: '00:00', end: '24:00', multiplier: 0.1 })]))
    expectCode(() => tiny.resolve(at, Number.MIN_VALUE), 'invalid_result')
  })
})

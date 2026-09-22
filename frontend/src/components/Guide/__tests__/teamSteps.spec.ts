import { describe, expect, it } from 'vitest'

import { getTeamSteps } from '../steps'

const translate = (key: string) => key

describe('getTeamSteps', () => {
  it('shows owner entry points and returns to the Plus team usage overview', () => {
    const steps = getTeamSteps(translate, true)

    expect(steps.map((step) => step.element)).toEqual([
      undefined,
      '[data-tour="team-members"]',
      '[data-tour="team-invitations"]',
      '[data-tour="team-settings-tab"]',
      '[data-tour="sidebar-my-keys"]',
      '[data-tour="keys-scope-switch"]',
      '[data-tour="keys-create-btn"]',
      '[data-tour="sidebar-usage"]',
      '[data-tour="team-member-usage-charts"]',
    ])
    expect(steps.at(-1)?.route).toEqual({ path: '/team' })
  })

  it('hides owner controls from members and targets their own usage records', () => {
    const steps = getTeamSteps(translate, false)
    const elements = steps.map((step) => step.element)

    expect(elements).toContain('[data-tour="team-limit-progress"]')
    expect(elements).toContain('[data-tour="team-usage-records"]')
    expect(elements).not.toContain('[data-tour="team-invitations"]')
    expect(elements).not.toContain('[data-tour="team-settings-tab"]')
  })

  it('opens the team key scope without making the scope switch interactive', () => {
    const steps = getTeamSteps(translate, true)
    const scopeStep = steps.find((step) => step.element === '[data-tour="keys-scope-switch"]')

    expect(scopeStep?.route).toEqual({ path: '/keys', query: { scope: 'team' } })
    expect(scopeStep?.popover?.showButtons).toBeUndefined()
  })

  it('keeps the no-team guide on read-only team page entry points', () => {
    const steps = getTeamSteps(translate, false, false)

    expect(steps.map((step) => step.element)).toEqual([
      undefined,
      '[data-tour="team-create-form"]',
      '[data-tour="sidebar-my-keys"]',
      '[data-tour="sidebar-usage"]',
    ])
    expect(steps.every((step) => step.route?.path === '/team')).toBe(true)
  })
})

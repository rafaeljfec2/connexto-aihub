import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { ProviderQuotaMetric } from '../types/quota'
import { ConsoleBudgetsPanel } from './ConsoleBudgetsPanel'

const budgets: ProviderQuotaMetric[] = [
  {
    metric: 'budget_rule_sanitized_budget_1',
    label: 'Console budget',
    nature: 'cost',
    used: 15,
    limit: 20,
    unit: 'USD',
    period: 'model/models',
    scope: 'each',
    resets_at: '2026-11-01T00:00:00Z',
  },
]

describe('ConsoleBudgetsPanel', () => {
  it('renders official spent and limit', () => {
    render(
      <ConsoleBudgetsPanel
        budgets={budgets}
        now={new Date('2026-10-31T12:00:00.000Z')}
        testIdPrefix="drawer"
      />,
    )

    expect(screen.getByTestId('drawer-panel')).toBeTruthy()
    expect(screen.getByTestId('drawer-budget_rule_sanitized_budget_1').textContent).toContain(
      'Usado',
    )
    expect(screen.getByTestId('drawer-budget_rule_sanitized_budget_1').textContent).toMatch(/15/)
    expect(screen.getByTestId('drawer-budget_rule_sanitized_budget_1').textContent).toMatch(/20/)
  })
})

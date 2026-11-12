import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ConsumptionChart } from './ConsumptionChart'

describe('ConsumptionChart headline', () => {
  it('does not label a provider-only peak as the aggregate limit', () => {
    render(
      <ConsumptionChart
        daily={[{ period_start: '2026-11-12T00:00:00Z', admitted_count: 12 }]}
        totalAdmitted={12}
        quotaUtilization={[
          {
            policy_id: 2,
            scope: 'provider',
            provider_id: 'anthropic',
            daily_limit: 20,
            enabled: true,
            period_start: '',
            reset_at: '',
            admitted_count: 18,
            remaining_requests: 2,
            utilization_percent: 90,
          },
        ]}
        periodLabel="Últimos 7 dias"
      />,
    )

    expect(screen.getByText('12')).toBeTruthy()
    expect(screen.getByText('ADMISSÕES NO PERÍODO')).toBeTruthy()
    expect(screen.queryByText('DO LIMITE AGREGADO')).toBeNull()
  })
})

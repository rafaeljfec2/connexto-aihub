import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { QuotaPoliciesTable } from './QuotaPoliciesTable'

describe('QuotaPoliciesTable', () => {
  it('shows loading copy instead of empty state while pending', () => {
    render(
      <QuotaPoliciesTable policies={[]} utilization={[]} isLoading />,
    )
    expect(screen.getByText('Loading quota policies…')).toBeTruthy()
    expect(screen.queryByText('No resource policies configured.')).toBeNull()
  })

  it('shows No limit badge when enabled policy has zero daily limit', () => {
    render(
      <QuotaPoliciesTable
        policies={[
          {
            id: 1,
            scope: 'global',
            daily_limit: 0,
            enabled: true,
            updated_at: '2026-11-03T00:00:00Z',
          },
        ]}
        utilization={[
          {
            policy_id: 1,
            scope: 'global',
            daily_limit: 0,
            enabled: true,
            period_start: '2026-11-03T00:00:00Z',
            reset_at: '2026-11-04T00:00:00Z',
            admitted_count: 0,
            remaining_requests: 0,
            utilization_percent: 0,
          },
        ]}
      />,
    )

    expect(screen.getByText('No limit')).toBeTruthy()
  })
})

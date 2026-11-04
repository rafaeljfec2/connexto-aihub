import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ProviderUsageSection } from './ProviderUsageSection'

describe('ProviderUsageSection', () => {
  it('shows Unknown for null token fields and counts measurement statuses', () => {
    render(
      <ProviderUsageSection
        usage={{
          from: '2026-11-01T00:00:00Z',
          to: '2026-11-07T00:00:00Z',
          input_tokens: null,
          output_tokens: 0,
          total_tokens: null,
          reported_executions: 3,
          partial_executions: 1,
          unavailable_executions: 2,
          invalid_executions: 1,
          daily: [],
        }}
        distribution={{ from: '', to: '', by_provider: [], by_model: [] }}
      />,
    )

    expect(screen.getAllByText('Unknown').length).toBeGreaterThanOrEqual(2)
    expect(screen.getByText('0')).toBeTruthy()
    expect(screen.getByText('3')).toBeTruthy()
    expect(screen.getByText('Partial')).toBeTruthy()
    expect(screen.getByText('Unavailable')).toBeTruthy()
  })

  it('surfaces distribution failure while summary is shown', () => {
    render(
      <ProviderUsageSection
        usage={{
          from: '2026-11-01T00:00:00Z',
          to: '2026-11-07T00:00:00Z',
          input_tokens: 10,
          output_tokens: null,
          total_tokens: null,
          reported_executions: 1,
          partial_executions: 0,
          unavailable_executions: 0,
          invalid_executions: 0,
          daily: [],
        }}
        distribution={undefined}
        isDistributionError
      />,
    )

    expect(screen.getByText(/distribution endpoint failed/i)).toBeTruthy()
    expect(screen.getAllByText('Distribution data unavailable.').length).toBe(2)
  })
})

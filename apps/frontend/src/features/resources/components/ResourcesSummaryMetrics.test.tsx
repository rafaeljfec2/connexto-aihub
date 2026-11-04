import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ResourcesSummaryMetrics } from './ResourcesSummaryMetrics'

describe('ResourcesSummaryMetrics', () => {
  it('shows ellipsis while loading instead of zero placeholders', () => {
    render(
      <ResourcesSummaryMetrics
        configuredPolicies={0}
        peakUtilizationPercent={null}
        concurrencyActive={0}
        concurrencyLimit={0}
        nearLimitAlerts={0}
        isLoading
      />,
    )

    expect(screen.getAllByText('…').length).toBe(4)
  })
})

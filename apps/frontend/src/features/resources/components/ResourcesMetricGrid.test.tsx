import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ResourcesMetricGrid } from './ResourcesMetricGrid'

describe('ResourcesMetricGrid', () => {
  it('shows ellipsis while loading instead of zero placeholders', () => {
    render(
      <ResourcesMetricGrid
        configuredPolicies={0}
        globalUtilizationPercent={null}
        concurrencyActive={0}
        concurrencyLimit={0}
        nearLimitAlerts={0}
        isLoading
      />,
    )

    expect(screen.getAllByText('…').length).toBe(4)
  })

  it('shows em dash for missing global utilization', () => {
    render(
      <ResourcesMetricGrid
        configuredPolicies={1}
        globalUtilizationPercent={null}
        concurrencyActive={0}
        concurrencyLimit={0}
        nearLimitAlerts={0}
      />,
    )

    expect(screen.getByText('Uso agregado').parentElement?.nextElementSibling?.textContent).toBe('—')
  })
})

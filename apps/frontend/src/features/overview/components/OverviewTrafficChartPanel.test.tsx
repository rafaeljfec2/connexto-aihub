import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { OverviewTrafficChartPanel } from './OverviewTrafficChartPanel'

describe('OverviewTrafficChartPanel', () => {
  it('renders empty history state when no sufficient history exists', () => {
    render(
      <OverviewTrafficChartPanel
        traffic={{
          range: '24h',
          total_requests: 0,
          trend_percentage: 0,
          buckets: [],
          has_sufficient_history: false,
        }}
      />,
    )

    expect(screen.getByText('Volume de requisições')).toBeTruthy()
    expect(
      screen.getByText('Sem histórico suficiente nas últimas 24 horas'),
    ).toBeTruthy()
    expect(screen.getByText('00:00')).toBeTruthy()
    expect(screen.getByText('Agora')).toBeTruthy()
  })

  it('renders chart when sufficient history exists', () => {
    const { container } = render(
      <div style={{ width: 560, height: 200 }}>
        <OverviewTrafficChartPanel
          traffic={{
            range: '24h',
            total_requests: 82418,
            trend_percentage: 14.2,
            buckets: [
              { timestamp: '2026-11-03T00:00:00Z', label: '00:00', count: 1200 },
              { timestamp: '2026-11-03T06:00:00Z', label: '06:00', count: 1800 },
              { timestamp: '2026-11-03T12:00:00Z', label: '12:00', count: 2400 },
              { timestamp: '2026-11-03T18:00:00Z', label: '18:00', count: 3200 },
              { timestamp: '2026-11-03T20:00:00Z', label: 'Agora', count: 3800 },
            ],
            has_sufficient_history: true,
          }}
        />
      </div>,
    )

    expect(screen.getByText('82.418')).toBeTruthy()
    expect(screen.getByText('+14.2%')).toBeTruthy()
    expect(container.querySelector('.recharts-area')).toBeTruthy()
  })
})

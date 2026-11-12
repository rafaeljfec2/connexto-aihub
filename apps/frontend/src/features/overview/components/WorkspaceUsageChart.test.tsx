import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { ProviderUsageChart } from '../types/quota'
import { WorkspaceUsageChart } from './WorkspaceUsageChart'

const chart: ProviderUsageChart = {
  title: 'Workspace usage',
  subtitle: 'go-plus',
  total_micro_cents: 1775065590,
  total_usd: 17.7506559,
  unit: 'USD',
  period_start: '2026-11-01',
  period_end: '2026-11-30',
  points: [
    { date: '2026-11-01', cost_micro_cents: 412345678, cost_usd: 4.12345678 },
    { date: '2026-11-02', cost_micro_cents: 523456789, cost_usd: 5.23456789 },
    { date: '2026-11-03', cost_micro_cents: 0, cost_usd: 0 },
    { date: '2026-11-04', cost_micro_cents: 450250778, cost_usd: 4.50250778 },
  ],
}

describe('WorkspaceUsageChart', () => {
  it('renders official total and daily bars', () => {
    render(<WorkspaceUsageChart chart={chart} testIdPrefix="drawer" />)

    expect(screen.getByTestId('drawer-chart')).toBeTruthy()
    expect(screen.getByTestId('drawer-total').textContent).toMatch(/17[,.]75/)
    expect(screen.getByText('Workspace usage')).toBeTruthy()
    expect(screen.getByText('Go Plus')).toBeTruthy()
    expect(screen.getByTestId('drawer-bar-2026-11-01').className).toContain('is-active')
    expect(screen.getByTestId('drawer-bar-2026-11-03').className).not.toContain('is-active')
  })

  it('shows day cost on hover', () => {
    render(<WorkspaceUsageChart chart={chart} testIdPrefix="drawer" />)

    fireEvent.mouseEnter(screen.getByTestId('drawer-bar-2026-11-02'))
    const tooltip = screen.getByTestId('drawer-tooltip')
    expect(tooltip.textContent).toMatch(/5[,.]23/)
    expect(tooltip.textContent).toContain('novembro')
    expect(tooltip.className).toContain('is-align-start')
  })

  it('aligns tooltip to the end on the last bars', () => {
    render(<WorkspaceUsageChart chart={chart} testIdPrefix="drawer" />)
    fireEvent.mouseEnter(screen.getByTestId('drawer-bar-2026-11-04'))
    expect(screen.getByTestId('drawer-tooltip').className).toContain('is-align-end')
  })
})

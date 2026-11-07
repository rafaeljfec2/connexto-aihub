import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { AntigravityQuotaOverview } from './AntigravityQuotaOverview'
import type { ProviderQuotaMetric } from '../types/quota'

describe('AntigravityQuotaOverview', () => {
  it('renders categorized pools for Gemini and Claude/GPT models with progress and count', () => {
    const quotas: ProviderQuotaMetric[] = [
      {
        metric: 'claude-sonnet-5-5-medium',
        label: 'Claude Sonnet 5.5 (Medium)',
        nature: 'rate_limit',
        remaining_percent: 46.8,
        period: 'weekly',
        resets_at: '2026-11-11T21:00:00Z',
      },
      {
        metric: 'claude-opus-5-5-low',
        label: 'Claude Opus 5.5 (Low)',
        nature: 'rate_limit',
        remaining_percent: 46.8,
        period: 'weekly',
        resets_at: '2026-11-11T21:00:00Z',
      },
      {
        metric: 'gemini-2.5-pro',
        label: 'Gemini 2.5 Pro',
        nature: 'rate_limit',
        remaining_percent: 73.83,
        period: '5h',
        resets_at: '2026-11-06T21:20:00Z',
      },
      {
        metric: 'tab_jump_flash_lite_preview',
        label: 'tab_jump_flash_lite_preview',
        nature: 'rate_limit',
        remaining_percent: 100,
      },
    ]

    render(<AntigravityQuotaOverview accountId="ag-acc-1" quotas={quotas} />)

    // Families rendered
    expect(screen.getByText('Gemini Models')).toBeTruthy()
    expect(screen.getByText('Claude and GPT models')).toBeTruthy()

    // Shared pools deduplicated with integer rounding matching IDE
    expect(screen.getByText('47% restante')).toBeTruthy()
    expect(screen.getByText('74% restante')).toBeTruthy()

    // Model count badges
    expect(screen.getByText('1 modelo(s)')).toBeTruthy()
    expect(screen.getByText('2 modelo(s)')).toBeTruthy()

    // Autocomplete condensed summary
    expect(screen.getByText(/Autocomplete & Background \(1 modelos\)/)).toBeTruthy()
  })
})

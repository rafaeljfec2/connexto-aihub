import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { AntigravityQuotaDrawerSection } from './AntigravityQuotaDrawerSection'
import type { ProviderQuotaMetric } from '../types/quota'

describe('AntigravityQuotaDrawerSection', () => {
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

  it('renders categorized drawer sections with model count toggle', () => {
    render(<AntigravityQuotaDrawerSection quotas={quotas} />)

    expect(screen.getByText('Cotas e Franquias Antigravity')).toBeTruthy()
    expect(screen.getByText('Gemini Models')).toBeTruthy()
    expect(screen.getByText('Claude and GPT models')).toBeTruthy()

    // Pools with remaining percentage
    expect(screen.getByText('46,8% restante')).toBeTruthy()
    expect(screen.getByText('73,83% restante')).toBeTruthy()

    // Toggle models
    const toggleButton = screen.getAllByRole('button', { name: /1 modelo\(s\) incluído\(s\)/i })[0]
    expect(screen.queryByText(/Modelos atendidos por esta franquia/i)).toBeNull()

    fireEvent.click(toggleButton)
    expect(screen.getByText(/Modelos atendidos por esta franquia/i)).toBeTruthy()
    expect(screen.getByText('gemini-2.5-pro')).toBeTruthy()
  })

  it('supports expanding autocomplete background group', () => {
    render(<AntigravityQuotaDrawerSection quotas={quotas} />)

    const bgButton = screen.getByRole('button', { name: /Autocomplete & Background/i })
    expect(screen.queryByText('tab_jump_flash_lite_preview')).toBeNull()

    fireEvent.click(bgButton)
    expect(screen.getByText('tab_jump_flash_lite_preview')).toBeTruthy()
  })
})

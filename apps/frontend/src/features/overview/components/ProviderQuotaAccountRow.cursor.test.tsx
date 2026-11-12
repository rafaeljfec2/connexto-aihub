import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { ProviderQuotaSnapshot } from '../types/quota'
import { ProviderQuotaAccountRow } from './ProviderQuotaAccountRow'

describe('ProviderQuotaAccountRow Cursor balance and monthly usage', () => {
  it('prefers prepaid_available_credits and shows monthly_additional_usage for Cursor', () => {
    const cursorSnapshot: ProviderQuotaSnapshot = {
      account_id: 'cursor-1',
      account_label: 'Cursor',
      provider_id: 'cursor',
      provider_name: 'Cursor',
      status: 'available',
      scope: 'account',
      quotas: [
        { metric: 'total_usage', label: 'Total account usage', nature: 'cost', used: 0.18, unit: 'USD' },
        { metric: 'total_credits', label: 'Weekly included budget', nature: 'quota', limit: 161.59, unit: 'USD' },
        { metric: 'monthly_additional_usage', label: 'Monthly additional usage', nature: 'cost', used: 9.22, unit: 'USD' },
        { metric: 'prepaid_available_credits', label: 'Prepaid credits available', nature: 'quota', remaining: 0.79, unit: 'USD' },
      ],
      source: {
        type: 'provider_api',
        name: 'cursor_proxy_usage_collect',
        scope: 'account',
        fetched_at: '2026-11-11T23:30:00Z',
      },
    }

    render(<ProviderQuotaAccountRow snapshots={[cursorSnapshot]} />)

    expect(screen.getByTestId('quota-balance-cursor-1').textContent).toContain('US$ 0,79')
    expect(screen.getByTestId('quota-monthly-additional-cursor-1').textContent).toContain('US$ 9,22')
  })

  it('falls back to total_credits - total_usage when prepaid_available_credits is absent (OpenRouter)', () => {
    const openRouterSnapshot: ProviderQuotaSnapshot = {
      account_id: 'openrouter-1',
      account_label: 'OpenRouter',
      provider_id: 'openrouter',
      provider_name: 'OpenRouter',
      status: 'available',
      scope: 'account',
      quotas: [
        { metric: 'total_usage', label: 'Total account usage', nature: 'cost', used: 4.84, unit: 'USD' },
        { metric: 'total_credits', label: 'Total account credits', nature: 'quota', limit: 10, unit: 'USD' },
      ],
      source: {
        type: 'provider_api',
        name: 'openrouter_credits',
        scope: 'account',
        fetched_at: '2026-11-11T23:30:00Z',
      },
    }

    render(<ProviderQuotaAccountRow snapshots={[openRouterSnapshot]} />)

    expect(screen.getByTestId('quota-balance-openrouter-1').textContent).toContain('US$ 5,16')
    expect(screen.queryByTestId('quota-monthly-additional-openrouter-1')).toBeNull()
  })
})


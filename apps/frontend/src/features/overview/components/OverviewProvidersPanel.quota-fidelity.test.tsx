import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import type { ProviderQuotaMetric, ProviderQuotaSnapshot, ProviderQuotaStatus } from '../types/quota'
import { OverviewProvidersPanel } from './OverviewProvidersPanel'

function renderPanel(snapshots: ProviderQuotaSnapshot[], isError = false) {
  render(
    <MemoryRouter>
      <OverviewProvidersPanel snapshots={snapshots} isError={isError} />
    </MemoryRouter>,
  )
}

function snapshot(
  scope: 'api_key' | 'account' | 'subscription',
  quotas: ProviderQuotaMetric[],
  status: ProviderQuotaStatus = 'available',
  accountId = 'openrouter-primary',
): ProviderQuotaSnapshot {
  const providerId = scope === 'subscription' ? 'opencode-go' : 'openrouter'
  const sourceName =
    scope === 'api_key'
      ? 'openrouter_key_api'
      : scope === 'account'
        ? 'openrouter_credits_api'
        : 'opencode_go_usage_api'
  return {
    account_id: accountId,
    account_label: scope === 'subscription' ? 'OpenCode Go' : 'OpenRouter Primary',
    provider_id: providerId,
    provider_name: scope === 'subscription' ? 'OpenCode Go' : 'OpenRouter',
    status,
    scope,
    quotas,
    workspace_name: scope === 'subscription' ? 'go-plus' : undefined,
    source: {
      type: 'provider_api',
      name: sourceName,
      scope,
      fetched_at: '2026-11-04T14:00:00Z',
    },
  }
}

describe('OverviewProvidersPanel quota fidelity', () => {
  it('shows the account summary without the API-key scope block when both scopes exist', () => {
    renderPanel([
      snapshot('api_key', [
        { metric: 'usage', label: 'Total usage', nature: 'cost', used: 0, unit: 'USD' },
        { metric: 'usage_daily', label: 'Daily usage', nature: 'cost', used: 2, unit: 'USD', period: 'daily' },
        { metric: 'byok_usage', label: 'Total BYOK usage', nature: 'cost', used: 0.5, unit: 'USD' },
        {
          metric: 'free_model_daily_requests',
          label: 'Free model daily requests',
          nature: 'quota',
          used: 0,
          limit: 1000,
          remaining: 1000,
          unit: 'requests',
          period: 'daily',
        },
      ]),
      snapshot('account', [
        { metric: 'total_usage', label: 'Total account usage', nature: 'cost', used: 4.84186443, unit: 'USD' },
        { metric: 'total_credits', label: 'Total account credits', nature: 'quota', limit: 10, unit: 'USD' },
      ]),
    ])

    expect(screen.getAllByTestId('provider-summary-openrouter-primary')).toHaveLength(1)
    expect(screen.queryByTestId('quota-scope-openrouter-primary-api_key')).toBeNull()
    expect(screen.queryByTestId('quota-metric-openrouter-primary-api_key-usage')).toBeNull()
    expect(screen.queryByText('Chave API')).toBeNull()
    expect(screen.getByText('Créditos comprados')).toBeTruthy()
    const creditsRow = screen.getByTestId('quota-metric-openrouter-primary-account-total_credits')
    expect(creditsRow.textContent).toContain('US$ 10,00')
    expect(creditsRow.textContent).not.toContain('Total ')
    expect(screen.getByText('Uso total da conta')).toBeTruthy()
    const usageRow = screen.getByTestId('quota-metric-openrouter-primary-account-total_usage')
    expect(usageRow.textContent).toContain('US$ 4,84')
    expect(usageRow.textContent).not.toContain('4,84186443')
    expect(usageRow.textContent).not.toContain('Usado ')
    expect(screen.getByTestId('quota-balance-openrouter-primary').textContent).toContain('Saldo disponível')
    expect(screen.getByTestId('quota-balance-openrouter-primary').textContent).toContain('US$ 5,16')
    expect(screen.queryByTestId('quota-metric-openrouter-primary-api_key-free_model_daily_requests')).toBeNull()
    expect(screen.queryByText('Daily usage')).toBeNull()
    expect(screen.queryByText('Total BYOK usage')).toBeNull()
    expect(screen.queryByText(/%/)).toBeNull()
  })

  it('keeps a missing account credential visible inside the existing account card', () => {
    renderPanel([
      snapshot('api_key', [
        { metric: 'usage', label: 'Key usage', nature: 'cost', used: 2, unit: 'USD' },
      ]),
      snapshot('account', [], 'unconfigured_credential'),
    ])

    expect(screen.getAllByTestId('provider-summary-openrouter-primary')).toHaveLength(1)
    expect(screen.getByText('Management API Key não configurada')).toBeTruthy()
  })

  it('omits available balance when the provider did not return usage', () => {
    renderPanel([
      snapshot('account', [
        { metric: 'total_credits', label: 'Total account credits', nature: 'quota', limit: 10, unit: 'USD' },
      ]),
    ])

    expect(screen.queryByTestId('quota-balance-openrouter-primary')).toBeNull()
  })

  it('keeps an explicit zero account usage and derives balance from that zero', () => {
    renderPanel([
      snapshot('account', [
        { metric: 'total_credits', label: 'Total account credits', nature: 'quota', limit: 10, unit: 'USD' },
        { metric: 'total_usage', label: 'Total account usage', nature: 'cost', used: 0, unit: 'USD' },
      ]),
    ])

    const usageRow = screen.getByTestId('quota-metric-openrouter-primary-account-total_usage')
    expect(usageRow.textContent).toContain('US$ 0,00')
    expect(usageRow.textContent).not.toContain('Usado ')
    expect(screen.getByTestId('quota-balance-openrouter-primary').textContent).toContain('US$ 10,00')
  })

  it('rounds account usage for display while calculating balance from the precise value', () => {
    renderPanel([
      snapshot('account', [
        { metric: 'total_credits', label: 'Total account credits', nature: 'quota', limit: 10, unit: 'USD' },
        { metric: 'total_usage', label: 'Total account usage', nature: 'cost', used: 4.845, unit: 'USD' },
      ]),
    ])

    expect(screen.getByTestId('quota-metric-openrouter-primary-account-total_usage').textContent).toContain('US$ 4,85')
    expect(screen.getByTestId('quota-balance-openrouter-primary').textContent).toContain('US$ 5,16')
  })

  it('does not derive a balance from metrics with incompatible units', () => {
    renderPanel([
      snapshot('account', [
        { metric: 'total_credits', label: 'Total account credits', nature: 'quota', limit: 10, unit: 'USD' },
        { metric: 'total_usage', label: 'Total account usage', nature: 'cost', used: 4, unit: 'EUR' },
      ]),
    ])

    expect(screen.queryByTestId('quota-balance-openrouter-primary')).toBeNull()
  })

  it('derives each account balance from that account snapshot only', () => {
    renderPanel([
      snapshot('account', [
        { metric: 'total_credits', label: 'Credits', nature: 'quota', limit: 10, unit: 'USD' },
        { metric: 'total_usage', label: 'Usage', nature: 'cost', used: 4, unit: 'USD' },
      ], 'available', 'openrouter-a'),
      snapshot('account', [
        { metric: 'total_credits', label: 'Credits', nature: 'quota', limit: 20, unit: 'USD' },
        { metric: 'total_usage', label: 'Usage', nature: 'cost', used: 7, unit: 'USD' },
      ], 'available', 'openrouter-b'),
    ])

    expect(screen.getByTestId('quota-balance-openrouter-a').textContent).toContain('US$ 6,00')
    expect(screen.getByTestId('quota-balance-openrouter-b').textContent).toContain('US$ 13,00')
  })

  it('shows a quota request failure instead of reporting that no accounts exist', () => {
    renderPanel([], true)

    expect(screen.getByRole('alert').textContent).toContain('Falha ao consultar os dados oficiais')
    expect(screen.queryByTestId('quota-empty')).toBeNull()
  })

  it('renders a compact OpenCode Go summary and opens the detail drawer on click', () => {
    renderPanel([
      snapshot(
        'subscription',
        [
          {
            metric: 'go_rolling',
            label: 'Go rolling',
            nature: 'quota',
            consumed_percent: 12,
            period: 'rolling',
            resets_at: '2026-11-04T18:31:46.000Z',
          },
          {
            metric: 'go_weekly',
            label: 'Go weekly',
            nature: 'quota',
            consumed_percent: 36,
            period: 'weekly',
            resets_at: '2026-11-09T00:00:00.000Z',
          },
          {
            metric: 'go_monthly',
            label: 'Go monthly',
            nature: 'quota',
            consumed_percent: 24,
            period: 'monthly',
            resets_at: '2026-12-01T19:00:43.000Z',
          },
        ],
        'available',
        'opencode-go',
      ),
    ])

    expect(screen.getByTestId('provider-summary-opencode-go')).toBeTruthy()
    expect(screen.getByTestId('go-plus-summary-opencode-go').textContent).toContain('Go Plus 36% used')
    expect(screen.queryByTestId('go-plus-overview-opencode-go-meter-go_rolling')).toBeNull()

    fireEvent.click(screen.getByTestId('provider-summary-opencode-go'))

    expect(screen.getByTestId('provider-quota-drawer')).toBeTruthy()
    expect(screen.getByTestId('go-plus-drawer-meter-go_rolling').textContent).toContain('12% used')
    expect(screen.getByTestId('go-plus-drawer-meter-go_weekly').textContent).toContain('36% used')
    expect(screen.getByTestId('go-plus-drawer-meter-go_monthly').textContent).toContain('24% used')
  })
})

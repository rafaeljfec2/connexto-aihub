import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { OverviewProvidersPanel } from './OverviewProvidersPanel'
import type { ProviderQuotaSnapshot } from '../types/quota'

describe('OverviewProvidersPanel', () => {
  it('renders official OpenRouter quota without inventing percentage or progress bar', () => {
    const snapshots: ProviderQuotaSnapshot[] = [
      {
        account_id: 'openrouter-main',
        account_label: 'OpenRouter Principal',
        provider_id: 'openrouter',
        provider_name: 'OpenRouter',
        status: 'available',
        scope: 'api_key',
        quotas: [
          {
            metric: 'usage',
            label: 'Consumo total',
            nature: 'cost',
            used: 25.5,
            unit: 'USD',
          },
          {
            metric: 'limit',
            label: 'Limite da chave',
            nature: 'quota',
            limit: 100.0,
            unit: 'USD',
            period: 'monthly',
          },
          {
            metric: 'limit_remaining',
            label: 'Saldo da chave',
            nature: 'quota',
            remaining: 74.5,
            unit: 'USD',
            period: 'monthly',
          },
        ],
        source: {
          type: 'provider_api',
          name: 'openrouter_key_api',
          scope: 'api_key',
          fetched_at: '2026-11-04T12:00:00Z',
        },
      },
    ]

    render(
      <MemoryRouter>
        <OverviewProvidersPanel snapshots={snapshots} />
      </MemoryRouter>,
    )

    expect(screen.getByText('Consumo de franquias')).toBeTruthy()
    expect(screen.getByText('OpenRouter Principal')).toBeTruthy()
    expect(screen.getByText(/US\$\s*25,50\s*consumido/)).toBeTruthy()
    expect(screen.getByText(/US\$\s*74,50\s*disponível/)).toBeTruthy()
    expect(screen.getByText('OP')).toBeTruthy() // Initials in badge

    // Must NOT contain an artificial percent string
    expect(screen.queryByText(/%\s*disponível/)).toBeNull()
    expect(screen.queryByText(/%\s*consumido/)).toBeNull()
  })

  it('renders multiple accounts for the same provider independently', () => {
    const snapshots: ProviderQuotaSnapshot[] = [
      {
        account_id: 'openrouter-squad-a',
        account_label: 'OpenRouter Squad A',
        provider_id: 'openrouter',
        provider_name: 'OpenRouter',
        status: 'available',
        scope: 'api_key',
        quotas: [
          {
            metric: 'usage',
            label: 'Consumo total',
            nature: 'cost',
            used: 10.0,
            unit: 'USD',
          },
          {
            metric: 'limit_remaining',
            label: 'Saldo da chave',
            nature: 'quota',
            remaining: 90.0,
            unit: 'USD',
          },
        ],
        source: {
          type: 'provider_api',
          name: 'openrouter_key_api',
          scope: 'api_key',
          fetched_at: '2026-11-04T12:00:00Z',
        },
      },
      {
        account_id: 'openrouter-squad-b',
        account_label: 'OpenRouter Squad B',
        provider_id: 'openrouter',
        provider_name: 'OpenRouter',
        status: 'available',
        scope: 'api_key',
        quotas: [
          {
            metric: 'usage',
            label: 'Consumo total',
            nature: 'cost',
            used: 40.0,
            unit: 'USD',
          },
          {
            metric: 'limit_remaining',
            label: 'Saldo da chave',
            nature: 'quota',
            remaining: 10.0,
            unit: 'USD',
          },
        ],
        source: {
          type: 'provider_api',
          name: 'openrouter_key_api',
          scope: 'api_key',
          fetched_at: '2026-11-04T12:00:00Z',
        },
      },
    ]

    render(
      <MemoryRouter>
        <OverviewProvidersPanel snapshots={snapshots} />
      </MemoryRouter>,
    )

    expect(screen.getByText('OpenRouter Squad A')).toBeTruthy()
    expect(screen.getByText('OpenRouter Squad B')).toBeTruthy()
    expect(screen.getByText(/US\$\s*90,00\s*disponível/)).toBeTruthy()
    expect(screen.getByText(/US\$\s*10,00\s*disponível/)).toBeTruthy()
  })

  it('renders unsupported provider as Sem leitura without inventing values', () => {
    const snapshots: ProviderQuotaSnapshot[] = [
      {
        account_id: 'anthropic-prod',
        account_label: 'Anthropic Produção',
        provider_id: 'anthropic',
        provider_name: 'Anthropic',
        status: 'unsupported',
        scope: 'unknown',
        quotas: [],
        source: {
          type: 'provider_api',
          name: 'anthropic',
          scope: 'unknown',
          fetched_at: '2026-11-04T12:00:00Z',
        },
      },
    ]

    render(
      <MemoryRouter>
        <OverviewProvidersPanel snapshots={snapshots} />
      </MemoryRouter>,
    )

    expect(screen.getByText('Anthropic Produção')).toBeTruthy()
    expect(screen.getByText('Franquia ainda não monitorada')).toBeTruthy()
    expect(screen.getByText('Sem leitura')).toBeTruthy()
  })

  it('renders unconfigured credential state clearly', () => {
    const snapshots: ProviderQuotaSnapshot[] = [
      {
        account_id: 'openai-dev',
        account_label: 'OpenAI Dev',
        provider_id: 'openai',
        provider_name: 'OpenAI',
        status: 'unconfigured_credential',
        scope: 'unknown',
        quotas: [],
        source: {
          type: 'provider_api',
          name: 'openai',
          scope: 'unknown',
          fetched_at: '2026-11-04T12:00:00Z',
        },
      },
    ]

    render(
      <MemoryRouter>
        <OverviewProvidersPanel snapshots={snapshots} />
      </MemoryRouter>,
    )

    expect(screen.getByText('OpenAI Dev')).toBeTruthy()
    expect(screen.getByText('Credencial não configurada')).toBeTruthy()
    expect(screen.getByText('Sem leitura')).toBeTruthy()
  })

  it('renders official percent and progress bar only when explicitly returned by provider', () => {
    const snapshots: ProviderQuotaSnapshot[] = [
      {
        account_id: 'official-percent-provider',
        account_label: 'Provedor com Percentual',
        provider_id: 'custom',
        provider_name: 'Custom',
        status: 'available',
        scope: 'account',
        quotas: [
          {
            metric: 'usage_percent',
            label: 'Consumo percentual',
            nature: 'usage',
            consumed_percent: 68,
            remaining_percent: 32,
          },
        ],
        source: {
          type: 'provider_api',
          name: 'custom_api',
          scope: 'account',
          fetched_at: '2026-11-04T12:00:00Z',
        },
      },
    ]

    render(
      <MemoryRouter>
        <OverviewProvidersPanel snapshots={snapshots} />
      </MemoryRouter>,
    )

    expect(screen.getByText('32% disponível')).toBeTruthy()
  })

  it('renders loading state when isLoading is true', () => {
    render(
      <MemoryRouter>
        <OverviewProvidersPanel isLoading={true} />
      </MemoryRouter>,
    )

    expect(screen.getByTestId('quota-loading')).toBeTruthy()
  })

  it('renders empty message when no accounts are configured', () => {
    render(
      <MemoryRouter>
        <OverviewProvidersPanel snapshots={[]} />
      </MemoryRouter>,
    )

    expect(screen.getByTestId('quota-empty')).toBeTruthy()
    expect(screen.getByText('Nenhuma conta de provedor configurada no momento.')).toBeTruthy()
  })

  it('navigates to /providers via Ver franquias button', () => {
    render(
      <MemoryRouter>
        <OverviewProvidersPanel snapshots={[]} />
      </MemoryRouter>,
    )

    const link = screen.getByRole('link', { name: /Ver franquias/ })
    expect(link.getAttribute('href')).toBe('/providers')
  })
})

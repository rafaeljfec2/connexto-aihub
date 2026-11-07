import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { ProviderDetailPanel } from './ProviderDetailPanel'
import type { AdminProvider } from '../types'
import type { ProviderQuotaSnapshot } from '../../overview/types/quota'

describe('ProviderDetailPanel', () => {
  const onOpenManage = vi.fn()

  it('renders placeholder when no provider is selected', () => {
    render(
      <MemoryRouter>
        <ProviderDetailPanel provider={null} onOpenManage={onOpenManage} />
      </MemoryRouter>,
    )
    expect(screen.getByText('SELECIONE UM PROVEDOR')).toBeTruthy()
    expect(
      screen.getByText('Selecione um provedor na tabela para inspecionar a conexão.'),
    ).toBeTruthy()
  })

  it('renders Antigravity with quota percentage, family pools and primary reading', () => {
    const agProvider: AdminProvider = {
      id: 'antigravity-1',
      name: 'Antigravity',
      type: 'antigravity',
      endpoint: 'http://localhost:9090',
      enabled: true,
      credential_configured: true,
      created_at: '2026-11-06T19:00:00Z',
      updated_at: '2026-11-06T20:00:00Z',
    }

    const snapshot: ProviderQuotaSnapshot = {
      account_id: 'antigravity-1',
      account_label: 'talita.ponciano@gmail.com',
      provider_id: 'antigravity',
      provider_name: 'Antigravity',
      status: 'available',
      scope: 'account',
      quotas: [
        {
          metric: 'claude-sonnet-5-5-medium',
          label: 'Claude Sonnet 5.5 (Medium)',
          nature: 'rate_limit',
          remaining_percent: 46.8,
          period: 'weekly',
          resets_at: '2026-11-11T22:05:00Z',
        },
        {
          metric: 'gemini-2.5-pro',
          label: 'Gemini 2.5 Pro',
          nature: 'rate_limit',
          remaining_percent: 68.83,
          period: '5h',
          resets_at: '2026-11-06T22:05:00Z',
        },
      ],
      source: {
        type: 'provider_api',
        name: 'antigravity_proxy_usage_api',
        scope: 'account',
        fetched_at: '2026-11-06T20:03:00Z',
      },
    }

    render(
      <MemoryRouter>
        <ProviderDetailPanel provider={agProvider} snapshot={snapshot} onOpenManage={onOpenManage} />
      </MemoryRouter>,
    )

    // Header & provider name
    expect(screen.getByText('Antigravity')).toBeTruthy()

    // Quota box main percentage
    expect(screen.getByText('46,8%')).toBeTruthy()
    expect(screen.getByText('restante')).toBeTruthy()

    // Families block
    expect(screen.getByText('Franquias por Família')).toBeTruthy()
    expect(screen.getByText('Claude and GPT models')).toBeTruthy()
    expect(screen.getByText('Gemini Models')).toBeTruthy()

    // Primary reading
    expect(screen.getByText(/46,8% \(Claude\/GPT \(Limite semanal\)\)/)).toBeTruthy()

    // Source name mapped
    expect(screen.getByText('Antigravity Proxy Usage API')).toBeTruthy()
  })
})

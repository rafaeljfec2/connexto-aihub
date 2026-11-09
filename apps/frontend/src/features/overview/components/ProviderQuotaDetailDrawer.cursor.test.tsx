import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as cursorQueries from '../../cursor/api/queries'
import type { CursorUsageAnalyticsEnvelope } from '../../cursor/types'
import type { ProviderQuotaSnapshot } from '../types/quota'
import { ProviderQuotaDetailDrawer } from './ProviderQuotaDetailDrawer'

vi.mock('../../cursor/api/queries', async () => {
  const actual = await vi.importActual<typeof import('../../cursor/api/queries')>(
    '../../cursor/api/queries',
  )
  return { ...actual, getCursorUsageAnalytics: vi.fn(), getCursorUsageAnalyticsForce: vi.fn() }
})

const cursorSnapshot: ProviderQuotaSnapshot = {
  account_id: 'cursor-secondary',
  account_label: 'Cursor',
  provider_id: 'cursor',
  provider_name: 'Cursor',
  status: 'available',
  scope: 'account',
  quotas: [],
  source: { type: 'provider_api', name: 'cursor_proxy_usage_collect', scope: 'account', fetched_at: '2026-11-08T10:00:00Z' },
}

const analytics: CursorUsageAnalyticsEnvelope = {
  providerId: 'cursor-secondary',
  providerName: 'Cursor',
  status: 'available',
  fetchedAt: '2026-11-08T10:00:00Z',
  sourceCollectedAt: '2026-11-08T09:55:00Z',
  analytics: {
    dailyMetrics: [
      { date: '1794096000000', agentRequests: 8, linesAdded: 42, totalApplies: 3, totalAccepts: 2, subscriptionIncludedReqs: 7, usageBasedReqs: 1, modelUsage: [{ name: 'gpt-5.1-codex-max', count: 8 }] },
    ],
  },
}

const clients: QueryClient[] = []

function renderDrawer() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  clients.push(client)
  vi.mocked(cursorQueries.getCursorUsageAnalytics).mockResolvedValue(analytics)
  return render(
    <QueryClientProvider client={client}>
      <ProviderQuotaDetailDrawer
        open
        onOpenChange={vi.fn()}
        accountLabel="Cursor"
        snapshots={[cursorSnapshot]}
      />
    </QueryClientProvider>,
  )
}

afterEach(() => {
  clients.splice(0).forEach((client) => client.clear())
  vi.clearAllMocks()
})

describe('ProviderQuotaDetailDrawer Cursor usage section', () => {
  it('shows Grok console activity inside the existing provider drawer', async () => {
    renderDrawer()

    expect(await screen.findByTestId('cursor-drawer-usage')).toBeTruthy()
    expect(cursorQueries.getCursorUsageAnalytics).toHaveBeenCalledWith('cursor-secondary')
    expect(screen.getByText('ATIVIDADE CURSOR · CONSOLE GROK')).toBeTruthy()
    expect(screen.getByText('gpt-5.1-codex-max')).toBeTruthy()
    expect(screen.getByText('42')).toBeTruthy()
    expect(screen.queryByText('Sem leitura oficial de franquia Go para esta conta.')).toBeNull()
  })

  it('keeps the displayed usage and warns when physical refresh fails', async () => {
    renderDrawer()
    expect(await screen.findByTestId('cursor-drawer-usage')).toBeTruthy()
    vi.mocked(cursorQueries.getCursorUsageAnalyticsForce).mockRejectedValue(new Error('proxy offline'))

    fireEvent.click(screen.getByRole('button', { name: 'Atualizar uso do Cursor' }))

    expect(await screen.findByTestId('cursor-drawer-refresh-warning')).toBeTruthy()
    expect(screen.getByText('gpt-5.1-codex-max')).toBeTruthy()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Atualizar uso do Cursor' }).hasAttribute('disabled')).toBe(false))
  })
})

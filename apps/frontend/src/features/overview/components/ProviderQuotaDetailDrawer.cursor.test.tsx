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

  it('renders Cursor weekly product breakdown and monthly additional usage in the drawer', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    clients.push(client)
    vi.mocked(cursorQueries.getCursorUsageAnalytics).mockResolvedValue(analytics)

    const enrichedSnapshot: ProviderQuotaSnapshot = {
      ...cursorSnapshot,
      quotas: [
        { metric: 'weekly_included_status', label: 'Weekly included status', nature: 'usage', scope: 'weekly_status:metered' },
        {
          metric: 'weekly_product_cursor',
          label: 'Weekly usage · Cursor',
          nature: 'usage',
          used: 0.18,
          consumed_percent: 0.11139,
          unit: 'USD',
          scope: 'product:cursor',
        },
        {
          metric: 'monthly_additional_usage',
          label: 'Monthly additional usage',
          nature: 'cost',
          used: 9.22,
          unit: 'USD',
          resets_at: '2026-12-01T00:00:00.000Z',
        },
        { metric: 'prepaid_available_credits', label: 'Prepaid credits available', nature: 'quota', remaining: 0.79, unit: 'USD' },
      ],
    }

    render(
      <QueryClientProvider client={client}>
        <ProviderQuotaDetailDrawer
          open
          onOpenChange={vi.fn()}
          accountLabel="Cursor"
          snapshots={[enrichedSnapshot]}
        />
      </QueryClientProvider>,
    )

    const breakdown = await screen.findByTestId('cursor-drawer-quota-summary')
    expect(breakdown.textContent).toContain('metered')
    expect(breakdown.textContent).toContain('Cursor (semanal)')
    expect(breakdown.textContent).toContain('US$ 9,22')
    expect(breakdown.textContent).toContain('Sem limite definido')
    expect(breakdown.textContent).toContain('US$ 0,79')
  })
})

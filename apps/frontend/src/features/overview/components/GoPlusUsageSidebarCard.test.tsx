import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ProviderQuotasResponse } from '../types/quota'

const { getProviderQuotasMock } = vi.hoisted(() => ({
  getProviderQuotasMock: vi.fn<() => Promise<ProviderQuotasResponse>>(),
}))

vi.mock('../api/queries', async () => {
  const actual = await vi.importActual<typeof import('../api/queries')>('../api/queries')
  return {
    ...actual,
    getProviderQuotas: getProviderQuotasMock,
  }
})

import { GoPlusUsageSidebarCard } from './GoPlusUsageSidebarCard'

function renderWithClient(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>)
}

const availableResponse: ProviderQuotasResponse = {
  fetched_at: '2026-11-04T16:00:00Z',
  snapshots: [
    {
      account_id: 'opencode-go',
      account_label: 'OpenCode Go Primary',
      provider_id: 'opencode-go',
      provider_name: 'OpenCode Go',
      status: 'available',
      scope: 'subscription',
      workspace_name: 'go-plus',
      quotas: [
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
      source: {
        type: 'provider_api',
        name: 'opencode_go_usage_api',
        scope: 'subscription',
        fetched_at: '2026-11-04T16:00:00Z',
      },
    },
  ],
}

describe('GoPlusUsageSidebarCard', () => {
  beforeEach(() => {
    getProviderQuotasMock.mockReset()
  })

  it('renders a compact Go Plus summary when available', async () => {
    getProviderQuotasMock.mockResolvedValue(availableResponse)
    renderWithClient(<GoPlusUsageSidebarCard collapsed={false} />)

    await waitFor(() => {
      expect(screen.getByTestId('go-plus-sidebar-summary')).toBeTruthy()
    })
    expect(screen.getByText('Go Plus')).toBeTruthy()
    expect(screen.getByText('36% used')).toBeTruthy()
  })

  it('shows a weekly percent chip when the sidebar is collapsed', async () => {
    getProviderQuotasMock.mockResolvedValue(availableResponse)
    renderWithClient(<GoPlusUsageSidebarCard collapsed={true} />)

    await waitFor(() => {
      expect(screen.getByText('36%')).toBeTruthy()
    })
    expect(screen.getByTestId('go-plus-sidebar-collapsed')).toBeTruthy()
  })

  it('shows sem leitura when no OpenCode snapshot is available', async () => {
    getProviderQuotasMock.mockResolvedValue({ fetched_at: '2026-11-04T16:00:00Z', snapshots: [] })
    renderWithClient(<GoPlusUsageSidebarCard collapsed={false} />)

    expect(await screen.findByText('Go: sem leitura')).toBeTruthy()
  })
})

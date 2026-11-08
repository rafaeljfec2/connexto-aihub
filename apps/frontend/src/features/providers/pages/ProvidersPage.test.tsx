import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as queries from '../api/queries'
import type { ProviderQuotaSnapshot, ProviderQuotasResponse } from '../../overview/types/quota'
import type { AdminProvider, ProviderListResponse } from '../types'
import { ProvidersPage } from './ProvidersPage'

vi.mock('../api/queries', async () => {
  const actual = await vi.importActual<typeof import('../api/queries')>('../api/queries')
  return {
    ...actual,
    getProviderQuotas: vi.fn(),
    getProviderQuotasForce: vi.fn(),
    listProviders: vi.fn(),
  }
})

const cursorProvider: AdminProvider = {
  id: 'cursor',
  name: 'Cursor',
  type: 'cursor',
  endpoint: 'http://127.0.0.1:9090/v1',
  enabled: true,
  credential_configured: true,
  created_at: '2026-11-04T00:00:00Z',
  updated_at: '2026-11-04T01:00:00Z',
}

const openRouterProvider: AdminProvider = {
  id: 'openrouter-secondary',
  name: 'OpenRouter Secondary',
  type: 'openrouter',
  endpoint: 'https://openrouter.ai/api/v1',
  enabled: true,
  credential_configured: true,
  created_at: '2026-11-04T00:00:00Z',
  updated_at: '2026-11-04T01:00:00Z',
}

const providerList: ProviderListResponse = {
  providers: [cursorProvider, openRouterProvider],
}

const cursorQuota: ProviderQuotaSnapshot = {
  account_id: 'cursor',
  account_label: 'Cursor',
  provider_id: 'cursor',
  provider_name: 'Cursor',
  status: 'available',
  scope: 'account',
  quotas: [
    {
      metric: 'weekly_used_percent',
      label: 'Weekly usage',
      nature: 'rate_limit',
      consumed_percent: 25,
      period: 'weekly',
    },
  ],
  source: {
    type: 'provider_api',
    name: 'cursor_usage_api',
    scope: 'account',
    fetched_at: '2026-11-04T01:00:00Z',
  },
}

const openRouterQuota: ProviderQuotaSnapshot = {
  account_id: 'openrouter-secondary',
  account_label: 'OpenRouter Secondary',
  provider_id: 'openrouter',
  provider_name: 'OpenRouter',
  status: 'available',
  scope: 'api_key',
  quotas: [
    {
      metric: 'key_usage',
      label: 'Key usage',
      nature: 'cost',
      used: 2.5,
      limit: 10,
      remaining: 7.5,
      unit: 'USD',
    },
  ],
  source: {
    type: 'provider_api',
    name: 'openrouter_key_api',
    scope: 'api_key',
    fetched_at: '2026-11-04T01:00:00Z',
  },
}

const quotaResponse: ProviderQuotasResponse = {
  snapshots: [cursorQuota, openRouterQuota],
  fetched_at: '2026-11-04T01:00:00Z',
}

const queryClients: QueryClient[] = []

function createQueryClient(): QueryClient {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, retryDelay: 0, gcTime: 0, staleTime: Infinity },
    },
  })
  queryClients.push(queryClient)
  return queryClient
}

function renderProvidersPage(queryClient = createQueryClient()) {
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/providers']}>
        <ProvidersPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function resolveProviderQueries(
  providers: ProviderListResponse = providerList,
  quotas: ProviderQuotasResponse = quotaResponse,
): void {
  vi.mocked(queries.getProviderQuotasForce).mockResolvedValue({
    snapshots: quotas.snapshots,
    fetched_at: quotas.fetched_at,
  })
  vi.mocked(queries.listProviders).mockResolvedValue(providers)
  vi.mocked(queries.getProviderQuotas).mockResolvedValue(quotas)
}

function createDeferred<T>() {
  let resolvePromise: (value: T) => void = () => undefined
  let rejectPromise: (reason: unknown) => void = () => undefined
  const promise = new Promise<T>((resolve, reject) => {
    resolvePromise = resolve
    rejectPromise = reject
  })

  return {
    promise,
    resolve: (value: T) => resolvePromise(value),
    reject: (reason: unknown) => rejectPromise(reason),
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

afterEach(() => {
  queryClients.splice(0).forEach((queryClient) => queryClient.clear())
})

describe('ProvidersPage loading states', () => {
  it('shows the page-shaped skeleton before the first responses arrive', () => {
    const providers = createDeferred<ProviderListResponse>()
    const quotas = createDeferred<ProviderQuotasResponse>()
    vi.mocked(queries.listProviders).mockReturnValue(providers.promise)
    vi.mocked(queries.getProviderQuotas).mockReturnValue(quotas.promise)

    renderProvidersPage()

    expect(screen.getByTestId('provider-quotas-page-skeleton')).toBeTruthy()
    expect(screen.getByTestId('provider-quotas-skeleton-status').textContent).toContain(
      'Carregando franquias',
    )
    expect(screen.getAllByTestId('provider-quota-summary-card-skeleton')).toHaveLength(4)
    expect(screen.getAllByTestId('provider-quota-skeleton-row')).toHaveLength(4)
    expect(screen.getByTestId('provider-quota-details-skeleton')).toBeTruthy()
    expect(screen.getByTestId('provider-quota-footer-skeleton')).toBeTruthy()
    expect(screen.getByTestId('providers-page-layout')).toBeTruthy()
    expect(screen.getByTestId('providers-page-footer')).toBeTruthy()
    expect(
      screen.getByRole('button', { name: 'Atualizar franquias' }).hasAttribute('disabled'),
    ).toBe(true)
    expect(screen.queryByText('00')).toBeNull()
  })

  it('replaces the skeleton with data and keeps provider details selectable', async () => {
    resolveProviderQueries()
    renderProvidersPage()

    expect(await screen.findByTestId('providers-quota-row-cursor')).toBeTruthy()
    expect(screen.queryByTestId('provider-quotas-page-skeleton')).toBeNull()

    fireEvent.click(screen.getByTestId('providers-quota-row-openrouter-secondary'))

    expect(screen.getByTestId('providers-detail-quota-openrouter-secondary')).toBeTruthy()
  })

  it('keeps cached data visible and marks the refresh button while background refetching', async () => {
    const queryClient = createQueryClient()
    queryClient.setQueryData(queries.providersQueryKeys.list, providerList)
    queryClient.setQueryData(queries.providersQueryKeys.quotas, quotaResponse)
    const nextProviders = createDeferred<ProviderListResponse>()
    const nextQuotas = createDeferred<ProviderQuotasResponse>()
    vi.mocked(queries.listProviders).mockReturnValue(nextProviders.promise)
    vi.mocked(queries.getProviderQuotas).mockReturnValue(nextQuotas.promise)
    vi.mocked(queries.getProviderQuotasForce).mockResolvedValue(quotaResponse)

    renderProvidersPage(queryClient)
    expect(screen.getByTestId('providers-quota-row-cursor')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: 'Atualizar franquias' }))

    expect(screen.getByTestId('providers-quota-row-cursor')).toBeTruthy()
    expect(screen.queryByTestId('provider-quotas-page-skeleton')).toBeNull()
    expect(screen.getByTestId('providers-page-layout')).toBeTruthy()
    expect(screen.getByTestId('providers-page-footer')).toBeTruthy()
    expect(await screen.findByTestId('providers-refreshing-status')).toBeTruthy()
    expect(
      screen.getByRole('button', { name: 'Atualizar franquias' }).hasAttribute('disabled'),
    ).toBe(true)

    await act(async () => {
      nextProviders.resolve(providerList)
      nextQuotas.resolve(quotaResponse)
    })

    expect(screen.getByTestId('providers-quota-row-cursor')).toBeTruthy()
    await waitFor(() => expect(screen.queryByTestId('providers-refreshing-status')).toBeNull())
  })

  it('preserves cached data and shows a warning when a background refetch fails', async () => {
    const queryClient = createQueryClient()
    queryClient.setQueryData(queries.providersQueryKeys.list, providerList)
    queryClient.setQueryData(queries.providersQueryKeys.quotas, quotaResponse)
    vi.mocked(queries.getProviderQuotasForce).mockResolvedValue(quotaResponse)
    vi.mocked(queries.listProviders).mockRejectedValue(new Error('list refetch failed'))
    vi.mocked(queries.getProviderQuotas).mockRejectedValue(new Error('quota refetch failed'))
    renderProvidersPage(queryClient)

    fireEvent.click(screen.getByRole('button', { name: 'Atualizar franquias' }))

    expect(await screen.findByTestId('providers-refetch-warning')).toBeTruthy()
    expect(screen.getByTestId('providers-quota-row-cursor')).toBeTruthy()
    expect(screen.getByTestId('providers-detail-quota-cursor')).toBeTruthy()
    expect(screen.queryByTestId('provider-quotas-page-skeleton')).toBeNull()
  })

  it('shows a retryable error state when the initial requests fail', async () => {
    vi.mocked(queries.listProviders)
      .mockRejectedValueOnce(new Error('providers unavailable'))
      .mockRejectedValueOnce(new Error('providers unavailable'))
      .mockResolvedValue(providerList)
    vi.mocked(queries.getProviderQuotas)
      .mockRejectedValueOnce(new Error('quotas unavailable'))
      .mockRejectedValueOnce(new Error('quotas unavailable'))
      .mockResolvedValue(quotaResponse)
    renderProvidersPage()

    expect(await screen.findByTestId('providers-initial-error')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Tentar novamente' }))
    expect(await screen.findByTestId('providers-quota-row-cursor')).toBeTruthy()
    expect(screen.queryByTestId('providers-initial-error')).toBeNull()
  })

  it('shows an empty state when the provider list response is valid and empty', async () => {
    resolveProviderQueries({ providers: [] }, { snapshots: [], fetched_at: '2026-11-04T01:00:00Z' })
    renderProvidersPage()

    expect(await screen.findByText('Nenhum provedor configurado.')).toBeTruthy()
    expect(screen.getByText('Nenhum provedor configurado para inspecionar.')).toBeTruthy()
  })

  it('selects the first provider when a previously empty cached list becomes populated', async () => {
    resolveProviderQueries({ providers: [] }, { snapshots: [], fetched_at: '2026-11-04T01:00:00Z' })
    renderProvidersPage()

    expect(await screen.findByText('Nenhum provedor configurado para inspecionar.')).toBeTruthy()
    resolveProviderQueries()
    fireEvent.click(screen.getByRole('button', { name: 'Atualizar franquias' }))

    expect(await screen.findByTestId('providers-detail-quota-cursor')).toBeTruthy()
  })

  it('forces a physical quota collection and keeps the cache visible while polling', async () => {
    const queryClient = createQueryClient()
    queryClient.setQueryData(queries.providersQueryKeys.list, providerList)
    queryClient.setQueryData(queries.providersQueryKeys.quotas, quotaResponse)
    vi.mocked(queries.listProviders).mockResolvedValue(providerList)
    const refreshingResponse: ProviderQuotasResponse = {
      snapshots: quotaResponse.snapshots.map((snapshot) => ({ ...snapshot, refreshing: true })),
      fetched_at: '2026-11-04T01:00:00Z',
    }
    const settledResponse: ProviderQuotasResponse = {
      snapshots: quotaResponse.snapshots.map((snapshot) => ({ ...snapshot, refreshing: false })),
      fetched_at: '2026-11-04T01:05:00Z',
    }
    vi.mocked(queries.getProviderQuotasForce).mockResolvedValue(refreshingResponse)
    vi.mocked(queries.getProviderQuotas).mockResolvedValue(settledResponse)

    renderProvidersPage(queryClient)
    await screen.findByTestId('providers-quota-row-cursor')

    fireEvent.click(screen.getByRole('button', { name: 'Atualizar franquias' }))

    expect(queries.getProviderQuotasForce).toHaveBeenCalledTimes(1)
    expect(screen.getByTestId('providers-quota-row-cursor')).toBeTruthy()
    expect(screen.queryByTestId('provider-quotas-page-skeleton')).toBeNull()
    expect(await screen.findByTestId('providers-refreshing-status')).toBeTruthy()

    await waitFor(
      () => expect(screen.queryByTestId('providers-refreshing-status')).toBeNull(),
      { timeout: 4000 },
    )
    expect(screen.getByTestId('providers-quota-row-cursor')).toBeTruthy()
    expect(queries.getProviderQuotasForce).toHaveBeenCalledTimes(1)
  })
})

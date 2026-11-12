import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ExecutionsPage } from './ExecutionsPage'

const executionId = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'

const generation = {
  execution_id: executionId,
  request_id: 'req-test-123',
  provider_id: 'opencode-go',
  model_id: 'vendor/model',
  started_at: '2026-11-03T12:00:00Z',
  completed_at: '2026-11-03T12:00:00.025Z',
  duration_ns: 25_000_000,
  outcome: 'completed',
  phase: 'forwarding',
  cause: null,
  upstream_status: 200,
  usage: {
    status: 'reported',
    source: 'provider_opencode_go',
    input_tokens: 0,
    output_tokens: 7,
    provider_reported_cost: '0.000000000000',
    provider_list_cost: '0.000016800000',
    currency: 'USD',
  },
  telemetry: {
    streaming: true,
    provider_latency_ns: 100_000_000,
    routing_overhead_ns: null,
    time_to_first_token_ns: null,
    output_stream_duration_ns: null,
    finish_reason: null,
    client_app: 'codex_cli_rs',
  },
  provider_responses: [],
}

function generationResponse(generations = [generation]) {
  return {
    generations,
    count: generations.length,
    next_cursor: null,
    degraded: false,
  }
}

function generationListPaths(paths: string[]): string[] {
  return paths.filter((path) => path.includes('/v1/generations'))
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  return render(
    <MemoryRouter initialEntries={['/executions']}>
      <QueryClientProvider client={queryClient}>
        <ExecutionsPage />
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe('ExecutionsPage generations experience', () => {
  it('loads the generation projection and renders the HTML-led view', async () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify(generationResponse())))
    vi.stubGlobal('fetch', fetchMock)

    renderPage()

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/v1/generations'),
        expect.objectContaining({ credentials: 'same-origin' }),
      )
    })
    expect(await screen.findByRole('tab', { name: 'Generations' })).toBeTruthy()
    expect(await screen.findByText('vendor/model')).toBeTruthy()
    expect(await screen.findByText('codex_cli_rs')).toBeTruthy()
  })

  it('starts without a selected generation or an open detail drawer', async () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify(generationResponse())))
    vi.stubGlobal('fetch', fetchMock)

    renderPage()

    const row = await screen.findByRole('row', { name: /vendor\/model/ })
    expect(row.getAttribute('aria-selected')).toBe('false')
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('offers catalog providers that are not present in the loaded generations', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      if (String(input).startsWith('/v1/catalog/providers')) {
        return new Response(JSON.stringify({
          object: 'list',
          data: [{ provider: 'catalog-only-provider' }, { provider: 'opencode-go' }],
        }))
      }
      return new Response(JSON.stringify(generationResponse()))
    })
    vi.stubGlobal('fetch', fetchMock)

    renderPage()

    const providerFilter = await screen.findByLabelText('Filtrar por provedor')
    await waitFor(() => {
      expect(within(providerFilter).getByRole('option', { name: 'catalog-only-provider' })).toBeTruthy()
    })
  })

  it('labels partial enrichment without claiming it is unavailable', async () => {
    const partialGeneration = {
      ...generation,
      enrichment_status: 'partial',
      usage: { ...generation.usage, status: 'partial' },
    }
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({
      ...generationResponse(),
      generations: [partialGeneration],
    })))
    vi.stubGlobal('fetch', fetchMock)

    renderPage()

    expect(await screen.findByText('ENRICHMENT PARTIAL')).toBeTruthy()
  })

  it('opens the selected generation with keyboard input and closes it with Escape', async () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify(generationResponse())))
    vi.stubGlobal('fetch', fetchMock)

    renderPage()

    const row = await screen.findByRole('row', { name: /vendor\/model/ })
    row.focus()
    fireEvent.keyDown(row, { key: 'Enter' })

    expect(await screen.findByRole('dialog')).toBeTruthy()
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' })
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(row.getAttribute('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(row)
  })

  it('switches generations while the overlay drawer stays open', async () => {
    const secondGeneration = {
      ...generation,
      execution_id: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
      request_id: 'req-second-123',
      model_id: 'vendor/second-model',
    }
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input)
      const selected = path.endsWith(secondGeneration.execution_id) ? secondGeneration : generation
      const response = path.includes('/v1/generations/')
        ? { ...selected, provider_session_id: null, provider_responses: [], degraded: false }
        : generationResponse([generation, secondGeneration])
      return new Response(JSON.stringify(response))
    })
    vi.stubGlobal('fetch', fetchMock)

    renderPage()

    fireEvent.click(await screen.findByRole('row', { name: /vendor\/model/ }))
    expect(await screen.findByRole('dialog')).toBeTruthy()
    fireEvent.click(screen.getByRole('row', { name: /vendor\/second-model/ }))

    expect(await screen.findByRole('heading', { name: 'vendor/second-model' })).toBeTruthy()
    expect(screen.getByRole('dialog')).toBeTruthy()
  })

  it('maps applied filters to the API and searches only loaded generations', async () => {
    const requestedPaths: string[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      requestedPaths.push(String(input))
      return new Response(JSON.stringify(generationResponse()))
    })
    vi.stubGlobal('fetch', fetchMock)

    renderPage()

    expect(await screen.findByRole('row', { name: /vendor\/model/ })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Concluídas' }))
    await waitFor(() => {
      expect(generationListPaths(requestedPaths).some((path) => path.includes('outcome=completed'))).toBe(true)
    })

    const providerSelect = await screen.findByLabelText('Filtrar por provedor')
    await waitFor(() => {
      expect(within(providerSelect).getByRole('option', { name: 'opencode-go' })).toBeTruthy()
    })
    fireEvent.change(providerSelect, { target: { value: 'opencode-go' } })
    await waitFor(() => {
      expect(generationListPaths(requestedPaths).some((path) => path.includes('provider_id=opencode-go'))).toBe(true)
    })

    fireEvent.change(screen.getByLabelText('Período'), { target: { value: 'custom' } })
    fireEvent.change(screen.getByLabelText('Filtrar por ID do modelo'), { target: { value: 'vendor/model' } })
    fireEvent.change(screen.getByLabelText('Data inicial'), { target: { value: '2026-11-03' } })
    fireEvent.change(screen.getByLabelText('Data final'), { target: { value: '2026-11-04' } })
    fireEvent.click(screen.getByRole('button', { name: 'Aplicar filtros' }))

    await waitFor(() => {
      expect(generationListPaths(requestedPaths).some((path) => path.includes('provider_id=opencode-go') &&
        path.includes('model_id=vendor%2Fmodel') && path.includes('outcome=completed') &&
        path.includes('started_after=2026-11-03T00%3A00%3A00.000Z') &&
        path.includes('started_before=2026-11-04T23%3A59%3A59.999Z'))).toBe(true)
    })

    const row = await screen.findByRole('row', { name: /vendor\/model/ })
    fireEvent.change(screen.getByLabelText('Buscar gerações carregadas'), { target: { value: 'req-test-123' } })
    expect(within(row).getByText('0')).toBeTruthy()
    fireEvent.change(screen.getByLabelText('Buscar gerações carregadas'), { target: { value: 'missing-value' } })
    expect(screen.getByText('No generations match the current filters.')).toBeTruthy()
  })

  it('loads the next cursor page and refreshes explicitly', async () => {
    const secondGeneration = {
      ...generation,
      execution_id: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
      request_id: 'req-second-123',
      model_id: 'vendor/second-model',
    }
    const requestedPaths: string[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input)
      requestedPaths.push(path)
      const response = path.includes('cursor=cursor-2')
        ? { generations: [secondGeneration], count: 1, next_cursor: null, degraded: false }
        : { ...generationResponse(), next_cursor: 'cursor-2' }
      return new Response(JSON.stringify(response))
    })
    vi.stubGlobal('fetch', fetchMock)

    renderPage()

    await screen.findByRole('row', { name: /vendor\/model/ })
    fireEvent.click(screen.getByRole('button', { name: 'Load more' }))
    expect(await screen.findByRole('row', { name: /vendor\/second-model/ })).toBeTruthy()
    expect(requestedPaths.some((path) => path.includes('cursor=cursor-2'))).toBe(true)

    const callsBeforeRefresh = requestedPaths.length
    fireEvent.click(screen.getByRole('button', { name: 'Atualizar' }))
    await waitFor(() => expect(requestedPaths.length).toBeGreaterThan(callsBeforeRefresh))
  })
})

import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AddProviderModal } from './AddProviderModal'
import * as queries from '../api/queries'
import type { AdminProvider, AvailableProvider } from '../types'

vi.mock('../api/queries', async () => {
  const actual = await vi.importActual<typeof queries>('../api/queries')
  return {
    ...actual,
    listAvailableProviders: vi.fn(),
    validateProvider: vi.fn(),
    createProvider: vi.fn(),
    updateProvider: vi.fn(),
    syncProviderModels: vi.fn(),
  }
})

const mockAvailable: AvailableProvider[] = [
  {
    id: 'cursor',
    name: 'Cursor',
    description: 'Modelos de inteligência artificial do Cursor (Composer, Grok, etc.)',
    logo: 'cursor',
    credential_type: 'api_key',
    credential_label: 'API Key / Token do Cursor',
    credential_placeholder: 'Informe seu token ou chave do Cursor',
    capabilities: ['models', 'responses', 'streaming'],
    connection_type: 'cursor',
    default_endpoint: 'http://127.0.0.1:9090/v1',
    enabled: true,
    management_credential_label: 'Gateway API Key',
    management_credential_placeholder: 'Chave do gateway do Connexto Proxy (X-Gateway-API-Key)',
  },
  {
    id: 'openrouter',
    name: 'OpenRouter',
    description: 'Roteamento unificado para centenas de modelos de IA',
    logo: 'openrouter',
    credential_type: 'api_key',
    credential_label: 'API Key do OpenRouter',
    credential_placeholder: 'sk-or-v1-...',
    capabilities: ['models', 'responses', 'streaming'],
    connection_type: 'openrouter',
    default_endpoint: 'https://openrouter.ai/api/v1',
    enabled: true,
  },
]

const existingProviders: AdminProvider[] = [
  {
    id: 'openrouter',
    name: 'OpenRouter',
    type: 'openrouter',
    endpoint: 'https://openrouter.ai/api/v1',
    enabled: true,
    credential_configured: true,
    created_at: '2026-11-04T00:00:00Z',
    updated_at: '2026-11-04T00:00:00Z',
  },
]

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('AddProviderModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(queries.listAvailableProviders).mockResolvedValue({ providers: mockAvailable })
  })

  it('renders available providers and shows connected count per provider type', async () => {
    renderWithClient(
      <AddProviderModal
        open={true}
        onOpenChange={vi.fn()}
        existingProviders={existingProviders}
      />,
    )

    await screen.findByText('Cursor')
    expect(screen.getByText('OpenRouter')).toBeDefined()
    expect(screen.getByText('1 conectada')).toBeDefined()
    expect(screen.getAllByText('Nova conta').length).toBeGreaterThanOrEqual(2)

    const cursorIcon = document.querySelector('img[data-ai-icon-slug="cursor"]')
    const openrouterIcon = document.querySelector('img[data-ai-icon-slug="openrouter"]')
    expect(cursorIcon).not.toBeNull()
    expect(openrouterIcon).not.toBeNull()
    expect(cursorIcon?.getAttribute('src')).toBeTruthy()
    expect(openrouterIcon?.getAttribute('src')).toBeTruthy()
  })

  it('advances to credential form when clicking an available provider', async () => {
    renderWithClient(
      <AddProviderModal
        open={true}
        onOpenChange={vi.fn()}
        existingProviders={existingProviders}
      />,
    )

    const cursorText = await screen.findByText('Cursor')
    const cursorButton = cursorText.closest('button')
    if (!cursorButton) throw new Error('Cursor button not found')
    fireEvent.click(cursorButton)

    await waitFor(() => {
      expect(screen.getByText('Conectar conta: Cursor')).toBeDefined()
      expect(screen.getByLabelText('Nome da conta')).toBeDefined()
      expect(screen.getByText('API Key / Token do Cursor')).toBeDefined()
      expect(screen.getByPlaceholderText('Informe seu token ou chave do Cursor')).toBeDefined()
    })
  })

  it('toggles password visibility with eye button', async () => {
    renderWithClient(
      <AddProviderModal
        open={true}
        onOpenChange={vi.fn()}
        existingProviders={existingProviders}
      />,
    )

    const cursorText = await screen.findByText('Cursor')
    const cursorButton = cursorText.closest('button')
    if (!cursorButton) throw new Error('Cursor button not found')
    fireEvent.click(cursorButton)

    const input = await screen.findByPlaceholderText('Informe seu token ou chave do Cursor')
    expect(input.getAttribute('type')).toBe('password')

    const toggleBtn = screen.getByLabelText('Exibir segredo')
    fireEvent.click(toggleBtn)

    expect(input.getAttribute('type')).toBe('text')
  })

  it('shows optional Gateway API Key field for Cursor and sends both secrets when filled', async () => {
    vi.mocked(queries.validateProvider).mockResolvedValue({ status: 'reachable' })
    vi.mocked(queries.createProvider).mockResolvedValue({
      id: 'cursor',
      name: 'Cursor',
      type: 'cursor',
      endpoint: 'http://127.0.0.1:9090/v1',
      enabled: true,
      credential_configured: true,
      management_credential_configured: true,
      created_at: '2026-11-05T00:00:00Z',
      updated_at: '2026-11-05T00:00:00Z',
    })
    vi.mocked(queries.syncProviderModels).mockResolvedValue({
      provider: 'cursor',
      outcome: 'success',
      counts: { discovered: 1, added: 1, updated: 0, unchanged: 0, missing: 0 },
      duration_ms: 50,
    })

    renderWithClient(
      <AddProviderModal open={true} onOpenChange={vi.fn()} existingProviders={existingProviders} />,
    )

    fireEvent.click((await screen.findByText('Cursor')).closest('button')!)

    fireEvent.change(
      await screen.findByPlaceholderText('Informe seu token ou chave do Cursor'),
      { target: { value: 'cur_token' } },
    )
    fireEvent.change(
      screen.getByPlaceholderText('Chave do gateway do Connexto Proxy (X-Gateway-API-Key)'),
      { target: { value: 'gw_cursor' } },
    )

    fireEvent.click(screen.getByRole('button', { name: /Validar e adicionar/i }))

    await waitFor(() => {
      expect(queries.createProvider).toHaveBeenCalledWith({
        template_id: 'cursor',
        name: 'Cursor',
        type: 'cursor',
        endpoint: 'http://127.0.0.1:9090/v1',
        credential: 'cur_token',
        management_credential: 'gw_cursor',
        enabled: true,
      })
    })
  })

  it('validates, saves connection and synchronizes models on submit', async () => {
    vi.mocked(queries.validateProvider).mockResolvedValue({ status: 'reachable' })
    vi.mocked(queries.createProvider).mockResolvedValue({
      id: 'cursor',
      name: 'Cursor',
      type: 'cursor',
      endpoint: 'http://127.0.0.1:9090/v1',
      enabled: true,
      credential_configured: true,
      created_at: '2026-11-05T00:00:00Z',
      updated_at: '2026-11-05T00:00:00Z',
    })
    vi.mocked(queries.syncProviderModels).mockResolvedValue({
      provider: 'cursor',
      outcome: 'success',
      counts: {
        discovered: 6,
        added: 6,
        updated: 0,
        unchanged: 0,
        missing: 0,
      },
      duration_ms: 120,
    })

    const onProviderAdded = vi.fn()

    renderWithClient(
      <AddProviderModal
        open={true}
        onOpenChange={vi.fn()}
        existingProviders={existingProviders}
        onProviderAdded={onProviderAdded}
      />,
    )

    const cursorText = await screen.findByText('Cursor')
    fireEvent.click(cursorText.closest('button')!)

    const input = await screen.findByPlaceholderText('Informe seu token ou chave do Cursor')
    fireEvent.change(input, { target: { value: 'cur_secret_token_123' } })

    const submitBtn = screen.getByRole('button', { name: /Validar e adicionar/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(queries.validateProvider).toHaveBeenCalledWith('cursor', {
        credential: 'cur_secret_token_123',
      })
      expect(queries.createProvider).toHaveBeenCalledWith({
        template_id: 'cursor',
        name: 'Cursor',
        type: 'cursor',
        endpoint: 'http://127.0.0.1:9090/v1',
        credential: 'cur_secret_token_123',
        enabled: true,
      })
      expect(queries.syncProviderModels).toHaveBeenCalledWith('cursor')
      expect(screen.getByText('Cursor conectado com sucesso!')).toBeDefined()
      expect(screen.getAllByText('6').length).toBeGreaterThanOrEqual(1)
      expect(onProviderAdded).toHaveBeenCalled()
    })
  })

  it('allows adding a second account for an already configured provider', async () => {
    vi.mocked(queries.validateProvider).mockResolvedValue({ status: 'reachable' })
    vi.mocked(queries.createProvider).mockResolvedValue({
      id: 'openrouter-trabalho',
      name: 'OpenRouter Trabalho',
      type: 'openrouter',
      endpoint: 'https://openrouter.ai/api/v1',
      enabled: true,
      credential_configured: true,
      created_at: '2026-11-05T00:00:00Z',
      updated_at: '2026-11-05T00:00:00Z',
    })
    vi.mocked(queries.syncProviderModels).mockResolvedValue({
      provider: 'openrouter-trabalho',
      outcome: 'success',
      counts: {
        discovered: 12,
        added: 12,
        updated: 0,
        unchanged: 0,
        missing: 0,
      },
      duration_ms: 200,
    })

    const onProviderAdded = vi.fn()

    renderWithClient(
      <AddProviderModal
        open={true}
        onOpenChange={vi.fn()}
        existingProviders={existingProviders}
        onProviderAdded={onProviderAdded}
      />,
    )

    const openRouterText = await screen.findByText('OpenRouter')
    const button = openRouterText.closest('button')
    if (!button) throw new Error('OpenRouter button not found')
    fireEvent.click(button)

    await waitFor(() => {
      expect(screen.getByText('Conectar conta: OpenRouter')).toBeDefined()
    })

    const nameInput = screen.getByLabelText('Nome da conta') as HTMLInputElement
    expect(nameInput.value).toBe('OpenRouter 2')
    fireEvent.change(nameInput, { target: { value: 'OpenRouter Trabalho' } })

    const input = screen.getByPlaceholderText('sk-or-v1-...')
    fireEvent.change(input, { target: { value: 'sk-or-v1-work-secret' } })

    const submitBtn = screen.getByRole('button', { name: /Validar e adicionar/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(queries.validateProvider).toHaveBeenCalledWith('openrouter', {
        credential: 'sk-or-v1-work-secret',
      })
      expect(queries.createProvider).toHaveBeenCalledWith({
        template_id: 'openrouter',
        name: 'OpenRouter Trabalho',
        type: 'openrouter',
        endpoint: 'https://openrouter.ai/api/v1',
        credential: 'sk-or-v1-work-secret',
        enabled: true,
      })
      expect(queries.syncProviderModels).toHaveBeenCalledWith('openrouter-trabalho')
      expect(screen.getByText('OpenRouter Trabalho conectado com sucesso!')).toBeDefined()
      expect(onProviderAdded).toHaveBeenCalled()
    })
  })
})

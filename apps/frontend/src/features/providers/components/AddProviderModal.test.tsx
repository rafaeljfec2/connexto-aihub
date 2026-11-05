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

  it('renders available providers and marks already configured ones as editable', async () => {
    renderWithClient(
      <AddProviderModal
        open={true}
        onOpenChange={vi.fn()}
        existingProviders={existingProviders}
      />,
    )

    await screen.findByText('Cursor')
    expect(screen.getByText('OpenRouter')).toBeDefined()
    expect(screen.getByText('Configurado · Editar')).toBeDefined()
    expect(screen.getByText('Disponível')).toBeDefined()
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
      expect(screen.getByText('Conectar Cursor')).toBeDefined()
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
      expect(queries.validateProvider).toHaveBeenCalledWith('cursor', 'cur_secret_token_123')
      expect(queries.createProvider).toHaveBeenCalledWith({
        id: 'cursor',
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

  it('allows editing an already configured provider and calls updateProvider', async () => {
    vi.mocked(queries.validateProvider).mockResolvedValue({ status: 'reachable' })
    vi.mocked(queries.updateProvider).mockResolvedValue({
      id: 'openrouter',
      name: 'OpenRouter',
      type: 'openrouter',
      endpoint: 'https://openrouter.ai/api/v1',
      enabled: true,
      credential_configured: true,
      created_at: '2026-11-04T00:00:00Z',
      updated_at: '2026-11-05T00:00:00Z',
    })
    vi.mocked(queries.syncProviderModels).mockResolvedValue({
      provider: 'openrouter',
      outcome: 'success',
      counts: {
        discovered: 12,
        added: 2,
        updated: 10,
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
      expect(screen.getByText('Atualizar credencial: OpenRouter')).toBeDefined()
    })

    const input = screen.getByPlaceholderText('sk-or-v1-...')
    fireEvent.change(input, { target: { value: 'sk-or-v1-new-secret' } })

    const submitBtn = screen.getByRole('button', { name: /Validar e atualizar/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(queries.validateProvider).toHaveBeenCalledWith('openrouter', 'sk-or-v1-new-secret')
      expect(queries.updateProvider).toHaveBeenCalledWith('openrouter', {
        credential: 'sk-or-v1-new-secret',
      })
      expect(queries.syncProviderModels).toHaveBeenCalledWith('openrouter')
      expect(screen.getByText('OpenRouter atualizado com sucesso!')).toBeDefined()
    })
  })
})

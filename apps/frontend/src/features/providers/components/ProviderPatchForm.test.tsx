import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ProviderPatchForm } from './ProviderPatchForm'
import * as queries from '../api/queries'
import type { AdminProvider } from '../types'

vi.mock('../api/queries', async () => {
  const actual = await vi.importActual<typeof queries>('../api/queries')
  return {
    ...actual,
    validateProvider: vi.fn(),
    clearAntigravityProxyRuntimeSessions: vi.fn(),
    startAntigravityProxyAuth: vi.fn(),
    clearCursorUsageProxyRuntimeSessions: vi.fn(),
    startCursorProxyAuth: vi.fn(),
    getCursorProxyAuthStatus: vi.fn(),
    getCursorProxyUsageSession: vi.fn(),
    probeCursorProxyUsageCollect: vi.fn(),
    updateProvider: vi.fn(),
  }
})

function renderPatchForm(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>)
}

const openRouterProvider: AdminProvider = {
  id: 'openrouter-primary',
  name: 'OpenRouter Primary',
  type: 'openrouter',
  endpoint: 'https://openrouter.ai/api/v1',
  enabled: true,
  credential_configured: true,
  management_credential_configured: false,
  created_at: '2026-11-04T00:00:00Z',
  updated_at: '2026-11-04T00:00:00Z',
}

const cursorProvider: AdminProvider = {
  id: 'cursor',
  name: 'Cursor',
  type: 'cursor',
  endpoint: 'http://127.0.0.1:9090/v1',
  enabled: true,
  credential_configured: true,
  management_credential_configured: true,
  created_at: '2026-11-04T00:00:00Z',
  updated_at: '2026-11-04T00:00:00Z',
}

const antigravityProvider: AdminProvider = {
  id: 'antigravity',
  name: 'Antigravity',
  type: 'antigravity',
  endpoint: 'http://127.0.0.1:9090/v1',
  enabled: true,
  credential_configured: false,
  management_credential_configured: true,
  created_at: '2026-11-04T00:00:00Z',
  updated_at: '2026-11-04T00:00:00Z',
}

describe('ProviderPatchForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(queries.getCursorProxyUsageSession).mockResolvedValue({
      status: 'authentication_required',
      accountId: 'cursor',
    })
    vi.mocked(queries.probeCursorProxyUsageCollect).mockResolvedValue({
      status: 'authentication_required',
      accountId: 'cursor',
    })
    vi.mocked(queries.validateProvider).mockResolvedValue({ status: 'reachable' })
    vi.mocked(queries.startAntigravityProxyAuth).mockResolvedValue({
      operation_id: 'op-1',
      status: 'pending',
      auth_url: 'https://oauth.example/authorize',
      state: 'st',
      expires_at: '2099-01-01T00:00:00Z',
    })
  })

  it('keeps name field locked and disabled so the user cannot change provider identity', () => {
    renderPatchForm(<ProviderPatchForm provider={openRouterProvider} isPending={false} onSubmit={vi.fn()} />)

    const nameInput = screen.getByLabelText('Name')
    expect(nameInput.hasAttribute('disabled')).toBe(true)
    expect(nameInput.getAttribute('readonly')).toBe('')
    expect((nameInput as HTMLInputElement).value).toBe('OpenRouter Primary')
  })

  it('submits separate OpenRouter credentials after successful connection test', async () => {
    const onSubmit = vi.fn()
    renderPatchForm(<ProviderPatchForm provider={openRouterProvider} isPending={false} onSubmit={onSubmit} />)

    const managementInput = screen.getByLabelText('Management API Key (write-only)')
    const inferenceInput = screen.getByLabelText('Credential (write-only)')
    expect(managementInput.getAttribute('type')).toBe('password')
    expect((managementInput as HTMLInputElement).value).toBe('')
    expect(managementInput.getAttribute('placeholder')).toBe('Not configured')
    expect(inferenceInput.getAttribute('type')).toBe('password')
    expect((inferenceInput as HTMLInputElement).value).toBe('')

    fireEvent.change(managementInput, { target: { value: 'management-key-fixture' } })
    fireEvent.change(inferenceInput, { target: { value: 'inference-key-fixture' } })

    const form = screen.getByRole('button', { name: /Save changes|Salvar alterações/i }).closest('form')
    if (!form) throw new Error('Provider patch form was not rendered')
    fireEvent.submit(form)

    await waitFor(() => {
      expect(queries.validateProvider).toHaveBeenCalledWith('openrouter-primary', {
        credential: 'inference-key-fixture',
      })
      expect(onSubmit).toHaveBeenCalledWith({
        management_credential: 'management-key-fixture',
        credential: 'inference-key-fixture',
      })
    })
  })

  it('allows testing connection explicitly via Testar conexão button', async () => {
    renderPatchForm(<ProviderPatchForm provider={openRouterProvider} isPending={false} onSubmit={vi.fn()} />)

    const inferenceInput = screen.getByLabelText('Credential (write-only)')
    fireEvent.change(inferenceInput, { target: { value: 'test-key-probe' } })

    const testBtn = screen.getByRole('button', { name: /Testar conexão/i })
    fireEvent.click(testBtn)

    await waitFor(() => {
      expect(queries.validateProvider).toHaveBeenCalledWith('openrouter-primary', {
        credential: 'test-key-probe',
      })
      expect(screen.getByText(/Conexão testada com sucesso!/i)).toBeDefined()
    })
  })

  it('for Antigravity, Testar conexão clarifies probe vs ACP context', async () => {
    renderPatchForm(<ProviderPatchForm provider={antigravityProvider} isPending={false} onSubmit={vi.fn()} />)

    const testBtn = screen.getByRole('button', { name: /Testar conexão/i })
    fireEvent.click(testBtn)

    await waitFor(() => {
      expect(queries.validateProvider).toHaveBeenCalledWith('antigravity', undefined)
      expect(screen.getByText(/Gateway alcançável e operacional/i)).toBeDefined()
    })
  })

  it('for Antigravity, passes optional OAuth redirect URI to startAntigravityProxyAuth', async () => {
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null)

    renderPatchForm(<ProviderPatchForm provider={antigravityProvider} isPending={false} onSubmit={vi.fn()} />)

    const redirectInput = screen.getByLabelText(/OAuth redirect URI/i)
    fireEvent.change(redirectInput, { target: { value: 'https://proxy.example/v1/providers/antigravity/auth/callback' } })

    fireEvent.click(screen.getByRole('button', { name: /Login OAuth/i }))

    await waitFor(() => {
      expect(queries.startAntigravityProxyAuth).toHaveBeenCalledWith('antigravity', {
        redirect_uri: 'https://proxy.example/v1/providers/antigravity/auth/callback',
      })
    })

    openSpy.mockRestore()
  })

  it('blocks saving and shows error message if connection test fails', async () => {
    vi.mocked(queries.validateProvider).mockRejectedValue(new Error('Credencial inválida no upstream'))
    const onSubmit = vi.fn()
    renderPatchForm(<ProviderPatchForm provider={openRouterProvider} isPending={false} onSubmit={onSubmit} />)

    const inferenceInput = screen.getByLabelText('Credential (write-only)')
    fireEvent.change(inferenceInput, { target: { value: 'invalid-secret' } })

    const form = screen.getByRole('button', { name: /Save changes|Salvar alterações/i }).closest('form')
    if (!form) throw new Error('Provider patch form was not rendered')
    fireEvent.submit(form)

    await waitFor(() => {
      expect(queries.validateProvider).toHaveBeenCalledWith('openrouter-primary', {
        credential: 'invalid-secret',
      })
      expect(onSubmit).not.toHaveBeenCalled()
      expect(screen.getByText(/Falha no teste de conexão: Credencial inválida no upstream/i)).toBeDefined()
    })
  })

  it('for Cursor, shows gateway field, proxy auth actions, and submits management_credential', async () => {
    vi.mocked(queries.startCursorProxyAuth).mockResolvedValue({
      status: 'authentication_pending',
      sessionId: 'sess-cur-1',
      expiresAt: '2099-01-01T00:00:00Z',
      interactionUrl: 'https://proxy.example/cursor/login',
    })
    const onSubmit = vi.fn()
    renderPatchForm(<ProviderPatchForm provider={cursorProvider} isPending={false} onSubmit={onSubmit} />)

    expect(screen.getByLabelText('Gateway API Key (write-only)')).toBeDefined()
    expect(screen.getByRole('button', { name: /Login web/i })).toBeDefined()

    fireEvent.change(screen.getByLabelText('Gateway API Key (write-only)'), {
      target: { value: 'gw-key-new' },
    })
    fireEvent.change(screen.getByLabelText('Credential (write-only)'), {
      target: { value: 'cursor-token-new' },
    })

    fireEvent.click(screen.getByRole('button', { name: /Login web/i }))
    await waitFor(() => {
      expect(queries.startCursorProxyAuth).toHaveBeenCalledWith('cursor')
    })

    const form = screen.getByRole('button', { name: /Save changes|Salvar alterações/i }).closest('form')
    if (!form) throw new Error('Provider patch form was not rendered')
    fireEvent.submit(form)

    await waitFor(() => {
      expect(queries.validateProvider).toHaveBeenCalledWith('cursor', {
        credential: 'cursor-token-new',
      })
      expect(queries.validateProvider).toHaveBeenCalledWith('cursor', {
        management_credential: 'gw-key-new',
      })
      expect(onSubmit).toHaveBeenCalledWith({
        credential: 'cursor-token-new',
        management_credential: 'gw-key-new',
      })
    })
  })

  it('does not show an OpenRouter Management API Key field for other providers', () => {
    renderPatchForm(
      <ProviderPatchForm
        provider={{ ...openRouterProvider, type: 'opencode-go' }}
        isPending={false}
        onSubmit={vi.fn()}
      />,
    )

    expect(screen.queryByLabelText('Management API Key (write-only)')).toBeNull()
  })

})

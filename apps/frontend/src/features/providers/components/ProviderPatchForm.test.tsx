import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ProviderPatchForm } from './ProviderPatchForm'
import type { AdminProvider } from '../types'

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

describe('ProviderPatchForm', () => {
  it('submits separate OpenRouter credentials through blank password fields', async () => {
    const onSubmit = vi.fn()
    render(<ProviderPatchForm provider={openRouterProvider} isPending={false} onSubmit={onSubmit} />)

    const managementInput = screen.getByLabelText('Management API Key (write-only)')
    const inferenceInput = screen.getByLabelText('Credential (write-only)')
    expect(managementInput.getAttribute('type')).toBe('password')
    expect((managementInput as HTMLInputElement).value).toBe('')
    expect(managementInput.getAttribute('placeholder')).toBe('Not configured')
    expect(inferenceInput.getAttribute('type')).toBe('password')
    expect((inferenceInput as HTMLInputElement).value).toBe('')

    fireEvent.change(managementInput, { target: { value: 'management-key-fixture' } })
    fireEvent.change(inferenceInput, { target: { value: 'inference-key-fixture' } })
    const form = screen.getByRole('button', { name: 'Save changes' }).closest('form')
    if (!form) throw new Error('Provider patch form was not rendered')
    fireEvent.submit(form)

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        management_credential: 'management-key-fixture',
        credential: 'inference-key-fixture',
      })
    })
  })

  it('does not show an OpenRouter Management API Key field for other providers', () => {
    render(
      <ProviderPatchForm
        provider={{ ...openRouterProvider, type: 'opencode-go' }}
        isPending={false}
        onSubmit={vi.fn()}
      />,
    )

    expect(screen.queryByLabelText('Management API Key (write-only)')).toBeNull()
  })
})

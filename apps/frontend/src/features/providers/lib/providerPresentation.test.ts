import { describe, expect, it } from 'vitest'
import type { AdminProvider } from '../types'
import { providerIconSlug, providerSituation } from './providerPresentation'

const base: AdminProvider = {
  id: 'openrouter',
  name: 'OpenRouter',
  type: 'openrouter',
  endpoint: 'https://example.com',
  enabled: true,
  credential_configured: true,
  created_at: '',
  updated_at: '',
}

describe('providerIconSlug', () => {
  it('maps configured provider ids to ai-icons slugs', () => {
    expect(providerIconSlug(base)).toBe('openrouter')
    expect(providerIconSlug({ ...base, id: 'opencode-go', type: 'opencode-go' })).toBe('opencode')
    expect(providerIconSlug({ ...base, id: 'cursor', type: 'cursor' })).toBe('cursor')
  })
})

describe('providerSituation', () => {
  it('marks enabled providers without considering quota as sem leitura', () => {
    expect(providerSituation(base)).toEqual({ state: 'na', label: 'Sem leitura' })
  })

  it('marks disabled providers', () => {
    expect(providerSituation({ ...base, enabled: false })).toEqual({
      state: 'off',
      label: 'Desabilitado',
    })
  })

  it('marks providers pending credential', () => {
    expect(providerSituation({ ...base, credential_configured: false })).toEqual({
      state: 'warn',
      label: 'Pendente',
    })
  })
})

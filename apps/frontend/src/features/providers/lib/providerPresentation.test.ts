import { describe, expect, it } from 'vitest'
import type { AdminProvider } from '../types'
import { providerSituation } from './providerPresentation'

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

describe('providerSituation', () => {
  it('marks enabled providers without quota as sem leitura', () => {
    expect(providerSituation(base)).toEqual({ state: 'na', label: 'Sem leitura' })
  })

  it('marks disabled providers', () => {
    expect(providerSituation({ ...base, enabled: false })).toEqual({
      state: 'off',
      label: 'Desabilitado',
    })
  })
})

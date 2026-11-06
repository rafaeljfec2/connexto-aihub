import { hubProviderIconSlug } from '../../../shared/branding'
import type { AdminProvider } from '../types'

export type ProviderSituation = 'ok' | 'warn' | 'na' | 'off'

export function providerInitials(provider: AdminProvider): string {
  if (provider.name.trim().length >= 2) {
    return provider.name.slice(0, 2).toUpperCase()
  }
  return provider.id.slice(0, 2).toUpperCase()
}

/** Resolves the ai-icons slug for a configured admin provider. */
export function providerIconSlug(provider: Pick<AdminProvider, 'id' | 'type'>): string {
  return hubProviderIconSlug(provider.id) || hubProviderIconSlug(provider.type)
}

/** Credential required for connection/UI gating (gateway for Antigravity, API key for others). */
export function providerCredentialConfigured(provider: AdminProvider): boolean {
  if (provider.type === 'antigravity') {
    return provider.management_credential_configured === true
  }
  return provider.credential_configured
}

export function providerConnectionSubtitle(provider: AdminProvider): string {
  const typeLabel = provider.type ? `${provider.type} · ` : ''
  if (!provider.enabled) {
    return `${typeLabel}desabilitado`
  }
  if (!providerCredentialConfigured(provider)) {
    return `${typeLabel}credencial pendente`
  }
  return `${typeLabel}conectado`
}

/** Connection-only situation when no quota snapshot is considered. */
export function providerSituation(provider: AdminProvider): {
  state: ProviderSituation
  label: string
} {
  if (!provider.enabled) {
    return { state: 'off', label: 'Desabilitado' }
  }
  if (!providerCredentialConfigured(provider)) {
    return { state: 'warn', label: 'Pendente' }
  }
  return { state: 'na', label: 'Sem leitura' }
}

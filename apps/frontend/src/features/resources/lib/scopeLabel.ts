import i18n from '../../../shared/i18n'
import type { ResourceScope } from '../types'

export function formatScopeLabel(scope: ResourceScope, providerId?: string, modelId?: string): string {
  const base = scope.toUpperCase()
  if (scope === 'provider' && providerId) {
    return `${base} · ${providerId}`
  }
  if (scope === 'model' && (providerId || modelId)) {
    const model = modelId ?? '—'
    const provider = providerId ?? '—'
    return `${base} · ${provider}/${model}`
  }
  if (scope === 'global') {
    return i18n.t('resources:scope.workspaceLabel')
  }
  return base
}

export function policyDisplayName(
  scope: ResourceScope,
  policyId: number,
  providerId?: string,
  modelId?: string,
): string {
  if (scope === 'global') {
    return i18n.t('resources:scope.workspacePolicy', { policyId })
  }
  if (scope === 'provider' && providerId) {
    return i18n.t('resources:scope.providerPolicy', { providerId })
  }
  if (scope === 'model') {
    return i18n.t('resources:scope.modelPolicy', {
      provider: providerId ?? '?',
      model: modelId ?? '?',
    })
  }
  return i18n.t('resources:scope.genericPolicy', { policyId })
}

export function concurrencyPolicyDisplayName(
  scope: ResourceScope,
  policyId: number,
  providerId?: string,
  modelId?: string,
): string {
  if (scope === 'global') {
    return i18n.t('resources:scope.workspaceConcurrency', { policyId })
  }
  if (scope === 'provider' && providerId) {
    return providerId
  }
  if (scope === 'model') {
    return i18n.t('resources:scope.modelConcurrency', {
      provider: providerId ?? '?',
      model: modelId ?? '?',
    })
  }
  return i18n.t('resources:scope.genericConcurrencyPolicy', { policyId })
}

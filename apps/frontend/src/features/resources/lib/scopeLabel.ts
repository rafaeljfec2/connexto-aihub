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
    return 'WORKSPACE'
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
    return `Limite do workspace #${policyId}`
  }
  if (scope === 'provider' && providerId) {
    return `Limite do provedor · ${providerId}`
  }
  if (scope === 'model') {
    return `Limite do modelo · ${providerId ?? '?'}/${modelId ?? '?'}`
  }
  return `Política #${policyId}`
}

export function concurrencyPolicyDisplayName(
  scope: ResourceScope,
  policyId: number,
  providerId?: string,
  modelId?: string,
): string {
  if (scope === 'global') {
    return `Concorrência do workspace #${policyId}`
  }
  if (scope === 'provider' && providerId) {
    return providerId
  }
  if (scope === 'model') {
    return `Concorrência do modelo · ${providerId ?? '?'}/${modelId ?? '?'}`
  }
  return `Política de concorrência #${policyId}`
}

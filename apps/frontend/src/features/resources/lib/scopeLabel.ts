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
  return base
}

export function policyDisplayName(
  scope: ResourceScope,
  policyId: number,
  providerId?: string,
  modelId?: string,
): string {
  if (scope === 'global') {
    return `Global quota #${policyId}`
  }
  if (scope === 'provider' && providerId) {
    return `Provider quota · ${providerId}`
  }
  if (scope === 'model') {
    return `Model quota · ${providerId ?? '?'}/${modelId ?? '?'}`
  }
  return `Policy #${policyId}`
}

export function concurrencyPolicyDisplayName(
  scope: ResourceScope,
  policyId: number,
  providerId?: string,
  modelId?: string,
): string {
  if (scope === 'global') {
    return `Global concurrency #${policyId}`
  }
  if (scope === 'provider' && providerId) {
    return `Provider concurrency · ${providerId}`
  }
  if (scope === 'model') {
    return `Model concurrency · ${providerId ?? '?'}/${modelId ?? '?'}`
  }
  return `Concurrency policy #${policyId}`
}

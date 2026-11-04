import { formatQuotaFetchedAt } from '../../overview/lib/formatQuotaDate'
import { formatQuotaValue } from '../../overview/lib/formatQuotaValue'
import {
  formatGoProductLabel,
  goPlusChipPercent,
  goPlusMetersFromSnapshot,
  type GoPlusMeterView,
} from '../../overview/lib/goPlusMeters'
import type { ProviderQuotaSnapshot, ProviderQuotaStatus } from '../../overview/types/quota'
import type { AdminProvider } from '../types'
import type { ProviderSituation } from './providerPresentation'

/** Account USD summary uses 2 fraction digits (Overview account row fidelity). */
function formatAccountUsd(value: number): string {
  return `US$ ${value.toLocaleString('pt-BR', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`
}

function formatMetricAmount(value: number, unit?: string): string {
  if (unit === 'USD') return formatAccountUsd(value)
  return formatQuotaValue(value, unit)
}

export type FranchiseSummary = {
  title: string
  value: string
}

export type ConsumptionCell =
  | {
      kind: 'percent'
      label: string
      percent: number
    }
  | {
      kind: 'usage'
      label: string
      primary: string
      secondary?: string
    }
  | {
      kind: 'empty'
      label: string
    }

export type GovernanceMetrics = {
  monitored: number
  nearLimit: number
  atLimit: number
  withoutReading: number
}

const EMPTY_DASH = '—'

export function primaryQuotaSnapshot(
  accountId: string,
  snapshots: readonly ProviderQuotaSnapshot[],
): ProviderQuotaSnapshot | undefined {
  const matches = snapshots.filter((snapshot) => snapshot.account_id === accountId)
  return (
    matches.find((snapshot) => snapshot.scope === 'subscription') ??
    matches.find((snapshot) => snapshot.scope === 'account') ??
    matches.find((snapshot) => snapshot.scope === 'api_key') ??
    matches[0]
  )
}

function snapshotStatusLabel(status: ProviderQuotaStatus): string {
  if (status === 'available') return 'Disponível'
  if (status === 'rate_limited') return 'Limite da API atingido'
  if (status === 'timeout') return 'Tempo limite excedido'
  if (status === 'unconfigured_credential') return 'Credencial ausente'
  if (status === 'unauthorized') return 'Sem permissão'
  if (status === 'unsupported') return 'Sem leitura'
  return 'Sem leitura'
}

function snapshotSituationState(status: ProviderQuotaStatus): ProviderSituation {
  if (status === 'available') return 'ok'
  if (status === 'unauthorized' || status === 'unconfigured_credential') return 'warn'
  return 'na'
}

/** Connection-level state first; then provider-reported quota status. */
export function providerQuotaSituation(
  provider: AdminProvider,
  snapshot: ProviderQuotaSnapshot | undefined,
): { state: ProviderSituation; label: string } {
  if (!provider.enabled) {
    return { state: 'off', label: 'Desabilitado' }
  }
  if (!provider.credential_configured) {
    return { state: 'warn', label: 'Pendente' }
  }
  if (!snapshot) {
    return { state: 'na', label: 'Sem leitura' }
  }
  return {
    state: snapshotSituationState(snapshot.status),
    label: snapshotStatusLabel(snapshot.status),
  }
}

const goPeriodLabels: Record<GoPlusMeterView['id'], string> = {
  go_rolling: 'janela 5h',
  go_weekly: 'semanal',
  go_monthly: 'mensal',
}

/** Primary Go meter matches Console chip: prefer weekly when present. */
function primaryGoMeter(snapshot: ProviderQuotaSnapshot): GoPlusMeterView | undefined {
  if (snapshot.status !== 'available') return undefined
  const meters = goPlusMetersFromSnapshot(snapshot)
  if (meters.length === 0) return undefined
  const percent = goPlusChipPercent(meters)
  if (percent == null) return undefined
  return meters.find((meter) => meter.percent === percent && meter.id === 'go_weekly') ??
    meters.find((meter) => meter.percent === percent) ??
    meters[0]
}

function primaryGoPercentFromSnapshot(snapshot: ProviderQuotaSnapshot): number | undefined {
  return primaryGoMeter(snapshot)?.percent
}

function accountUsageMetrics(snapshot: ProviderQuotaSnapshot): {
  usage?: { used: number; unit?: string }
  credits?: { limit: number; unit?: string }
} {
  if (snapshot.status !== 'available') return {}
  const usageMetric = snapshot.quotas.find((metric) => metric.metric === 'total_usage')
  const creditsMetric = snapshot.quotas.find((metric) => metric.metric === 'total_credits')
  const usage =
    usageMetric?.used != null ? { used: usageMetric.used, unit: usageMetric.unit } : undefined
  const credits =
    creditsMetric?.limit != null
      ? { limit: creditsMetric.limit, unit: creditsMetric.unit }
      : undefined
  return { usage, credits }
}

function hasDisplayableReading(snapshot: ProviderQuotaSnapshot | undefined): boolean {
  if (!snapshot || snapshot.status !== 'available') return false
  if (primaryGoPercentFromSnapshot(snapshot) != null) return true
  const { usage, credits } = accountUsageMetrics(snapshot)
  return usage != null || credits != null
}

export function franchiseSummary(snapshot: ProviderQuotaSnapshot | undefined): FranchiseSummary {
  if (!snapshot || snapshot.status !== 'available') {
    return {
      title: 'Não monitorada',
      value: snapshot?.error?.message ?? 'Sem fonte de quota configurada',
    }
  }

  const goMeter = primaryGoMeter(snapshot)
  if (goMeter != null) {
    const product = formatGoProductLabel(snapshot.workspace_name)
    const period = goPeriodLabels[goMeter.id]
    return {
      title: `Franquia ${product} (${period})`,
      value: `${goMeter.percent.toLocaleString('pt-BR')}% reportado pelo provedor`,
    }
  }

  const { usage, credits } = accountUsageMetrics(snapshot)
  if (usage != null || credits != null) {
    const parts: string[] = []
    if (usage != null) {
      parts.push(`${formatMetricAmount(usage.used, usage.unit)} consumido`)
    }
    if (credits != null) {
      parts.push(`Limite ${formatMetricAmount(credits.limit, credits.unit)}`)
    }
    return {
      title: 'Leitura oficial',
      value: parts.join(' · '),
    }
  }

  return {
    title: 'Leitura oficial',
    value: `${snapshot.quotas.length} métrica(s) do provedor`,
  }
}

export function consumptionCell(snapshot: ProviderQuotaSnapshot | undefined): ConsumptionCell {
  if (!snapshot || snapshot.status !== 'available') {
    return { kind: 'empty', label: 'Sem dados' }
  }

  const goMeter = primaryGoMeter(snapshot)
  if (goMeter != null) {
    return {
      kind: 'percent',
      label: `Percentual ${goPeriodLabels[goMeter.id]}`,
      percent: goMeter.percent,
    }
  }

  const { usage, credits } = accountUsageMetrics(snapshot)
  if (usage != null || credits != null) {
    const primary =
      usage != null
        ? formatMetricAmount(usage.used, usage.unit)
        : credits != null
          ? formatMetricAmount(credits.limit, credits.unit)
          : EMPTY_DASH
    const secondary =
      usage != null && credits != null
        ? `Limite ${formatMetricAmount(credits.limit, credits.unit)}`
        : usage != null
          ? 'Uso reportado'
          : 'Créditos da conta'
    return {
      kind: 'usage',
      label: 'Leitura oficial',
      primary,
      secondary,
    }
  }

  return { kind: 'empty', label: 'Sem dados' }
}

export function availableBalanceDisplay(snapshot: ProviderQuotaSnapshot | undefined): string {
  if (!snapshot || snapshot.status !== 'available' || snapshot.scope !== 'account') {
    return EMPTY_DASH
  }

  const credits = snapshot.quotas.find((metric) => metric.metric === 'total_credits')
  const usage = snapshot.quotas.find((metric) => metric.metric === 'total_usage')
  if (
    credits?.limit == null ||
    usage?.used == null ||
    credits.unit !== 'USD' ||
    usage.unit !== 'USD'
  ) {
    return EMPTY_DASH
  }

  return formatAccountUsd(credits.limit - usage.used)
}

export function formatQuotaResetAt(snapshot: ProviderQuotaSnapshot | undefined): string {
  if (!snapshot) return 'Não informado'
  const goMeter = primaryGoMeter(snapshot)
  if (goMeter?.resetsAt) return formatQuotaFetchedAt(goMeter.resetsAt)
  const fallback = snapshot.quotas.find((metric) => metric.resets_at)?.resets_at
  if (!fallback) return 'Não informado'
  return formatQuotaFetchedAt(fallback)
}

export function primaryGoPeriodLabel(snapshot: ProviderQuotaSnapshot | undefined): string | undefined {
  if (!snapshot) return undefined
  const goMeter = primaryGoMeter(snapshot)
  return goMeter ? goPeriodLabels[goMeter.id] : undefined
}

export function formatQuotaSourceName(snapshot: ProviderQuotaSnapshot | undefined): string {
  if (!snapshot) return 'Sem leitura na API'
  if (snapshot.status !== 'available') {
    return snapshot.error?.message ?? 'Sem leitura na API'
  }
  return snapshot.source.name
}

export function formatQuotaUpdatedAt(
  snapshot: ProviderQuotaSnapshot | undefined,
  fallbackIso?: string,
): string {
  if (snapshot?.source.fetched_at) {
    return formatQuotaFetchedAt(snapshot.source.fetched_at)
  }
  if (fallbackIso) {
    return formatQuotaFetchedAt(fallbackIso)
  }
  return EMPTY_DASH
}

export function primaryGoPercent(snapshot: ProviderQuotaSnapshot | undefined): number | undefined {
  if (!snapshot) return undefined
  return primaryGoPercentFromSnapshot(snapshot)
}

function meterHasProviderAlert(snapshot: ProviderQuotaSnapshot): boolean {
  return snapshot.quotas.some(
    (metric) =>
      typeof metric.scope === 'string' &&
      metric.scope.startsWith('status:') &&
      metric.scope !== 'status:ok',
  )
}

function meterAtLimit(snapshot: ProviderQuotaSnapshot): boolean {
  return snapshot.quotas.some((metric) => {
    if (typeof metric.scope !== 'string' || !metric.scope.startsWith('status:')) return false
    const status = metric.scope.slice('status:'.length).toLowerCase()
    return status.includes('limit') || status.includes('exceed') || status === 'blocked'
  })
}

export function countGovernanceMetrics(
  providers: readonly AdminProvider[],
  snapshots: readonly ProviderQuotaSnapshot[],
): GovernanceMetrics {
  let withoutReading = 0
  let nearLimit = 0
  let atLimit = 0

  for (const provider of providers) {
    const snapshot = primaryQuotaSnapshot(provider.id, snapshots)
    if (!hasDisplayableReading(snapshot)) {
      withoutReading += 1
      continue
    }
    if (!snapshot) continue
    if (meterAtLimit(snapshot)) {
      atLimit += 1
    } else if (meterHasProviderAlert(snapshot)) {
      nearLimit += 1
    }
  }

  return {
    monitored: providers.length,
    nearLimit,
    atLimit,
    withoutReading,
  }
}

import { formatNumber } from '../../../shared/i18n/format'
import i18n from '../../../shared/i18n'
import { formatQuotaFetchedAt } from '../../overview/lib/formatQuotaDate'
import { formatQuotaValue } from '../../overview/lib/formatQuotaValue'
import {
  formatGoProductLabel,
  goPlusMetersFromSnapshot,
  type GoPlusMeterView,
} from '../../overview/lib/goPlusMeters'
import { categorizeAntigravityQuotas } from '../../overview/lib/antigravityCategories'
import type { ProviderQuotaSnapshot, ProviderQuotaStatus } from '../../overview/types/quota'
import type { AdminProvider } from '../types'
import {
  cursorPrepaidBalanceUsd,
  formatAccountUsd,
  formatCursorWeeklyPercent,
} from './cursorQuotaMetrics'
import { providerCredentialConfigured, type ProviderSituation } from './providerPresentation'

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
      barValue: number
      barMax: number
    }
  | {
      kind: 'usage'
      label: string
      primary: string
      secondary?: string
      /** Official used/limit fill when both exist; omit when bar cannot be drawn from provider fields. */
      barValue?: number
      barMax?: number
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
  if (status === 'available') return i18n.t('providers:situation.available')
  if (status === 'rate_limited') return i18n.t('providers:situation.rateLimited')
  if (status === 'timeout') return i18n.t('providers:situation.timeout')
  if (status === 'unconfigured_credential') return i18n.t('providers:situation.unconfiguredCredential')
  if (status === 'unauthorized') return i18n.t('providers:situation.unauthorized')
  if (status === 'unsupported') return i18n.t('providers:situation.unsupported')
  return i18n.t('providers:situation.noReading')
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
    return { state: 'off', label: i18n.t('providers:situation.disabled') }
  }
  if (!providerCredentialConfigured(provider)) {
    return { state: 'warn', label: i18n.t('providers:situation.pending') }
  }
  if (!snapshot) {
    return { state: 'na', label: i18n.t('providers:situation.noReading') }
  }
  if (provider.type === 'antigravity' && snapshot.status === 'unsupported') {
    return { state: 'na', label: i18n.t('providers:situation.noQuotaReading') }
  }
  if (provider.type === 'cursor' && snapshot.status === 'unauthorized') {
    return { state: 'warn', label: i18n.t('providers:situation.proxyWebLogin') }
  }
  return {
    state: snapshotSituationState(snapshot.status),
    label: snapshotStatusLabel(snapshot.status),
  }
}

export function goPeriodLabel(id: GoPlusMeterView['id']): string {
  if (id === 'go_rolling') return i18n.t('providers:quota.goPeriodRolling')
  if (id === 'go_weekly') return i18n.t('providers:quota.goPeriodWeekly')
  return i18n.t('providers:quota.goPeriodMonthly')
}

/** Primary Go meter for provider governance: prefer monthly meter when present. */
function primaryGoMeter(snapshot: ProviderQuotaSnapshot): GoPlusMeterView | undefined {
  if (snapshot.status !== 'available') return undefined
  const meters = goPlusMetersFromSnapshot(snapshot)
  if (meters.length === 0) return undefined
  return (
    meters.find((meter) => meter.id === 'go_monthly') ??
    meters.find((meter) => meter.id === 'go_weekly') ??
    meters[0]
  )
}

function primaryGoPercentFromSnapshot(snapshot: ProviderQuotaSnapshot): number | undefined {
  return primaryGoMeter(snapshot)?.percent
}

export function primaryCursorPercent(snapshot: ProviderQuotaSnapshot | undefined): number | undefined {
  if (!snapshot || snapshot.status !== 'available' || snapshot.provider_id !== 'cursor') {
    return undefined
  }
  const percentMetric = snapshot.quotas.find((metric) => metric.metric === 'weekly_used_percent')
  const creditsMetric = snapshot.quotas.find((metric) => metric.metric === 'total_credits')
  const usageMetric = snapshot.quotas.find((metric) => metric.metric === 'total_usage')
  return (
    percentMetric?.consumed_percent ??
    creditsMetric?.consumed_percent ??
    usageMetric?.consumed_percent
  )
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
  if (primaryCursorPercent(snapshot) != null) return true
  const { usage, credits } = accountUsageMetrics(snapshot)
  return usage != null || credits != null
}

function cursorQuotaUnavailableSummary(snapshot: ProviderQuotaSnapshot): FranchiseSummary | undefined {
  if (snapshot.provider_id !== 'cursor') return undefined
  if (snapshot.status === 'unauthorized' || snapshot.error?.code === 'cursor_auth_required') {
    return {
      title: i18n.t('providers:quota.notMonitored'),
      value: i18n.t('providers:quota.webLoginPending'),
    }
  }
  return undefined
}

export function franchiseSummary(snapshot: ProviderQuotaSnapshot | undefined): FranchiseSummary {
  if (!snapshot || snapshot.status !== 'available') {
    if (snapshot?.status === 'unsupported' && snapshot.error?.code === 'quota_not_applicable') {
      return {
        title: i18n.t('providers:quota.franchiseNotIntegrated'),
        value: i18n.t('providers:quota.noOfficialQuota'),
      }
    }
    const cursorSummary = snapshot ? cursorQuotaUnavailableSummary(snapshot) : undefined
    if (cursorSummary) return cursorSummary
    return {
      title: i18n.t('providers:quota.notMonitored'),
      value: snapshot?.error?.message ?? i18n.t('providers:quota.noQuotaSource'),
    }
  }

  const goMeter = primaryGoMeter(snapshot)
  if (goMeter != null) {
    const product = formatGoProductLabel(snapshot.workspace_name)
    const period = goPeriodLabel(goMeter.id)
    return {
      title: i18n.t('providers:quota.franchiseGo', { product, period }),
      value: i18n.t('providers:quota.reportedByProvider', {
        percent: formatNumber(goMeter.percent),
      }),
    }
  }

  if (snapshot.provider_id === 'cursor') {
    const percent = primaryCursorPercent(snapshot)
    const { usage, credits } = accountUsageMetrics(snapshot)
    if (percent != null || usage != null || credits != null) {
      const parts: string[] = []
      if (percent != null) {
        parts.push(
          i18n.t('providers:quota.reported', {
            percent: formatNumber(formatCursorWeeklyPercent(percent)),
          }),
        )
      }
      if (usage != null) {
        parts.push(
          i18n.t('providers:quota.consumed', {
            amount: formatMetricAmount(usage.used, usage.unit),
          }),
        )
      }
      if (credits != null) {
        parts.push(
          i18n.t('providers:quota.limit', {
            amount: formatMetricAmount(credits.limit, credits.unit),
          }),
        )
      }
      return {
        title: i18n.t('providers:quota.cursorWeeklyFranchise'),
        value: parts.join(' · '),
      }
    }
  }

  if (snapshot.provider_id === 'antigravity') {
    const tier = snapshot.workspace_name ? ` (${snapshot.workspace_name})` : ''
    const families = categorizeAntigravityQuotas(snapshot.quotas)
    const active = families.filter((f) => f.id !== 'autocomplete_background')
    if (active.length > 0) {
      const summaryParts = active.map((f) => {
        // Pick the bottleneck (lowest remaining percentage) limit for this family
        const primaryLimit =
          f.limits.length > 0
            ? f.limits.reduce((prev, curr) =>
                curr.remainingPercent < prev.remainingPercent ? curr : prev,
              )
            : undefined
        const shortTitle =
          f.id === 'claude_gpt_models' ? 'Claude/GPT' : f.id === 'gemini_models' ? 'Gemini' : f.title
        return `${shortTitle} ${primaryLimit ? `${formatNumber(primaryLimit.remainingPercent)}%` : ''}`
      })
      return {
        title: `Antigravity${tier}`,
        value: summaryParts.join(' · '),
      }
    }
    return {
      title: `Antigravity${tier}`,
      value: i18n.t('providers:quota.modelsReported', { count: snapshot.quotas.length }),
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
      title: i18n.t('providers:quota.officialReading'),
      value: parts.join(' · '),
    }
  }

  return {
    title: i18n.t('providers:quota.officialReading'),
    value: i18n.t('providers:quota.metricsReported', { count: snapshot.quotas.length }),
  }
}

export function consumptionCell(snapshot: ProviderQuotaSnapshot | undefined): ConsumptionCell {
  if (!snapshot || snapshot.status !== 'available') {
    return { kind: 'empty', label: i18n.t('providers:quota.noData') }
  }

  const goMeter = primaryGoMeter(snapshot)
  if (goMeter != null) {
    return {
      kind: 'percent',
      label: i18n.t('providers:quota.percentLabel', { period: goPeriodLabel(goMeter.id) }),
      percent: goMeter.percent,
      barValue: goMeter.percent,
      barMax: 100,
    }
  }

  if (snapshot.provider_id === 'antigravity') {
    const families = categorizeAntigravityQuotas(snapshot.quotas)
    const active = families.filter((f) => f.id !== 'autocomplete_background')

    // Prioritize Gemini models weekly consumption with actual reported models
    const geminiFamily = active.find((f) => f.id === 'gemini_models')
    if (geminiFamily && geminiFamily.limits.length > 0) {
      const weeklyLimit = geminiFamily.limits.find(
        (l) => l.windowType === 'weekly' && l.models.length > 0,
      )
      const targetLimit =
        weeklyLimit ??
        geminiFamily.limits.find((l) => l.models.length > 0) ??
        geminiFamily.limits[0]
      return {
        kind: 'percent',
        label: `Gemini (${targetLimit.windowLabel})`,
        percent: targetLimit.remainingPercent,
        barValue: targetLimit.remainingPercent,
        barMax: 100,
      }
    }

    let bottleneck: { title: string; remainingPercent: number; windowLabel: string } | undefined
    for (const f of active) {
      for (const limit of f.limits) {
        if (!bottleneck || limit.remainingPercent < bottleneck.remainingPercent) {
          const shortTitle =
            f.id === 'claude_gpt_models' ? 'Claude/GPT' : f.id === 'gemini_models' ? 'Gemini' : f.title
          bottleneck = {
            title: shortTitle,
            remainingPercent: limit.remainingPercent,
            windowLabel: limit.windowLabel,
          }
        }
      }
    }

    if (bottleneck) {
      return {
        kind: 'percent',
        label: `${bottleneck.title} (${bottleneck.windowLabel})`,
        percent: bottleneck.remainingPercent,
        barValue: bottleneck.remainingPercent,
        barMax: 100,
      }
    }

    return {
      kind: 'usage',
      label: i18n.t('providers:quota.quotaByModel'),
      primary: i18n.t('providers:quota.modelsCount', { count: snapshot.quotas.length }),
      secondary: i18n.t('providers:quota.ratePerModel'),
    }
  }

  if (snapshot.provider_id === 'cursor') {
    const { usage, credits } = accountUsageMetrics(snapshot)
    const percent =
      primaryCursorPercent(snapshot) ??
      (usage != null && credits?.limit != null && credits.limit > 0
        ? (usage.used / credits.limit) * 100
        : undefined)

    if (percent != null) {
      const rounded = formatCursorWeeklyPercent(percent)
      return {
        kind: 'percent',
        label: i18n.t('providers:quota.weeklyFranchise'),
        percent: rounded,
        barValue: Math.min(100, rounded),
        barMax: 100,
      }
    }
  }

  const { usage, credits } = accountUsageMetrics(snapshot)
  const usageLabel =
    snapshot.provider_id === 'cursor'
      ? i18n.t('providers:quota.weeklyFranchise')
      : i18n.t('providers:quota.officialReading')
  if (usage != null && credits != null) {
    return {
      kind: 'usage',
      label: usageLabel,
      primary: `${formatMetricAmount(usage.used, usage.unit)} / ${formatMetricAmount(credits.limit, credits.unit)}`,
      secondary:
        snapshot.provider_id === 'cursor'
          ? i18n.t('providers:quota.weeklyUsageLimit')
          : i18n.t('providers:quota.officialUsageLimit'),
      barValue: usage.used,
      barMax: credits.limit,
    }
  }
  if (usage != null) {
    return {
      kind: 'usage',
      label: i18n.t('providers:quota.officialReading'),
      primary: formatMetricAmount(usage.used, usage.unit),
      secondary: i18n.t('providers:quota.reportedUsage'),
    }
  }
  if (credits != null) {
    return {
      kind: 'usage',
      label: i18n.t('providers:quota.officialReading'),
      primary: formatMetricAmount(credits.limit, credits.unit),
      secondary: i18n.t('providers:quota.accountCredits'),
    }
  }

  return { kind: 'empty', label: i18n.t('providers:quota.noData') }
}

export function availableBalanceDisplay(snapshot: ProviderQuotaSnapshot | undefined): string {
  if (!snapshot || snapshot.status !== 'available' || snapshot.scope !== 'account') {
    return EMPTY_DASH
  }

  const prepaid = cursorPrepaidBalanceUsd(snapshot)
  if (prepaid != null) {
    return formatAccountUsd(prepaid)
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
  if (!snapshot) return i18n.t('providers:quota.notInformed')
  const goMeter = primaryGoMeter(snapshot)
  if (goMeter?.resetsAt) return formatQuotaFetchedAt(goMeter.resetsAt)
  const fallback = snapshot.quotas.find((metric) => metric.resets_at)?.resets_at
  if (!fallback) return i18n.t('providers:quota.notInformed')
  return formatQuotaFetchedAt(fallback)
}

export function primaryGoPeriodLabel(snapshot: ProviderQuotaSnapshot | undefined): string | undefined {
  if (!snapshot) return undefined
  const goMeter = primaryGoMeter(snapshot)
  return goMeter ? goPeriodLabel(goMeter.id) : undefined
}

export function formatQuotaSourceName(snapshot: ProviderQuotaSnapshot | undefined): string {
  if (!snapshot) return i18n.t('providers:quota.noApiReading')
  if (snapshot.status !== 'available') {
    if (snapshot.provider_id === 'cursor') {
      return i18n.t('providers:quota.sourceCursorCollector')
    }
    return snapshot.error?.message ?? i18n.t('providers:quota.noApiReading')
  }
  if (snapshot.source.name === 'antigravity_proxy_usage_api') {
    return i18n.t('providers:quota.sourceAntigravityApi')
  }
  if (snapshot.source.name === 'cursor_proxy_usage_collect') {
    return i18n.t('providers:quota.sourceCursorCollect')
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
    if (
      provider.type === 'antigravity' &&
      providerCredentialConfigured(provider) &&
      snapshot?.status === 'unsupported'
    ) {
      continue
    }
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

import i18n from '../../../shared/i18n'
import { formatNumber } from '../../../shared/i18n/format'
import type { DataScope, ProviderQuotaMetric, ProviderQuotaSnapshot } from '../types/quota'

export function overviewDefaultAccountLabel(): string {
  return i18n.t('overview:providersPanel.defaultAccountLabel')
}

export function overviewScopeLabel(scope: DataScope): string {
  if (scope === 'api_key') return i18n.t('overview:quotaRow.scopeApiKey')
  if (scope === 'account') return i18n.t('overview:quotaRow.scopeAccount')
  if (scope === 'subscription') return i18n.t('overview:quotaRow.scopeSubscription')
  if (scope === 'workspace') return i18n.t('overview:quotaRow.scopeWorkspace')
  return i18n.t('overview:quotaRow.scopeOther', { scope })
}

export function overviewStatusLabel(status: ProviderQuotaSnapshot['status']): string {
  if (status === 'available') return i18n.t('overview:quotaRow.statusAvailable')
  if (status === 'rate_limited') return i18n.t('overview:quotaRow.statusRateLimited')
  if (status === 'timeout') return i18n.t('overview:quotaRow.statusTimeout')
  if (status === 'unconfigured_credential') return i18n.t('overview:quotaRow.statusUnconfigured')
  if (status === 'unauthorized') return i18n.t('overview:quotaRow.statusUnauthorized')
  return i18n.t('overview:quotaRow.statusUnsupported')
}

export function overviewStatusMessage(snapshot: ProviderQuotaSnapshot): string {
  if (snapshot.status === 'unconfigured_credential' && snapshot.scope === 'account') {
    return i18n.t('overview:quotaRow.statusUnconfiguredManagement')
  }
  if (snapshot.status === 'unconfigured_credential') {
    return i18n.t('overview:quotaRow.statusUnconfiguredApiKey')
  }
  if (snapshot.status === 'unauthorized') {
    return i18n.t('overview:quotaRow.statusUnauthorizedDetail')
  }
  if (snapshot.status === 'unsupported') return i18n.t('overview:quotaRow.statusNoSummary')
  if (snapshot.status === 'rate_limited') return i18n.t('overview:quotaRow.statusRateLimitedHint')
  if (snapshot.status === 'timeout') return i18n.t('overview:quotaRow.statusTimeoutHint')
  if (snapshot.status === 'error') return i18n.t('overview:quotaRow.statusQueryError')
  return i18n.t('overview:quotaRow.statusNoFields')
}

export function overviewMetricLabel(scope: DataScope, metric: ProviderQuotaMetric): string {
  if (scope === 'api_key' && metric.metric === 'usage') return i18n.t('overview:metrics.keyUsage')
  if (scope === 'account' && metric.metric === 'total_usage') return i18n.t('overview:metrics.accountTotalUsage')
  if (scope === 'account' && metric.metric === 'total_credits') return i18n.t('overview:metrics.accountCredits')
  if (metric.metric === 'go_rolling') return i18n.t('overview:metrics.franchiseRolling')
  if (metric.metric === 'go_weekly') return i18n.t('overview:metrics.franchiseWeekly')
  if (metric.metric === 'go_monthly') return i18n.t('overview:metrics.franchiseMonthly')
  return metric.label || metric.metric
}

export function overviewRemainingPercentLabel(percent: number): string {
  return i18n.t('overview:quotaRow.remainingPercent', { percent: formatNumber(percent) })
}

export function overviewConsumedPercentLabel(percent: number): string {
  return i18n.t('overview:quotaRow.consumedPercent', { percent: formatNumber(percent) })
}

export function overviewUsedValueLabel(value: string): string {
  return i18n.t('overview:quotaRow.used', { value })
}

export function overviewLimitValueLabel(value: string): string {
  return i18n.t('overview:quotaRow.limit', { value })
}

export function overviewTotalValueLabel(value: string): string {
  return i18n.t('overview:quotaRow.total', { value })
}

export function overviewRemainingValueLabel(value: string): string {
  return i18n.t('overview:quotaRow.remaining', { value })
}

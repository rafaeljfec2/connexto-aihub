import i18n from '../../../shared/i18n'
import { formatNumber } from '../../../shared/i18n/format'
import type { ProviderQuotaSnapshot } from '../../overview/types/quota'

export type CursorProductWeeklyEntry = Readonly<{
  product: string
  label: string
  usedUsd: number
  percent?: number
}>

export type CursorMonthlyAdditionalUsage = Readonly<{
  usedUsd?: number
  limitUsd?: number
  creditsUsedUsd?: number
  additionalUsageOff?: boolean
  startsAt?: string
  resetsAt?: string
}>

export function formatAccountUsd(value: number): string {
  return `US$ ${formatNumber(value, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

/**
 * Matches Cursor/Grok dashboard display: positive non-zero usage below 1%
 * rounds up to 1%, while 1% and above uses standard integer rounding.
 */
export function formatCursorWeeklyPercent(percent: number): number {
  if (!Number.isFinite(percent) || percent <= 0) return 0
  if (percent < 1) return 1
  return Math.round(percent)
}

function formatCursorProductLabel(product: string): string {
  const normalized = product.trim().toLowerCase()
  if (normalized === 'cursor') return 'Cursor'
  if (normalized === 'grok') return 'Grok'
  if (!normalized) return i18n.t('providers:productDefault')
  return normalized.charAt(0).toUpperCase() + normalized.slice(1)
}

export function cursorWeeklyByProduct(
  snapshot: ProviderQuotaSnapshot | undefined,
): CursorProductWeeklyEntry[] {
  if (!snapshot || snapshot.status !== 'available') return []
  const items: CursorProductWeeklyEntry[] = []
  for (const metric of snapshot.quotas) {
    if (typeof metric.scope !== 'string' || !metric.scope.startsWith('product:')) continue
    if (metric.used == null) continue
    const product = metric.scope.slice('product:'.length).trim()
    if (!product) continue
    items.push({
      product,
      label: formatCursorProductLabel(product),
      usedUsd: metric.used,
      percent:
        metric.consumed_percent != null
          ? formatCursorWeeklyPercent(metric.consumed_percent)
          : undefined,
    })
  }
  return items
}

export function cursorMonthlyAdditionalUsage(
  snapshot: ProviderQuotaSnapshot | undefined,
): CursorMonthlyAdditionalUsage | undefined {
  if (!snapshot || snapshot.status !== 'available') return undefined
  const monthly = snapshot.quotas.find((metric) => metric.metric === 'monthly_additional_usage')
  const creditsUsed = snapshot.quotas.find((metric) => metric.metric === 'monthly_credits_used')
  if (!monthly && !creditsUsed) return undefined

  let additionalUsageOff: boolean | undefined
  if (monthly?.scope === 'additional_off') additionalUsageOff = true
  if (monthly?.scope === 'additional_enabled') additionalUsageOff = false

  return {
    usedUsd: monthly?.used,
    limitUsd: monthly?.limit,
    creditsUsedUsd: creditsUsed?.used,
    additionalUsageOff,
    startsAt: monthly?.starts_at,
    resetsAt: monthly?.resets_at,
  }
}

export function cursorWeeklyIncludedStatus(
  snapshot: ProviderQuotaSnapshot | undefined,
): string | undefined {
  if (!snapshot || snapshot.status !== 'available') return undefined
  const metric = snapshot.quotas.find((item) => item.metric === 'weekly_included_status')
  if (!metric?.scope?.startsWith('weekly_status:')) return undefined
  const status = metric.scope.slice('weekly_status:'.length).trim()
  return status || undefined
}

export function cursorPrepaidBalanceUsd(
  snapshot: ProviderQuotaSnapshot | undefined,
): number | undefined {
  if (!snapshot || snapshot.status !== 'available' || snapshot.scope !== 'account') {
    return undefined
  }
  const prepaid = snapshot.quotas.find((metric) => metric.metric === 'prepaid_available_credits')
  if (prepaid?.remaining != null && prepaid.unit === 'USD') {
    return prepaid.remaining
  }
  return undefined
}


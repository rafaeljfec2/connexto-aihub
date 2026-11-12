import { formatDateTime } from '../../../shared/i18n/format'

function utcDateFromIsoDate(isoDate: string): Date | undefined {
  const [year, month, day] = isoDate.split('-').map(Number)
  if (!year || !month || !day) {
    return undefined
  }
  return new Date(Date.UTC(year, month - 1, day))
}

/** Provider RFC3339 instant (e.g. quota fetch time). */
export function formatQuotaFetchedAt(isoInstant: string): string {
  const parsed = Date.parse(isoInstant)
  if (Number.isNaN(parsed)) {
    return isoInstant
  }
  return formatDateTime(new Date(parsed), { dateStyle: 'short', timeStyle: 'short' })
}

/** Calendar day from provider `YYYY-MM-DD` (UTC, no timezone shift). */
export function formatQuotaCalendarDay(isoDate: string): string {
  const date = utcDateFromIsoDate(isoDate)
  if (!date) {
    return isoDate
  }
  return formatDateTime(date, {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    timeZone: 'UTC',
  })
}

/** Compact axis label for daily usage charts (UTC calendar day). */
export function formatQuotaChartAxisDay(isoDate: string): string {
  const date = utcDateFromIsoDate(isoDate)
  if (!date) {
    return isoDate
  }
  return formatDateTime(date, { day: 'numeric', month: 'short', timeZone: 'UTC' })
}

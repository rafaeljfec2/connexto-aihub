import i18n from '../../../shared/i18n'

const MS_MINUTE = 60_000
const MS_HOUR = 60 * MS_MINUTE
const MS_DAY = 24 * MS_HOUR

/**
 * Formats a provider-returned reset timestamp as a relative label.
 * Presentation only — does not invent franchise values.
 */
export function formatResetsIn(resetsAt: string, now: Date = new Date()): string | undefined {
  const resetMs = Date.parse(resetsAt)
  if (Number.isNaN(resetMs)) return undefined

  const delta = resetMs - now.getTime()
  if (delta <= 0) return i18n.t('overview:resets.soon')

  const days = Math.floor(delta / MS_DAY)
  const hours = Math.floor((delta % MS_DAY) / MS_HOUR)
  const minutes = Math.floor((delta % MS_HOUR) / MS_MINUTE)

  if (days > 0) {
    return hours > 0
      ? i18n.t('overview:resets.inDaysHours', { days, hours })
      : i18n.t('overview:resets.inDays', { days })
  }
  if (hours > 0) {
    return minutes > 0
      ? i18n.t('overview:resets.inHoursMinutes', { hours, minutes })
      : i18n.t('overview:resets.inHours', { hours })
  }
  return i18n.t('overview:resets.inMinutes', { minutes: Math.max(minutes, 1) })
}

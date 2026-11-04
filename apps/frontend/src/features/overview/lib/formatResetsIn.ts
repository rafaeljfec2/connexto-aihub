const MS_MINUTE = 60_000
const MS_HOUR = 60 * MS_MINUTE
const MS_DAY = 24 * MS_HOUR

/**
 * Formats a provider-returned reset timestamp as a relative label (pt-BR).
 * Presentation only — does not invent franchise values.
 */
export function formatResetsIn(resetsAt: string, now: Date = new Date()): string | undefined {
  const resetMs = Date.parse(resetsAt)
  if (Number.isNaN(resetMs)) return undefined

  const delta = resetMs - now.getTime()
  if (delta <= 0) return 'Reinicia em breve'

  const days = Math.floor(delta / MS_DAY)
  const hours = Math.floor((delta % MS_DAY) / MS_HOUR)
  const minutes = Math.floor((delta % MS_HOUR) / MS_MINUTE)

  if (days > 0) {
    return hours > 0 ? `Reinicia em ${days}d ${hours}h` : `Reinicia em ${days}d`
  }
  if (hours > 0) {
    return minutes > 0 ? `Reinicia em ${hours}h ${minutes}m` : `Reinicia em ${hours}h`
  }
  return `Reinicia em ${Math.max(minutes, 1)}m`
}

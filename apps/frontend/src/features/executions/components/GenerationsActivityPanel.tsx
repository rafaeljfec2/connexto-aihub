import type { Generation } from '../types'

export type GenerationActivityDay = {
  readonly dateKey: string
  readonly dayLabel: string
  readonly count: number
  readonly heightPercent: number
  readonly showLabel: boolean
}

const DAY_MS = 24 * 60 * 60 * 1000
const DEFAULT_DAYS = 30

function toUtcDateKey(date: Date): string {
  return date.toISOString().slice(0, 10)
}

function formatDayLabel(date: Date): string {
  return new Intl.DateTimeFormat('pt-BR', {
    day: '2-digit',
    month: 'short',
    timeZone: 'UTC',
  })
    .format(date)
    .replace(/\./g, '')
    .replace(' de ', ' ')
    .toUpperCase()
}

/** Groups loaded generations by UTC day across a 30-day window ending at `now`. */
export function buildGenerationDailySeries(
  generations: readonly Generation[],
  days = DEFAULT_DAYS,
  now = new Date(),
): GenerationActivityDay[] {
  const countsByDay = new Map<string, number>()
  for (const item of generations) {
    const key = item.started_at?.slice(0, 10)
    if (!key) continue
    countsByDay.set(key, (countsByDay.get(key) ?? 0) + 1)
  }

  const endUtc = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate())
  const rawDays: Array<{ dateKey: string; dayLabel: string; count: number }> = []

  for (let offset = days - 1; offset >= 0; offset -= 1) {
    const date = new Date(endUtc - offset * DAY_MS)
    const dateKey = toUtcDateKey(date)
    rawDays.push({
      dateKey,
      dayLabel: formatDayLabel(date),
      count: countsByDay.get(dateKey) ?? 0,
    })
  }

  const maxCount = rawDays.reduce((max, entry) => Math.max(max, entry.count), 0)

  return rawDays.map((entry, index) => {
    const heightPercent =
      entry.count > 0 && maxCount > 0 ? Math.max(14, Math.round((entry.count / maxCount) * 100)) : 0
    const showLabel = index === 0 || index === rawDays.length - 1 || index % 5 === 0
    return {
      ...entry,
      heightPercent,
      showLabel,
    }
  })
}

type GenerationsActivityPanelProps = {
  readonly generations: readonly Generation[]
}

/** Daily activity chart computed from loaded generations in the current view. */
export function GenerationsActivityPanel({ generations }: GenerationsActivityPanelProps) {
  const series = buildGenerationDailySeries(generations)
  const totalLoaded = generations.length
  const totalLabel =
    totalLoaded === 1 ? '1 GERAÇÃO CARREGADA · 30 DIAS' : `${totalLoaded} GERAÇÕES CARREGADAS · 30 DIAS`

  return (
    <section className="generations-activity-panel" aria-label="Atividade de gerações">
      <div className="generations-activity-head">
        <h2 className="generations-activity-title">Atividade de gerações</h2>
        <span className="generations-activity-note">{totalLabel}</span>
      </div>
      {totalLoaded === 0 ? (
        <div className="generations-activity-empty">
          Nenhuma geração carregada para exibir atividade no período.
        </div>
      ) : (
        <>
          <div className="generations-activity-chart-frame" role="img" aria-label={totalLabel}>
            <div className="generations-activity-gridlines">
              <span />
              <span />
              <span />
            </div>
            <div className="generations-activity-chart">
              {series.map((bar) => (
                <div
                  key={bar.dateKey}
                  className="generations-activity-bar-col"
                  title={`${bar.dayLabel}: ${bar.count} ${bar.count === 1 ? 'geração' : 'gerações'}`}
                >
                  <div
                    className={
                      bar.count > 0
                        ? 'generations-activity-bar'
                        : 'generations-activity-bar generations-activity-bar-empty'
                    }
                    style={{ height: bar.count > 0 ? `${bar.heightPercent}%` : '2px' }}
                  />
                </div>
              ))}
            </div>
          </div>
          <div className="generations-activity-axis" aria-hidden="true">
            {series
              .filter((bar) => bar.showLabel)
              .map((bar) => (
                <span key={bar.dateKey}>{bar.dayLabel}</span>
              ))}
          </div>
        </>
      )}
    </section>
  )
}

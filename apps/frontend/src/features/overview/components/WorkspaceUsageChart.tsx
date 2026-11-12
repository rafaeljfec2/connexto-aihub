import { useState } from 'react'
import { formatCurrencyUsd } from '../../../shared/i18n/format'
import { formatGoProductLabel } from '../lib/goPlusMeters'
import {
  formatQuotaCalendarDay,
  formatQuotaChartAxisDay,
} from '../lib/formatQuotaDate'
import type { ProviderUsageChart, ProviderUsageChartPoint } from '../types/quota'

type WorkspaceUsageChartProps = Readonly<{
  chart: ProviderUsageChart
  testIdPrefix?: string
}>

function formatUsd(value: number): string {
  return formatCurrencyUsd(value)
}

function axisLabels(periodStart: string, periodEnd: string): [string, string, string] {
  const start = new Date(`${periodStart}T00:00:00.000Z`)
  const end = new Date(`${periodEnd}T00:00:00.000Z`)
  const midMs = start.getTime() + (end.getTime() - start.getTime()) / 2
  const mid = new Date(midMs).toISOString().slice(0, 10)
  return [
    formatQuotaChartAxisDay(periodStart),
    formatQuotaChartAxisDay(mid),
    formatQuotaChartAxisDay(periodEnd),
  ]
}

function barTooltipLabel(point: ProviderUsageChartPoint): string {
  return `${formatQuotaCalendarDay(point.date)} · ${formatUsd(point.cost_usd)}`
}

function tooltipAlign(index: number, total: number): 'start' | 'center' | 'end' {
  if (total <= 1) return 'center'
  if (index <= 2) return 'start'
  if (index >= total - 3) return 'end'
  return 'center'
}

export function WorkspaceUsageChart({
  chart,
  testIdPrefix = 'workspace-usage',
}: WorkspaceUsageChartProps) {
  const [hoveredDate, setHoveredDate] = useState<string | null>(null)
  const maxCost = Math.max(...chart.points.map((point) => point.cost_usd), 0)
  const [startLabel, midLabel, endLabel] = axisLabels(chart.period_start, chart.period_end)
  const subtitle = formatGoProductLabel(chart.subtitle)

  return (
    <section
      className="workspace-usage-card"
      data-testid={`${testIdPrefix}-chart`}
      aria-label={chart.title}
    >
      <header className="workspace-usage-head">
        <div className="min-w-0">
          <h3 className="workspace-usage-title">{chart.title}</h3>
          <p className="workspace-usage-subtitle">{subtitle}</p>
        </div>
        <p className="workspace-usage-total" data-testid={`${testIdPrefix}-total`}>
          {formatUsd(chart.total_usd)}
        </p>
      </header>

      <div
        className="workspace-usage-bars"
        role="list"
        aria-label={`Daily ${chart.unit} usage`}
        onMouseLeave={() => setHoveredDate(null)}
      >
        {chart.points.map((point, index) => {
          const heightPct =
            maxCost > 0 ? Math.max((point.cost_usd / maxCost) * 100, point.cost_usd > 0 ? 8 : 4) : 4
          const active = point.cost_usd > 0
          const isHovered = hoveredDate === point.date
          const align = tooltipAlign(index, chart.points.length)
          return (
            <button
              key={point.date}
              type="button"
              className={
                active
                  ? isHovered
                    ? 'workspace-usage-bar-hit is-active is-hovered'
                    : 'workspace-usage-bar-hit is-active'
                  : isHovered
                    ? 'workspace-usage-bar-hit is-hovered'
                    : 'workspace-usage-bar-hit'
              }
              style={{ ['--bar-height' as string]: `${heightPct}%` }}
              aria-label={barTooltipLabel(point)}
              data-testid={`${testIdPrefix}-bar-${point.date}`}
              onMouseEnter={() => setHoveredDate(point.date)}
              onFocus={() => setHoveredDate(point.date)}
              onBlur={() => setHoveredDate((current) => (current === point.date ? null : current))}
            >
              {isHovered ? (
                <span
                  className={`workspace-usage-tooltip is-align-${align}`}
                  data-testid={`${testIdPrefix}-tooltip`}
                  role="status"
                >
                  {barTooltipLabel(point)}
                </span>
              ) : null}
              <span className="workspace-usage-bar" />
            </button>
          )
        })}
      </div>

      <div className="workspace-usage-axis" aria-hidden="true">
        <span>{startLabel}</span>
        <span>{midLabel}</span>
        <span>{endLabel}</span>
      </div>
    </section>
  )
}

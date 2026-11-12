import { globalQuotaUtilizationPercent } from '../lib/summaryMetrics'
import {
  admissionsChartEndpoint,
  buildAdmissionsAreaPath,
  buildAdmissionsLinePath,
  consumptionChartDayLabels,
  consumptionChartViewBox,
} from '../lib/consumptionChart'
import type { AdmissionsDailyRow, QuotaUtilizationEntry } from '../types'
import { SectionFetchedAt } from './SectionFetchedAt'

type ConsumptionChartProps = {
  daily: AdmissionsDailyRow[]
  totalAdmitted: number
  quotaUtilization: QuotaUtilizationEntry[]
  periodLabel: string
  onTogglePeriod?: () => void
  isLoading?: boolean
  isError?: boolean
  fetchedAtMs?: number
}

export function ConsumptionChart({
  daily,
  totalAdmitted,
  quotaUtilization,
  periodLabel,
  onTogglePeriod,
  isLoading,
  isError,
  fetchedAtMs,
}: ConsumptionChartProps) {
  const headlinePercent = globalQuotaUtilizationPercent(quotaUtilization)
  const areaPath = buildAdmissionsAreaPath(daily)
  const linePath = buildAdmissionsLinePath(daily)
  const endpoint = admissionsChartEndpoint(daily)
  const dayLabels = consumptionChartDayLabels(daily)
  const hasSeries = daily.length > 0 && areaPath !== null

  return (
    <section className="resources-panel" aria-label="Consumo de recursos">
      <div className="resources-panel-head">
        <div>
          <h3 className="resources-panel-title">Consumo de recursos</h3>
          <p className="resources-panel-subtitle">Evolução agregada no período</p>
        </div>
        <button type="button" className="resources-select" onClick={onTogglePeriod}>
          {periodLabel} ⌄
        </button>
      </div>

      <div className="resources-consumption-hero">
        <strong>
          {headlinePercent === null ? totalAdmitted.toLocaleString('pt-BR') : `${headlinePercent.toFixed(0)}%`}
        </strong>
        <span>{headlinePercent === null ? 'ADMISSÕES NO PERÍODO' : 'DO LIMITE AGREGADO'}</span>
      </div>

      {isLoading ? (
        <p className="resources-empty">Carregando consumo…</p>
      ) : isError ? (
        <p className="resources-empty">Série de admissões indisponível para o período selecionado.</p>
      ) : hasSeries ? (
        <svg
          className="resources-chart-svg"
          viewBox={consumptionChartViewBox}
          preserveAspectRatio="none"
          role="img"
          aria-label="Gráfico de consumo de recursos no período"
        >
          <defs>
            <linearGradient id="resources-admissions-gradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#7896E8" stopOpacity={0.24} />
              <stop offset="100%" stopColor="#7896E8" stopOpacity={0} />
            </linearGradient>
          </defs>
          <g stroke="rgba(180,194,218,0.10)" strokeWidth="1">
            <path d="M0 14H560M0 44H560M0 74H560M0 104H560" />
          </g>
          <path className="resources-chart-area" d={areaPath} />
          {linePath ? (
            <path className="resources-chart-line" d={linePath} />
          ) : null}
          {endpoint ? (
            <circle cx={endpoint.x} cy={endpoint.y} r="3.5" fill="#CBD7F5" stroke="#202A38" strokeWidth="2" />
          ) : null}
        </svg>
      ) : (
        <p className="resources-empty">Nenhuma admissão no período selecionado.</p>
      )}

      {dayLabels.length > 0 ? (
        <div className="resources-chart-labels">
          {dayLabels.map((label, index) => (
            <span key={`${label}-${index}`}>{label}</span>
          ))}
        </div>
      ) : null}

      <SectionFetchedAt updatedAtMs={fetchedAtMs} />
    </section>
  )
}

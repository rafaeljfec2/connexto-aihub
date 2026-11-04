import { Panel } from '../../../shared/ui'
import type { TrafficSummary } from '../api/queries'

type OverviewTrafficChartPanelProps = {
  traffic?: TrafficSummary
  isLoading?: boolean
}

function buildSvgPath(buckets: Array<{ count: number }>, width = 560, height = 120): {
  areaPath: string
  linePath: string
  lastPoint: { x: number; y: number }
} {
  if (buckets.length === 0) {
    return {
      areaPath: '',
      linePath: '',
      lastPoint: { x: width, y: height - 10 },
    }
  }

  const paddingY = 16
  const usableHeight = height - paddingY * 2
  const maxCount = Math.max(...buckets.map((b) => b.count), 1)

  const points = buckets.map((b, i) => {
    const x = (i / (buckets.length - 1)) * width
    const ratio = b.count / maxCount
    const y = height - paddingY - ratio * usableHeight
    return { x, y }
  })

  // Build smooth cubic bezier curve
  let d = `M ${points[0].x} ${points[0].y}`
  for (let i = 0; i < points.length - 1; i++) {
    const p0 = points[i === 0 ? 0 : i - 1]
    const p1 = points[i]
    const p2 = points[i + 1]
    const p3 = points[i + 2 < points.length ? i + 2 : i + 1]

    const cp1x = p1.x + (p2.x - p0.x) / 6
    const cp1y = p1.y + (p2.y - p0.y) / 6
    const cp2x = p2.x - (p3.x - p1.x) / 6
    const cp2y = p2.y - (p3.y - p1.y) / 6

    d += ` C ${cp1x.toFixed(1)} ${cp1y.toFixed(1)}, ${cp2x.toFixed(1)} ${cp2y.toFixed(1)}, ${p2.x.toFixed(1)} ${p2.y.toFixed(1)}`
  }

  const lastPoint = points[points.length - 1]
  const areaPath = `${d} L ${width} ${height} L 0 ${height} Z`

  return { areaPath, linePath: d, lastPoint }
}

export function OverviewTrafficChartPanel({
  traffic,
  isLoading,
}: OverviewTrafficChartPanelProps) {
  const totalRequests = traffic?.total_requests ?? 0
  const trendPercent = traffic?.trend_percentage ?? 0
  const buckets = traffic?.buckets ?? []
  const hasHistory = traffic?.has_sufficient_history === true && buckets.length > 0

  const { areaPath, linePath, lastPoint } = buildSvgPath(buckets)
  const defaultLabels = ['00:00', '06:00', '12:00', '18:00', 'Agora']
  const labels = buckets.length === 5 ? buckets.map((b) => b.label) : defaultLabels

  return (
    <Panel className="overview-panel">
      <div className="overview-panel-head">
        <div>
          <h3 className="text-xs font-semibold text-[#E7ECF5]">Volume de requisições</h3>
          <p className="mt-0.5 text-[10px] text-[#8D9AAF]">Tráfego processado pelo gateway</p>
        </div>
        <button
          type="button"
          className="overview-panel-link cursor-default"
        >
          Últimas 24 horas ⌄
        </button>
      </div>

      <div className="overview-chart-summary">
        <strong className="text-lg font-semibold tracking-[-0.03em] text-[#F0F3EC]">
          {isLoading ? '—' : totalRequests.toLocaleString('pt-BR')}
        </strong>
        {trendPercent !== 0 ? (
          <span className="font-mono text-[10px] text-healthy">
            {trendPercent > 0 ? `+${trendPercent.toFixed(1)}%` : `${trendPercent.toFixed(1)}%`}
          </span>
        ) : null}
      </div>

      <div className="min-h-0 flex-1">
        <svg
          className="overview-chart-svg"
          viewBox="0 0 560 120"
            preserveAspectRatio="none"
            role="img"
            aria-label="Gráfico de volume de requisições ao longo das últimas 24 horas"
          >
            <defs>
              <linearGradient id="chartGradient" x1="0" x2="0" y1="0" y2="1">
                <stop offset="0" stopColor="#7896E8" stopOpacity="0.26" />
                <stop offset="1" stopColor="#7896E8" stopOpacity="0" />
              </linearGradient>
            </defs>
            {/* Horizontal subtle grid lines */}
            <g stroke="rgba(180, 194, 218, 0.11)" strokeWidth="1">
              <path d="M0 15H560M0 48H560M0 81H560M0 114H560" />
            </g>

            {hasHistory ? (
              <>
                <path d={areaPath} fill="url(#chartGradient)" />
                <path
                  d={linePath}
                  fill="none"
                  stroke="#8DA8F2"
                  strokeWidth="2.2"
                  strokeLinecap="round"
                />
                <circle
                  cx={lastPoint.x}
                  cy={lastPoint.y}
                  r="3.5"
                  fill="#CBD7F5"
                  stroke="#202A38"
                  strokeWidth="2"
                />
              </>
            ) : (
              <g>
                <line
                  x1="0"
                  y1="110"
                  x2="560"
                  y2="110"
                  stroke="rgba(180, 194, 218, 0.18)"
                  strokeWidth="1.5"
                  strokeDasharray="4 4"
                />
                <text
                  x="280"
                  y="65"
                  textAnchor="middle"
                  fill="#7D8A9D"
                  className="font-mono text-[11px]"
                >
                  {isLoading
                    ? 'Carregando série temporal…'
                    : 'Sem histórico suficiente nas últimas 24 horas'}
                </text>
              </g>
            )}
          </svg>

          <div className="mt-0.5 flex justify-between font-mono text-[9px] text-[#8592A7]">
            {labels.map((l) => (
              <span key={l}>{l}</span>
            ))}
          </div>
        </div>
    </Panel>
  )
}

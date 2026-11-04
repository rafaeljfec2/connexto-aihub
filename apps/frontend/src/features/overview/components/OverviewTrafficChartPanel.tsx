import { ChevronDown } from 'lucide-react'
import { useId, useMemo } from 'react'
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  XAxis,
  YAxis,
} from 'recharts'
import { Panel } from '../../../shared/ui'
import type { TrafficSummary } from '../api/queries'

type OverviewTrafficChartPanelProps = {
  traffic?: TrafficSummary
  isLoading?: boolean
}

const DEFAULT_LABELS = ['00:00', '06:00', '12:00', '18:00', 'Agora']

type ChartPoint = {
  label: string
  count: number
}

function buildChartData(buckets: TrafficSummary['buckets']): ChartPoint[] {
  if (buckets.length === 5) {
    return buckets.map((bucket) => ({
      label: bucket.label,
      count: bucket.count,
    }))
  }

  return DEFAULT_LABELS.map((label) => ({ label, count: 0 }))
}

export function OverviewTrafficChartPanel({
  traffic,
  isLoading,
}: OverviewTrafficChartPanelProps) {
  const gradientId = useId().replace(/:/g, '')
  const totalRequests = traffic?.total_requests ?? 0
  const trendPercent = traffic?.trend_percentage ?? 0
  const buckets = traffic?.buckets ?? []
  const hasHistory = traffic?.has_sufficient_history === true && buckets.length > 0
  const chartData = useMemo(() => buildChartData(buckets), [buckets])

  return (
    <Panel className="overview-panel overview-traffic-panel">
      <div className="overview-panel-head">
        <div>
          <h3 className="text-xs font-semibold text-[#E7ECF5]">Volume de requisições</h3>
          <p className="mt-0.5 text-[10px] text-[#8D9AAF]">Tráfego processado pelo gateway</p>
        </div>
        <button type="button" className="overview-panel-link cursor-default" aria-disabled>
          <span>Últimas 24 horas</span>
          <ChevronDown className="size-3 opacity-70" aria-hidden />
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

      <div className="overview-chart-body">
        <div className="overview-chart-plot" aria-hidden={!hasHistory && !isLoading}>
          {hasHistory ? (
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart
                data={chartData}
                margin={{ top: 12, right: 8, left: 0, bottom: 0 }}
              >
                <defs>
                  <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#7896E8" stopOpacity={0.26} />
                    <stop offset="100%" stopColor="#7896E8" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid
                  stroke="rgba(180, 194, 218, 0.11)"
                  vertical={false}
                  syncWithTicks
                />
                <XAxis
                  dataKey="label"
                  axisLine={false}
                  tickLine={false}
                  tick={{
                    fill: '#8592A7',
                    fontSize: 9,
                    fontFamily: 'IBM Plex Mono, ui-monospace, monospace',
                  }}
                  interval={0}
                  padding={{ left: 0, right: 0 }}
                  dy={6}
                />
                <YAxis hide domain={[0, 'auto']} />
                <Area
                  type="monotone"
                  dataKey="count"
                  stroke="#8DA8F2"
                  strokeWidth={2.2}
                  fill={`url(#${gradientId})`}
                  isAnimationActive={false}
                  dot={(props) => {
                    const { cx, cy, index } = props
                    if (index !== chartData.length - 1 || cx === undefined || cy === undefined) {
                      return null
                    }
                    return (
                      <circle
                        cx={cx}
                        cy={cy}
                        r={3.5}
                        fill="#CBD7F5"
                        stroke="#202A38"
                        strokeWidth={2}
                      />
                    )
                  }}
                />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="overview-chart-empty" role="status">
              <span className="overview-chart-empty-line" />
              <p className="overview-chart-empty-copy">
                {isLoading
                  ? 'Carregando série temporal…'
                  : 'Sem histórico suficiente nas últimas 24 horas'}
              </p>
            </div>
          )}
        </div>

        {!hasHistory ? (
          <div className="overview-chart-labels">
            {DEFAULT_LABELS.map((label) => (
              <span key={label}>{label}</span>
            ))}
          </div>
        ) : null}
      </div>
    </Panel>
  )
}

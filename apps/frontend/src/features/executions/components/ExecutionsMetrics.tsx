import { Activity, AlertCircle, CheckCircle2, Clock } from 'lucide-react'
import { StatCard } from '../../../shared/ui'
import type { ExecutionRecord } from '../types'

type ExecutionsMetricsProps = {
  records: ExecutionRecord[]
  isLoading?: boolean
}

export function ExecutionsMetrics({ records, isLoading }: ExecutionsMetricsProps) {
  const total = records.length
  const completed = records.filter((r) => r.outcome === 'completed').length
  const failed = records.filter((r) => r.outcome !== 'completed').length

  const durations = records
    .map((r) => (r.duration_ns ? Math.round(r.duration_ns / 1_000_000) : undefined))
    .filter((d): d is number => d !== undefined)

  const avgDuration =
    durations.length > 0 ? Math.round(durations.reduce((a, b) => a + b, 0) / durations.length) : null

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard
        label="Execuções registradas"
        icon={<Activity className="size-3.5" />}
        value={isLoading ? '…' : total}
        note="Amostra persistida de chamadas"
      />

      <StatCard
        label="Concluídas"
        icon={<CheckCircle2 className="size-3.5 text-healthy" />}
        value={
          isLoading ? (
            '…'
          ) : (
            <span className="text-healthy">{completed}</span>
          )
        }
        note="Processadas com sucesso"
      />

      <StatCard
        label="Falhas"
        icon={<AlertCircle className="size-3.5 text-failure" />}
        value={
          isLoading ? (
            '…'
          ) : (
            <span className="text-failure">{failed}</span>
          )
        }
        note="Requerem investigação operacional"
      />

      <StatCard
        label="Latência média"
        icon={<Clock className="size-3.5" />}
        value={
          isLoading ? (
            '…'
          ) : avgDuration !== null ? (
            <div className="flex items-baseline gap-1">
              <span>{avgDuration}</span>
              <span className="text-xs font-normal text-muted">ms</span>
            </div>
          ) : (
            <span className="text-sm font-normal text-muted">—</span>
          )
        }
        note="Tempo médio de resposta"
      />
    </div>
  )
}

import { Activity, AlertCircle, CheckCircle2, Clock } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { StatCard } from '../../../shared/ui'
import type { ExecutionRecord } from '../types'

type ExecutionsMetricsProps = {
  records: ExecutionRecord[]
  isLoading?: boolean
}

export function ExecutionsMetrics({ records, isLoading }: ExecutionsMetricsProps) {
  const { t } = useTranslation('executions')
  const total = records.length
  const completed = records.filter((r) => r.outcome === 'completed').length
  const failed = records.filter((r) => r.outcome !== 'completed').length

  const durations = records
    .map((r) => (r.duration_ns ? Math.round(r.duration_ns / 1_000_000) : undefined))
    .filter((d): d is number => d !== undefined)

  const avgDuration =
    durations.length > 0 ? Math.round(durations.reduce((a, b) => a + b, 0) / durations.length) : null

  const emDash = t('format.emDash')

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard
        label={t('metrics.registered.label')}
        icon={<Activity className="size-3.5" />}
        value={isLoading ? '…' : total}
        note={t('metrics.registered.note')}
      />

      <StatCard
        label={t('metrics.completed.label')}
        icon={<CheckCircle2 className="size-3.5 text-healthy" />}
        value={
          isLoading ? (
            '…'
          ) : (
            <span className="text-healthy">{completed}</span>
          )
        }
        note={t('metrics.completed.note')}
      />

      <StatCard
        label={t('metrics.failed.label')}
        icon={<AlertCircle className="size-3.5 text-failure" />}
        value={
          isLoading ? (
            '…'
          ) : (
            <span className="text-failure">{failed}</span>
          )
        }
        note={t('metrics.failed.note')}
      />

      <StatCard
        label={t('metrics.avgLatency.label')}
        icon={<Clock className="size-3.5" />}
        value={
          isLoading ? (
            '…'
          ) : avgDuration !== null ? (
            <div className="flex items-baseline gap-1">
              <span>{avgDuration}</span>
              <span className="text-xs font-normal text-muted">{t('metrics.avgLatency.unit')}</span>
            </div>
          ) : (
            <span className="text-sm font-normal text-muted">{emDash}</span>
          )
        }
        note={t('metrics.avgLatency.note')}
      />
    </div>
  )
}

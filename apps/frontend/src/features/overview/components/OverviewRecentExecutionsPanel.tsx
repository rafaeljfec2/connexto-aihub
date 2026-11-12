import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Panel } from '../../../shared/ui'
import type { ExecutionRecord } from '../../executions/types'

type OverviewRecentExecutionsPanelProps = {
  executions?: ExecutionRecord[]
  isLoading?: boolean
}

export function OverviewRecentExecutionsPanel({
  executions = [],
  isLoading,
}: OverviewRecentExecutionsPanelProps) {
  const { t } = useTranslation('overview')
  const visibleExecutions = executions.slice(0, 3)

  return (
    <Panel className="overview-panel overview-executions-panel">
      <div className="overview-panel-head">
        <div>
          <h3 className="text-xs font-semibold text-[#E7ECF5]">{t('executions.title')}</h3>
          <p className="mt-0.5 text-[0.625rem] text-[#8D9AAF]">{t('executions.subtitle')}</p>
        </div>
        <Link to="/executions" className="overview-panel-link">
          {t('executions.viewAll')}
        </Link>
      </div>

      {isLoading ? (
        <div className="py-3 text-center text-xs text-muted animate-pulse">{t('executions.loading')}</div>
      ) : visibleExecutions.length === 0 ? (
        <div className="py-3 text-center text-xs text-muted">{t('executions.empty')}</div>
      ) : (
        <table className="overview-table w-full">
          <thead>
            <tr>
              <th>{t('executions.colTime')}</th>
              <th>{t('executions.colType')}</th>
              <th>{t('executions.colModel')}</th>
              <th>{t('executions.colProvider')}</th>
              <th>{t('executions.colStatus')}</th>
              <th>{t('executions.colLatency')}</th>
            </tr>
          </thead>
          <tbody>
            {visibleExecutions.map((execution) => {
              const isSuccess = execution.outcome === 'completed'
              const formattedTime = execution.started_at
                ? new Date(execution.started_at).toLocaleTimeString(undefined, {
                    hour: '2-digit',
                    minute: '2-digit',
                    second: '2-digit',
                  })
                : '—'
              const durationMs =
                execution.duration_ns !== undefined
                  ? Math.round(execution.duration_ns / 1_000_000)
                  : undefined
              const latencyText = durationMs !== undefined ? `${durationMs} ms` : '—'

              return (
                <tr key={execution.execution_id}>
                  <td className="font-mono text-[#9AA8BB]">{formattedTime}</td>
                  <td>responses.create</td>
                  <td className="model">{execution.model_id ?? '—'}</td>
                  <td className="capitalize">{execution.provider_id ?? '—'}</td>
                  <td>
                    <span className={isSuccess ? 'overview-done-badge' : 'font-mono text-[0.5625rem] text-failure'}>
                      {isSuccess ? t('executions.completed') : t('executions.failed')}
                    </span>
                  </td>
                  <td className="font-mono text-[#9AA8BB]">{latencyText}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
    </Panel>
  )
}

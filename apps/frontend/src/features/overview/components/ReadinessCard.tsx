import { useTranslation } from 'react-i18next'
import { statusClass } from '../lib/statusClass'
import { Skeleton, StatCard } from '../../../shared/ui'

type ReadinessCardProps = {
  status?: string
  checks?: Record<string, string>
  isLoading: boolean
}

export function ReadinessCard({ status, checks, isLoading }: ReadinessCardProps) {
  const { t } = useTranslation('overview')

  return (
    <StatCard>
      <div className="text-[0.625rem] text-secondary">{t('readiness.database')}</div>
      {isLoading ? (
        <Skeleton className="mt-4 h-7 w-28" />
      ) : status ? (
        <>
          <p className={`mt-3 text-[1.4rem] font-medium tracking-tight ${statusClass(status)}`}>
            {status.toUpperCase()}
          </p>
          {checks ? (
            <ul className="mt-3 space-y-1 text-xs text-muted">
              {Object.entries(checks).map(([name, value]) => (
                <li key={name}>
                  {name}: <span className={statusClass(value)}>{value}</span>
                </li>
              ))}
            </ul>
          ) : null}
        </>
      ) : null}
    </StatCard>
  )
}

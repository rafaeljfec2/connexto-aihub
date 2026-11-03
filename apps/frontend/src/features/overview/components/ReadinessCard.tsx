import { statusClass } from '../lib/statusClass'
import { Skeleton, StatCard } from '../../../shared/ui'

type ReadinessCardProps = {
  status?: string
  checks?: Record<string, string>
  isLoading: boolean
}

export function ReadinessCard({ status, checks, isLoading }: ReadinessCardProps) {
  return (
    <StatCard>
      <div className="text-[10px] text-secondary">Database readiness</div>
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

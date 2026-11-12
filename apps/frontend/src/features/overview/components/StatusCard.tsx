import { statusClass } from '../lib/statusClass'
import { Skeleton, StatCard } from '../../../shared/ui'

type StatusCardProps = {
  title: string
  status?: string
  isLoading: boolean
}

export function StatusCard({ title, status, isLoading }: StatusCardProps) {
  return (
    <StatCard>
      <div className="flex items-center justify-between text-[0.625rem] text-secondary">
        <span>{title}</span>
      </div>
      {isLoading ? (
        <Skeleton className="mt-4 h-7 w-24" />
      ) : status ? (
        <p className={`mt-3 text-[1.4rem] font-medium tracking-tight ${statusClass(status)}`}>
          {status.toUpperCase()}
        </p>
      ) : null}
    </StatCard>
  )
}

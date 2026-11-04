import { cn } from '../../../shared/ui/cn'
import { formatOutcomeLabel } from '../lib/formatExecution'

type ExecutionOutcomeBadgeProps = {
  outcome: string
  className?: string
}

export function ExecutionOutcomeBadge({ outcome, className }: ExecutionOutcomeBadgeProps) {
  const tone =
    outcome === 'completed'
      ? 'ok'
      : outcome === 'failed' || outcome === 'timed_out'
        ? 'fail'
        : outcome === 'canceled'
          ? 'warn'
          : 'neutral'

  return (
    <span className={cn('executions-state', `executions-state-${tone}`, className)}>
      <i aria-hidden="true" />
      {formatOutcomeLabel(outcome)}
    </span>
  )
}

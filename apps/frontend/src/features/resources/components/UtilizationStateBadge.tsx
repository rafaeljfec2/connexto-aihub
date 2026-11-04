import { cn } from '../../../shared/ui/cn'

type UtilizationStateBadgeProps = {
  variant: 'ok' | 'warn' | 'muted'
  children: string
}

export function UtilizationStateBadge({ variant, children }: UtilizationStateBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded px-1.5 py-1 font-mono text-[9px] whitespace-nowrap',
        variant === 'ok' && 'bg-healthy/10 text-healthy',
        variant === 'warn' && 'bg-warning/10 text-warning',
        variant === 'muted' && 'bg-white/[0.04] text-muted',
      )}
    >
      <i className="size-[5px] rounded-full bg-current" aria-hidden />
      {children}
    </span>
  )
}

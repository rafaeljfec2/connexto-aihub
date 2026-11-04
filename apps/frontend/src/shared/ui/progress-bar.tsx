import type { HTMLAttributes } from 'react'
import { cn } from './cn'

export type ProgressBarVariant = 'cobalt' | 'warn' | 'ok' | 'na' | 'failure'

type ProgressBarProps = HTMLAttributes<HTMLDivElement> & {
  value: number
  max?: number
  variant?: ProgressBarVariant
  size?: 'sm' | 'md'
}

const variantClasses: Record<ProgressBarVariant, string> = {
  cobalt: 'bg-cobalt',
  warn: 'bg-warning',
  ok: 'bg-healthy',
  failure: 'bg-failure',
  na: 'bg-[#5E6B7C]',
}

export function ProgressBar({
  value,
  max = 100,
  variant = 'cobalt',
  size = 'sm',
  className,
  ...props
}: ProgressBarProps) {
  const percentage = max > 0 ? Math.min(100, Math.max(0, (value / max) * 100)) : 0
  const colorClass = variantClasses[variant] ?? variantClasses.cobalt

  return (
    <div
      role="progressbar"
      aria-valuenow={value}
      aria-valuemin={0}
      aria-valuemax={max}
      className={cn(
        'w-full overflow-hidden rounded-full bg-[#2A3443]',
        size === 'sm' ? 'h-1.5' : 'h-2.5',
        className,
      )}
      {...props}
    >
      <div
        className={cn('h-full rounded-full transition-all duration-300', colorClass)}
        style={{ width: `${percentage}%` }}
      />
    </div>
  )
}


import type { HTMLAttributes, ReactNode } from 'react'
import { cn } from './cn'

export function Card({ className, ...props }: HTMLAttributes<HTMLElement>) {
  return (
    <article
      className={cn(
        'rounded-[10px] border border-line bg-gradient-to-br from-raised to-surface p-4 min-h-[7rem]',
        className,
      )}
      {...props}
    />
  )
}

type StatCardProps = HTMLAttributes<HTMLElement> & {
  label?: string
  value?: ReactNode
  icon?: ReactNode
  note?: ReactNode
  valueClassName?: string
}

export function StatCard({
  label,
  value,
  icon,
  note,
  valueClassName,
  className,
  children,
  ...props
}: StatCardProps) {
  return (
    <article
      className={cn(
        'relative min-h-[7rem] rounded-[10px] border border-line bg-gradient-to-br from-raised to-surface p-4',
        className,
      )}
      {...props}
    >
      {label || icon ? (
        <div className="flex items-center justify-between gap-2 text-xs text-secondary">
          {label ? <span className="font-medium text-secondary">{label}</span> : <div />}
          {icon ? (
            <div className="grid size-[26px] place-items-center rounded-md border border-cobalt/20 bg-cobalt/10 text-accent-light">
              {icon}
            </div>
          ) : null}
        </div>
      ) : null}

      {value !== undefined ? (
        <div
          className={cn(
            'my-2.5 text-2xl font-semibold tracking-[-0.03em] text-primary',
            valueClassName,
          )}
        >
          {value}
        </div>
      ) : null}

      {note ? <div className="text-[0.625rem] text-muted">{note}</div> : null}

      {children}
    </article>
  )
}

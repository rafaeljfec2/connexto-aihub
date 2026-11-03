import type { HTMLAttributes, ReactNode } from 'react'
import { cn } from './cn'

export function Panel({ className, ...props }: HTMLAttributes<HTMLElement>) {
  return (
    <section
      className={cn('min-w-0 rounded-[10px] border border-line bg-surface p-4', className)}
      {...props}
    />
  )
}

type PanelHeaderProps = HTMLAttributes<HTMLDivElement> & {
  title: string
  subtitle?: string
  action?: ReactNode
}

export function PanelHeader({ title, subtitle, action, className, ...props }: PanelHeaderProps) {
  return (
    <div className={cn('mb-3 flex items-start justify-between gap-3', className)} {...props}>
      <div>
        <h2 className="text-xs font-semibold text-[#E7ECF5]">{title}</h2>
        {subtitle ? <p className="mt-1 text-[10px] text-muted">{subtitle}</p> : null}
      </div>
      {action}
    </div>
  )
}

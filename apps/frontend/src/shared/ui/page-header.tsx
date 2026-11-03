import type { ReactNode } from 'react'
import { cn } from './cn'

type PageHeaderProps = {
  eyebrow: string
  title: string
  lead?: string
  actions?: ReactNode
  className?: string
}

export function PageHeader({ eyebrow, title, lead, actions, className }: PageHeaderProps) {
  return (
    <header className={cn('mb-5 flex flex-wrap items-end justify-between gap-4', className)}>
      <div className="min-w-0">
        <p className="mb-2 font-mono text-[10px] uppercase tracking-[0.13em] text-cobalt">
          {eyebrow}
        </p>
        <h1 className="font-display text-[clamp(1.75rem,4vw,2.15rem)] font-normal leading-[1.05] tracking-[-0.03em] text-[#F0F3F8]">
          {title}
        </h1>
        {lead ? <p className="mt-2 max-w-2xl text-xs text-secondary">{lead}</p> : null}
      </div>
      {actions ? <div className="flex shrink-0 flex-wrap items-center gap-3">{actions}</div> : null}
    </header>
  )
}

import type { HTMLAttributes, ReactNode } from 'react'
import { cn } from './cn'

type MasterDetailLayoutProps = HTMLAttributes<HTMLDivElement> & {
  master: ReactNode
  detail: ReactNode
  detailWidth?: string
}

export function MasterDetailLayout({
  master,
  detail,
  detailWidth = 'lg:grid-cols-[minmax(0,1fr)_300px]',
  className,
  ...props
}: MasterDetailLayoutProps) {
  return (
    <div className={cn('grid grid-cols-1 items-start gap-3', detailWidth, className)} {...props}>
      <div className="min-w-0">{master}</div>
      <aside className="sticky top-6 min-w-0">{detail}</aside>
    </div>
  )
}


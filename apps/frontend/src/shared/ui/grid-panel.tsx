import type { ReactNode } from 'react'
import { cn } from './cn'
import { GridPagination, type GridPaginationProps } from './grid-pagination'

type GridPanelProps = {
  title: string
  countLabel?: string
  footnoteLeft?: string
  footnoteRight?: string
  pagination?: GridPaginationProps
  className?: string
  bodyClassName?: string
  children: ReactNode
}

export function GridPanel({
  title,
  countLabel,
  footnoteLeft,
  footnoteRight,
  pagination,
  className,
  bodyClassName,
  children,
}: GridPanelProps) {
  return (
    <section className={cn('grid-panel', className)}>
      <div className="grid-panel-head">
        <strong>{title}</strong>
        {countLabel ? <span>{countLabel}</span> : null}
      </div>
      <div className={cn('grid-panel-body', bodyClassName)}>{children}</div>
      {pagination ? <GridPagination {...pagination} className="grid-panel-pagination" /> : null}
      {footnoteLeft || footnoteRight ? (
        <footer className="grid-panel-footnote">
          <span>{footnoteLeft}</span>
          <span>{footnoteRight}</span>
        </footer>
      ) : null}
    </section>
  )
}

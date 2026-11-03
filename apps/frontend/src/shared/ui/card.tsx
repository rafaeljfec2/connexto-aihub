import type { HTMLAttributes } from 'react'
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

export function StatCard({ className, ...props }: HTMLAttributes<HTMLElement>) {
  return (
    <article
      className={cn(
        'relative min-h-[7rem] rounded-[10px] border border-line bg-gradient-to-br from-raised to-surface px-4 py-[15px]',
        className,
      )}
      {...props}
    />
  )
}

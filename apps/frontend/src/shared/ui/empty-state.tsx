import type { HTMLAttributes } from 'react'
import { cn } from './cn'

export function EmptyState({ className, ...props }: HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p className={cn('rounded-lg border border-dashed border-line py-6 text-center text-sm text-muted', className)} {...props} />
  )
}

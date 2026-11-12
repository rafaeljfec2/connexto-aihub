import { cva, type VariantProps } from 'class-variance-authority'
import type { HTMLAttributes } from 'react'
import { cn } from './cn'

const badgeVariants = cva(
  'inline-flex items-center rounded-[5px] border font-mono text-[0.5625rem] uppercase tracking-[0.07em]',
  {
    variants: {
      variant: {
        sample:
          'border-warning/25 bg-warning/10 px-2 py-1.5 text-[#D6AD72]',
        mono: 'border-line bg-raised/80 px-2 py-1 text-muted',
        live: 'border-healthy/25 bg-healthy/10 px-2 py-1 text-healthy',
      },
    },
    defaultVariants: {
      variant: 'mono',
    },
  },
)

type BadgeProps = HTMLAttributes<HTMLSpanElement> & VariantProps<typeof badgeVariants>

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant, className }))} {...props} />
}

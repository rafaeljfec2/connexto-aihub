import { cva, type VariantProps } from 'class-variance-authority'
import type { HTMLAttributes } from 'react'
import { cn } from './cn'

const bannerVariants = cva('mb-4 rounded-lg border px-4 py-3 text-sm', {
  variants: {
    variant: {
      error: 'border-failure/40 bg-failure/10 text-[#F5D0D2]',
      warning: 'border-warning/35 bg-warning/10 text-[#E8D4B0]',
      info: 'border-cobalt/35 bg-cobalt/10 text-accent-light',
    },
  },
  defaultVariants: {
    variant: 'info',
  },
})

type BannerProps = HTMLAttributes<HTMLDivElement> & VariantProps<typeof bannerVariants>

export function Banner({ className, variant, ...props }: BannerProps) {
  return <div className={cn(bannerVariants({ variant, className }))} {...props} />
}

/** @deprecated Prefer `<Banner variant="…" />` — kept for existing feature markup. */
export function legacyBannerClass(tone: 'error' | 'warning' | 'info'): string {
  return cn(bannerVariants({ variant: tone }))
}

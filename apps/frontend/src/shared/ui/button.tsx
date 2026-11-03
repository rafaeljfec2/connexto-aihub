import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import type { ButtonHTMLAttributes } from 'react'
import { cn } from './cn'

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 rounded-[7px] border text-[11px] font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cobalt/40 disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        primary:
          'border-cobalt/35 bg-accent-deep text-[#E4EBFC] hover:bg-[#314A78]',
        secondary:
          'border-cobalt/28 bg-accent-deep text-[#E4EBFC] hover:bg-[#314A78]',
        ghost:
          'border-line bg-transparent text-secondary hover:border-cobalt/30 hover:text-primary',
        icon: 'size-[30px] border-line bg-transparent p-0 text-secondary hover:border-cobalt/30',
      },
      size: {
        sm: 'h-8 px-3',
        md: 'h-9 px-3.5',
        lg: 'h-10 px-4',
      },
    },
    defaultVariants: {
      variant: 'ghost',
      size: 'md',
    },
  },
)

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }

export function Button({
  className,
  variant,
  size,
  asChild = false,
  type = 'button',
  ...props
}: ButtonProps) {
  const Comp = asChild ? Slot : 'button'
  const mergedClassName = cn(buttonVariants({ variant, size, className }))
  if (asChild) {
    return <Comp className={mergedClassName} {...props} />
  }
  return <Comp className={mergedClassName} type={type} {...props} />
}

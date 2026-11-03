import type { InputHTMLAttributes, LabelHTMLAttributes } from 'react'
import { cn } from './cn'

type InputProps = InputHTMLAttributes<HTMLInputElement>

export function Input({ className, ...props }: InputProps) {
  return (
    <input
      className={cn(
        'w-full rounded-md border border-line bg-[#09090b] px-3 py-2 text-sm text-primary outline-none transition-colors placeholder:text-muted focus:border-cobalt focus:ring-2 focus:ring-cobalt/15',
        className,
      )}
      {...props}
    />
  )
}

export function SearchField({ className, ...props }: InputProps) {
  return (
    <input
      className={cn(
        'h-[42px] w-full rounded-lg border border-line/80 bg-[#171E29] px-3 text-xs text-[#E5EBF4] outline-none placeholder:text-muted focus:border-cobalt focus:ring-[3px] focus:ring-cobalt/10',
        className,
      )}
      {...props}
    />
  )
}

export function FieldLabel({
  className,
  ...props
}: LabelHTMLAttributes<HTMLLabelElement>) {
  return (
    <label className={cn('grid gap-1.5 text-sm text-secondary', className)} {...props} />
  )
}

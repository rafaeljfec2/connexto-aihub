import type { HTMLAttributes } from 'react'
import { cn } from './cn'

export type ChipOption<T extends string = string> = {
  id: T
  label: string
  count?: number
}

type FilterChipsProps<T extends string = string> = Omit<HTMLAttributes<HTMLDivElement>, 'onChange' | 'onSelect'> & {
  options: ChipOption<T>[]
  selectedId: T
  onSelect: (id: T) => void
  label?: string
}

export function FilterChips<T extends string = string>({
  options,
  selectedId,
  onSelect,
  label,
  className,
  ...props
}: FilterChipsProps<T>) {
  return (
    <div className={cn('flex flex-wrap items-center gap-1.5', className)} {...props}>
      {label ? (
        <span className="mr-1 font-mono text-[0.5625rem] uppercase tracking-[0.07em] text-muted">
          {label}
        </span>
      ) : null}
      {options.map((option) => {
        const isActive = option.id === selectedId
        return (
          <button
            key={option.id}
            type="button"
            onClick={() => onSelect(option.id)}
            className={cn(
              'inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-[0.625rem] font-medium transition-colors cursor-pointer',
              isActive
                ? 'border-cobalt/30 bg-accent-deep text-[#EAF0FC]'
                : 'border-transparent bg-transparent text-secondary hover:bg-white/[0.04] hover:text-primary',
            )}
          >
            <span>{option.label}</span>
            {option.count !== undefined ? (
              <span className="font-mono text-[0.5625rem] opacity-75">{option.count}</span>
            ) : null}
          </button>
        )
      })}
    </div>
  )
}

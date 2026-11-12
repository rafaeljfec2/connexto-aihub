import type { HTMLAttributes, ReactNode } from 'react'
import { Info } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from './cn'

type EmptyTelemetryStateProps = HTMLAttributes<HTMLDivElement> & {
  title?: string
  description?: string
  icon?: ReactNode
}

export function EmptyTelemetryState({
  title,
  description,
  icon,
  className,
  ...props
}: EmptyTelemetryStateProps) {
  const { t } = useTranslation('common')
  const resolvedTitle = title ?? t('emptyTelemetryTitle')
  const resolvedDescription = description ?? t('emptyTelemetryDescription')

  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center rounded-[10px] border border-dashed border-line bg-surface/50 px-6 py-8 text-center',
        className,
      )}
      {...props}
    >
      <div className="mb-3 grid size-8 place-items-center rounded-lg border border-line bg-raised text-muted">
        {icon ?? <Info className="size-4 opacity-75" aria-hidden />}
      </div>
      <h3 className="text-xs font-semibold text-primary">{resolvedTitle}</h3>
      <p className="mt-1 max-w-sm text-[0.6875rem] leading-relaxed text-secondary">{resolvedDescription}</p>
    </div>
  )
}

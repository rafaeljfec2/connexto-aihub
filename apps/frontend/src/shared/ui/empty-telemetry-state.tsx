import type { HTMLAttributes, ReactNode } from 'react'
import { Info } from 'lucide-react'
import { cn } from './cn'

type EmptyTelemetryStateProps = HTMLAttributes<HTMLDivElement> & {
  title?: string
  description?: string
  icon?: ReactNode
}

export function EmptyTelemetryState({
  title = 'Dados não instrumentados',
  description = 'Esta métrica requer telemetria que ainda não está instrumentada no gateway atual. Nenhum dado é simulado.',
  icon,
  className,
  ...props
}: EmptyTelemetryStateProps) {
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
      <h3 className="text-xs font-semibold text-primary">{title}</h3>
      <p className="mt-1 max-w-sm text-[11px] leading-relaxed text-secondary">{description}</p>
    </div>
  )
}


import { ProgressBar } from '../../../shared/ui'
import { formatResetsIn } from '../lib/formatResetsIn'
import type { GoPlusMeterView } from '../lib/goPlusMeters'
import { productUsageTitle } from '../lib/goPlusMeters'

type GoPlusUsagePanelProps = Readonly<{
  meters: readonly GoPlusMeterView[]
  workspaceName?: string
  compact?: boolean
  now?: Date
  testIdPrefix?: string
}>

export function GoPlusUsagePanel({
  meters,
  workspaceName,
  compact = false,
  now,
  testIdPrefix = 'go-plus',
}: GoPlusUsagePanelProps) {
  if (meters.length === 0) return null

  return (
    <div
      className={compact ? 'space-y-2' : 'space-y-2.5 rounded-lg border border-line/80 bg-[#182333]/70 p-2.5'}
      data-testid={`${testIdPrefix}-panel`}
    >
      <header className="flex items-center justify-between gap-2">
        <h4 className="text-[10px] font-semibold text-[#E7ECF5]">{productUsageTitle(workspaceName)}</h4>
      </header>
      <ul className="space-y-2">
        {meters.map((meter) => {
          const resets = meter.resetsAt ? formatResetsIn(meter.resetsAt, now) : undefined
          const percentLabel = `${meter.percent.toLocaleString('pt-BR')}% used`
          return (
            <li key={meter.id} data-testid={`${testIdPrefix}-meter-${meter.id}`}>
              <div className="mb-1 flex items-baseline justify-between gap-2 text-[9px]">
                <span className="min-w-0 truncate font-medium text-[#CBD5E2]">
                  {meter.label} · {percentLabel}
                </span>
                {resets ? (
                  <span className="shrink-0 text-[#8290A5]" data-testid={`${testIdPrefix}-resets-${meter.id}`}>
                    {resets}
                  </span>
                ) : null}
              </div>
              <ProgressBar value={meter.percent} max={100} variant="cobalt" size="sm" />
            </li>
          )
        })}
      </ul>
    </div>
  )
}

import { formatResetsIn } from '../lib/formatResetsIn'
import {
  categorizeAntigravityQuotas,
  type AntigravityFamilyGroup,
} from '../lib/antigravityCategories'
import type { ProviderQuotaMetric } from '../types/quota'
import { CircularQuotaProgress } from './CircularQuotaProgress'

type AntigravityQuotaOverviewProps = Readonly<{
  accountId: string
  quotas: readonly ProviderQuotaMetric[]
}>

export function AntigravityQuotaOverview({ accountId, quotas }: AntigravityQuotaOverviewProps) {
  const families = categorizeAntigravityQuotas(quotas)

  // In overview card, we highlight active generative families; internal completions are condensed
  const visibleFamilies = families.filter((f) => f.id !== 'autocomplete_background')
  const backgroundGroup = families.find((f) => f.id === 'autocomplete_background')

  return (
    <div className="space-y-2" data-testid={`antigravity-overview-${accountId}`}>
      {visibleFamilies.map((family: AntigravityFamilyGroup) => (
        <div key={family.id} className="space-y-1">
          <div className="flex items-center justify-between text-[10px] font-semibold text-[#CBD5E2]">
            <span>{family.title}</span>
            <span className="text-[9px] font-normal text-[#8290A5]">
              {family.totalModelsCount} modelo(s)
            </span>
          </div>

          <div className="space-y-1 rounded bg-[#0f1722]/50 p-1.5 border border-line/30">
            {family.limits.map((limit, idx) => {
              const resetLabel = limit.resetsAt ? formatResetsIn(limit.resetsAt) : undefined
              return (
                <div
                  key={`${limit.windowType}-${idx}`}
                  className="flex items-center justify-between text-[9px]"
                  data-testid={`antigravity-pool-${family.id}-${limit.windowType}`}
                >
                  <div className="min-w-0 pr-1">
                    <span className="block font-medium text-[#AEBBD0]">{limit.windowLabel}</span>
                    {resetLabel ? (
                      <span className="block text-[8px] text-[#68778C]">{resetLabel}</span>
                    ) : null}
                  </div>

                  <div className="flex items-center gap-1.5 shrink-0">
                    <span className="font-semibold text-[#CBD5E2]">
                      {Math.round(limit.remainingPercent)}% restante
                    </span>
                    <CircularQuotaProgress percent={limit.remainingPercent} size={14} />
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      ))}

      {backgroundGroup ? (
        <div className="flex items-center justify-between text-[8px] text-[#8290A5] pt-0.5 border-t border-line/20">
          <span>{backgroundGroup.title} ({backgroundGroup.totalModelsCount} modelos)</span>
          <span className="text-[#AEBBD0]">100% restante</span>
        </div>
      ) : null}
    </div>
  )
}


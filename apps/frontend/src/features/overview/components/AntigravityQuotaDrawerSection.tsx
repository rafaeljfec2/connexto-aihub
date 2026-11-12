import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { formatNumber } from '../../../shared/i18n/format'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { formatResetsIn } from '../lib/formatResetsIn'
import {
  categorizeAntigravityQuotas,
  type AntigravityFamilyGroup,
  type AntigravityWindowLimit,
} from '../lib/antigravityCategories'
import type { ProviderQuotaMetric } from '../types/quota'
import { CircularQuotaProgress } from './CircularQuotaProgress'

type AntigravityQuotaDrawerSectionProps = Readonly<{
  quotas: readonly ProviderQuotaMetric[]
}>

function FamilyCard({ family }: { family: AntigravityFamilyGroup }) {
  const { t } = useTranslation('overview')
  const [showModels, setShowModels] = useState(false)

  return (
    <div
      className="rounded-lg border border-line bg-raised/70 p-3.5 space-y-3"
      data-testid={`antigravity-drawer-family-${family.id}`}
    >
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-semibold text-[#E1E8F3]">{family.title}</h4>
        <button
          type="button"
          onClick={() => setShowModels((prev) => !prev)}
          className="flex items-center gap-1 text-[0.6875rem] text-muted hover:text-[#CBD5E2] transition-colors"
          aria-expanded={showModels}
        >
          <span>{family.totalModelsCount} modelo(s) incluído(s)</span>
          {showModels ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
        </button>
      </div>

      <div className="space-y-2.5">
        {family.limits.map((limit: AntigravityWindowLimit, idx: number) => {
          const resetLabel = limit.resetsAt ? formatResetsIn(limit.resetsAt) : undefined
          return (
            <div
              key={`${limit.windowType}-${idx}`}
              className="flex items-center justify-between rounded-md bg-[#0f1722]/60 px-3 py-2 border border-line/40"
              data-testid={`drawer-pool-${family.id}-${limit.windowType}`}
            >
              <div>
                <p className="text-xs font-medium text-[#AEBBD0]">{limit.windowLabel}</p>
                {resetLabel ? (
                  <p className="text-[0.6875rem] text-[#68778C]">{resetLabel}</p>
                ) : null}
              </div>

              <div className="flex items-center gap-2.5">
                <span className="text-sm font-semibold text-[#CBD5E2]">
                  {t('antigravity.percentRemaining', {
                    percent: formatNumber(Math.round(limit.remainingPercent)),
                  })}
                </span>
                <CircularQuotaProgress percent={limit.remainingPercent} size={20} />
              </div>
            </div>
          )
        })}
      </div>

      {showModels ? (
        <div className="mt-2 space-y-1.5 pt-2 border-t border-line/30">
          <p className="text-[0.625rem] uppercase tracking-wider font-medium text-[#8290A5]">
            Modelos atendidos por esta franquia
          </p>
          <div className="space-y-1">
            {family.limits.flatMap((limit) => limit.models).map((model) => (
              <div
                key={model.id}
                className="flex items-center justify-between text-xs py-1 px-2 rounded bg-black/20"
              >
                <div className="min-w-0 pr-2">
                  <span className="font-medium text-[#CBD5E2]">{model.name}</span>
                  {model.id !== model.name ? (
                    <span className="ml-1.5 font-mono text-[0.625rem] text-muted truncate">{model.id}</span>
                  ) : null}
                </div>
                <span className="text-[0.6875rem] text-[#8290A5] shrink-0">
                  {model.remainingPercent != null
                    ? `${formatNumber(Math.round(model.remainingPercent))}%`
                    : '100%'}
                </span>
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  )
}

export function AntigravityQuotaDrawerSection({ quotas }: AntigravityQuotaDrawerSectionProps) {
  const { t } = useTranslation('overview')
  const families = categorizeAntigravityQuotas(quotas)
  const [showBackground, setShowBackground] = useState(false)

  const activeFamilies = families.filter((f) => f.id !== 'autocomplete_background')
  const backgroundGroup = families.find((f) => f.id === 'autocomplete_background')

  return (
    <div className="space-y-4" data-testid="antigravity-drawer-quotas">
      <div className="flex items-baseline justify-between">
        <div>
          <h3 className="text-xs font-semibold text-primary">{t('antigravity.drawerTitle')}</h3>
          <p className="text-[0.6875rem] text-muted">
            Agrupadas por franquia oficial de modelos (pools compartilhados)
          </p>
        </div>
        <span className="text-[0.625rem] text-muted font-mono">
          {quotas.length} modelo(s) reportado(s)
        </span>
      </div>

      <div className="space-y-3">
        {activeFamilies.map((family) => (
          <FamilyCard key={family.id} family={family} />
        ))}
      </div>

      {backgroundGroup ? (
        <div className="rounded-lg border border-line/50 bg-raised/30 p-2.5">
          <button
            type="button"
            onClick={() => setShowBackground((prev) => !prev)}
            className="flex w-full items-center justify-between text-xs text-muted hover:text-[#CBD5E2]"
            aria-expanded={showBackground}
          >
            <div className="flex items-center gap-1.5">
              {showBackground ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
              <span className="font-medium">{backgroundGroup.title}</span>
              <span className="text-[0.625rem] text-[#68778C]">
                {t('antigravity.backgroundModelsLine', { count: backgroundGroup.totalModelsCount })}
              </span>
            </div>
            <span className="text-[0.625rem]">{t('antigravity.autofill')}</span>
          </button>

          {showBackground ? (
            <div className="mt-2 space-y-1 pt-2 border-t border-line/20">
              {backgroundGroup.limits.flatMap((l) => l.models).map((model) => (
                <div
                  key={model.id}
                  className="flex items-center justify-between text-xs py-0.5 px-2 font-mono text-[0.625rem] text-muted"
                >
                  <span>{model.id}</span>
                  <span>{t('antigravity.hundredPercentRemaining')}</span>
                </div>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}


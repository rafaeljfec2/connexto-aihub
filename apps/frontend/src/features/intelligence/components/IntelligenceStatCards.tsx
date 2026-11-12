import { ArrowUpRight, Brain, CircleDot, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { LucideIcon } from 'lucide-react'

const CARD_IDS = [
  'currentRouting',
  'requestUnderstanding',
  'resultEvaluation',
  'learnedPolicies',
] as const

const CARD_ICONS: Record<(typeof CARD_IDS)[number], LucideIcon> = {
  currentRouting: ArrowUpRight,
  requestUnderstanding: Zap,
  resultEvaluation: CircleDot,
  learnedPolicies: Brain,
}

export function IntelligenceStatCards() {
  const { t } = useTranslation('intelligence')

  return (
    <div className="grid grid-cols-2 gap-2.5 lg:grid-cols-4">
      {CARD_IDS.map((cardId) => {
        const Icon = CARD_ICONS[cardId]
        return (
          <div
            key={cardId}
            className="rounded-[10px] border border-line bg-surface p-2.5 transition-colors hover:border-line/80 sm:p-3"
          >
            <div className="flex items-center justify-between">
              <span className="font-mono text-[0.5625rem] uppercase tracking-[0.08em] text-muted sm:text-[0.625rem]">
                {t(`statCards.${cardId}.label`)}
              </span>
              <Icon className="size-3.5 text-muted/70" aria-hidden />
            </div>
            <div className="mt-1 font-display text-sm font-bold text-primary sm:text-base">
              {t(`statCards.${cardId}.value`)}
            </div>
            <div className="mt-0.5 text-[0.625rem] leading-tight text-muted/90">
              {t(`statCards.${cardId}.note`)}
            </div>
          </div>
        )
      })}
    </div>
  )
}

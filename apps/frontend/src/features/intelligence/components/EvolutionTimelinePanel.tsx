import { useTranslation } from 'react-i18next'
import { cn } from '../../../shared/ui/cn'

const PHASE_IDS = ['01', '02', '03'] as const

export function EvolutionTimelinePanel() {
  const { t } = useTranslation('intelligence')

  return (
    <div className="rounded-[10px] border border-line bg-surface p-3">
      <div className="mb-2 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-xs font-semibold text-primary">{t('evolutionTimeline.title')}</h2>
          <p className="mt-0.5 text-[0.625rem] text-muted">{t('evolutionTimeline.subtitle')}</p>
        </div>
        <span className="shrink-0 rounded-full border border-warning/25 bg-warning/10 px-2 py-0.5 font-mono text-[0.5625rem] font-medium text-warning">
          {t('evolutionTimeline.badge')}
        </span>
      </div>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
        {PHASE_IDS.map((phaseId) => {
          const active = phaseId === '01'
          return (
            <div
              key={phaseId}
              className={cn(
                'rounded-lg border p-2 transition-colors',
                active ? 'border-cobalt/35 bg-cobalt/[0.04]' : 'border-line/70 bg-graphite/60',
              )}
            >
              <div className="font-mono text-[0.5625rem] uppercase tracking-wider text-muted">
                {phaseId} · {t(`evolutionTimeline.phases.${phaseId}.label`)}
              </div>
              <div className="mt-0.5 text-xs font-semibold text-primary">
                {t(`evolutionTimeline.phases.${phaseId}.title`)}
              </div>
              <p className="mt-0.5 text-[0.625rem] leading-relaxed text-muted">
                {t(`evolutionTimeline.phases.${phaseId}.desc`)}
              </p>
            </div>
          )
        })}
      </div>
    </div>
  )
}

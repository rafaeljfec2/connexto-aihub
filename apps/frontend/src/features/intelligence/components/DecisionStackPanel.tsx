import { useTranslation } from 'react-i18next'
import { cn } from '../../../shared/ui/cn'

const STEP_IDS = ['01', '02', '03', '04'] as const

export function DecisionStackPanel() {
  const { t } = useTranslation('intelligence')

  return (
    <div className="rounded-[10px] border border-line bg-surface p-3">
      <div className="mb-2 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-xs font-semibold text-primary">{t('decisionStack.title')}</h2>
          <p className="mt-0.5 text-[0.625rem] text-muted">{t('decisionStack.subtitle')}</p>
        </div>
        <span className="shrink-0 rounded-full border border-cobalt/25 bg-cobalt/10 px-2 py-0.5 font-mono text-[0.5625rem] font-medium text-cobalt">
          {t('decisionStack.badge')}
        </span>
      </div>

      <div className="grid gap-1.5">
        {STEP_IDS.map((stepId) => {
          const future = stepId === '04'
          return (
            <div
              key={stepId}
              className={cn(
                'flex items-center gap-2.5 rounded-lg border border-transparent px-2 py-1 transition-colors',
                future ? 'opacity-65' : 'hover:bg-white/[0.02]',
              )}
            >
              <span className="flex size-5 shrink-0 items-center justify-center rounded border border-line bg-raised font-mono text-[0.625rem] font-medium text-muted">
                {stepId}
              </span>
              <div className="min-w-0 flex-1">
                <div className="text-xs font-medium text-primary">{t(`decisionStack.steps.${stepId}.title`)}</div>
                <p className="text-[0.625rem] text-muted">{t(`decisionStack.steps.${stepId}.desc`)}</p>
              </div>
              <span
                className={cn(
                  'shrink-0 rounded px-1.5 py-0.5 font-mono text-[0.5625rem] font-medium uppercase',
                  future
                    ? 'border border-line bg-raised text-muted'
                    : 'border border-healthy/25 bg-healthy/10 text-healthy',
                )}
              >
                {t(`decisionStack.steps.${stepId}.state`)}
              </span>
            </div>
          )
        })}
      </div>

      <div className="mt-2 border-t border-line/60 pt-2 text-[0.625rem] leading-relaxed text-muted">
        {t('decisionStack.footer')}
      </div>
    </div>
  )
}

import { useTranslation } from 'react-i18next'
import { cn } from '../../../shared/ui/cn'

const SIGNAL_IDS = ['routing', 'outcomeLatency', 'providerUsage', 'contextQuality'] as const

export function SignalsPanel() {
  const { t } = useTranslation('intelligence')

  return (
    <div className="rounded-[10px] border border-line bg-surface p-3">
      <div className="mb-2 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-xs font-semibold text-primary">{t('signals.title')}</h2>
          <p className="mt-0.5 text-[0.625rem] text-muted">{t('signals.subtitle')}</p>
        </div>
        <span className="shrink-0 rounded-full border border-line bg-raised/80 px-2 py-0.5 font-mono text-[0.5625rem] text-muted">
          {t('signals.badge')}
        </span>
      </div>

      <div className="grid gap-1.5">
        {SIGNAL_IDS.map((signalId) => {
          const pending = signalId === 'contextQuality'
          return (
            <div
              key={signalId}
              className="flex items-center justify-between gap-3 rounded-lg px-2 py-1 transition-colors hover:bg-white/[0.02]"
            >
              <div className="flex min-w-0 items-center gap-2.5">
                <span className={`size-1.5 shrink-0 rounded-full ${pending ? 'bg-warning' : 'bg-healthy'}`} />
                <div className="min-w-0">
                  <div className="text-xs font-medium text-primary">{t(`signals.items.${signalId}.title`)}</div>
                  <div className="text-[0.625rem] text-muted">{t(`signals.items.${signalId}.sub`)}</div>
                </div>
              </div>
              <span
                className={cn(
                  'shrink-0 rounded px-1.5 py-0.5 font-mono text-[0.5625rem] font-medium uppercase',
                  pending
                    ? 'border border-warning/25 bg-warning/10 text-warning'
                    : 'border border-healthy/25 bg-healthy/10 text-healthy',
                )}
              >
                {pending ? t('signals.statusPending') : t('signals.statusExists')}
              </span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

import { AlertTriangle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export function IntelligenceNoticeBanner() {
  const { t } = useTranslation('intelligence')

  return (
    <div className="flex items-center justify-between gap-3 rounded-[10px] border border-warning/20 bg-warning/[0.04] px-3 py-2.5">
      <div className="flex min-w-0 items-center gap-3">
        <div className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-warning/10 text-warning">
          <AlertTriangle className="size-4" />
        </div>
        <div className="min-w-0">
          <strong className="text-xs font-semibold text-primary">{t('noticeBanner.title')}</strong>
          <p className="mt-0.5 text-[0.6875rem] leading-relaxed text-secondary">
            {t('noticeBanner.bodyLead')}{' '}
            <code className="font-mono text-[0.625rem] text-accent-light">aihub/auto</code>{' '}
            {t('noticeBanner.bodyTrail')}
          </p>
        </div>
      </div>
      <div className="flex shrink-0 flex-col items-end gap-1">
        <span className="font-mono text-[0.5625rem] uppercase tracking-wider text-muted">
          {t('noticeBanner.moduleStatus')}
        </span>
        <span className="inline-flex rounded-full border border-warning/25 bg-warning/10 px-2 py-0.5 font-mono text-[0.5625rem] font-medium text-warning">
          {t('noticeBanner.waitingE10')}
        </span>
      </div>
    </div>
  )
}

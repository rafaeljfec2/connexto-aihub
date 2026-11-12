import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { appPageClassName } from '../../../app/layouts/appShellLayout'
import { Button } from '../../../shared/ui/button'

const WINNER_STEP_IDS = ['explicitPair', 'providerOnly', 'modelOnly', 'nothing'] as const
const CONTENDER_LIST_KEYS = ['provider', 'catalog', 'capabilities', 'outputLimit'] as const

export function IntelligenceHelpPage() {
  const { t } = useTranslation('intelligence')

  return (
    <div className={appPageClassName('space-y-4', 'py-3', 'pb-8')}>
      <AppPageHeader
        actions={
          <Button asChild variant="ghost" size="sm">
            <Link to="/intelligence">{t('help.back')}</Link>
          </Button>
        }
      />

      <article className="space-y-4 text-[0.8125rem] leading-relaxed text-secondary">
        <p className="rounded-[10px] border border-line bg-surface px-4 py-3 text-primary">{t('help.intro')}</p>

        <section className="rounded-[10px] border border-line bg-surface px-4 py-3">
          <h2 className="text-[0.9375rem] font-semibold text-primary">{t('help.contenders.title')}</h2>
          <p className="mt-2">{t('help.contenders.intro')}</p>
          <ul className="mt-2 list-disc space-y-1.5 pl-5">
            {CONTENDER_LIST_KEYS.map((key) => (
              <li key={key}>{t(`help.contenders.list.${key}`)}</li>
            ))}
          </ul>
          <p className="mt-3">{t('help.contenders.footer')}</p>
        </section>

        <section className="rounded-[10px] border border-line bg-surface px-4 py-3">
          <h2 className="text-[0.9375rem] font-semibold text-primary">{t('help.winner.title')}</h2>
          <p className="mt-2">{t('help.winner.intro')}</p>
          <ol className="mt-2 list-decimal space-y-2 pl-5">
            {WINNER_STEP_IDS.map((stepId) => (
              <li key={stepId}>
                <strong className="font-semibold text-primary">{t(`help.winner.steps.${stepId}.title`)}</strong>{' '}
                {t(`help.winner.steps.${stepId}.body`)}
              </li>
            ))}
          </ol>
          <p className="mt-3">
            <code className="font-mono text-[0.75rem] text-primary">{t('help.winner.antigravityCode')}</code>{' '}
            {t('help.winner.antigravityNoteLead')}
          </p>
        </section>

        <section className="rounded-[10px] border border-line bg-surface px-4 py-3">
          <h2 className="text-[0.9375rem] font-semibold text-primary">{t('help.auto.title')}</h2>
          <p className="mt-2">
            {t('help.auto.bodyLead')}{' '}
            <code className="font-mono text-[0.75rem] text-primary">efficiency</code>{' '}
            {t('help.auto.bodyEfficiencyTrail')}{' '}
            <code className="font-mono text-[0.75rem] text-primary">intelligence</code>{' '}
            {t('help.auto.bodyIntelligenceTrail')}
          </p>
        </section>
      </article>
    </div>
  )
}

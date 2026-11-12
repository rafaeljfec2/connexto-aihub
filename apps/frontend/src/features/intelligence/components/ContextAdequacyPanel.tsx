import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '../../../shared/ui/cn'

const CONTEXT_IDS = ['code', 'reasoning', 'extraction', 'chat'] as const

const CONTEXT_GLYPHS: Record<(typeof CONTEXT_IDS)[number], string> = {
  code: '⌘',
  reasoning: '∴',
  extraction: '{ }',
  chat: '◌',
}

export function ContextAdequacyPanel() {
  const { t } = useTranslation('intelligence')
  const [selectedId, setSelectedId] = useState<string>(CONTEXT_IDS[0])
  const selectedName = t(`contextAdequacy.contexts.${selectedId as (typeof CONTEXT_IDS)[number]}.name`)

  return (
    <div className="rounded-[10px] border border-line bg-surface p-3">
      <div className="mb-2 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-xs font-semibold text-primary">{t('contextAdequacy.title')}</h2>
          <p className="mt-0.5 text-[0.625rem] text-muted">{t('contextAdequacy.subtitle')}</p>
        </div>
        <span className="shrink-0 rounded-full border border-line bg-raised/80 px-2 py-0.5 font-mono text-[0.5625rem] text-muted">
          {t('contextAdequacy.badge')}
        </span>
      </div>

      <div className="grid gap-1">
        <div className="flex justify-between px-2.5 py-1 font-mono text-[0.5625rem] uppercase tracking-wider text-muted">
          <span>{t('contextAdequacy.colContext')}</span>
          <span className="hidden sm:inline">{t('contextAdequacy.colEvidence')}</span>
          <span className="text-right">{t('contextAdequacy.colFit')}</span>
        </div>
        {CONTEXT_IDS.map((ctxId) => (
          <button
            key={ctxId}
            type="button"
            onClick={() => setSelectedId(ctxId)}
            className={cn(
              'flex items-center justify-between rounded-lg px-2.5 py-1.5 text-left transition-colors',
              selectedId === ctxId
                ? 'border border-line/80 bg-raised/90'
                : 'border border-transparent hover:bg-white/[0.03]',
            )}
          >
            <div className="flex items-center gap-2.5">
              <span className="flex size-5 shrink-0 items-center justify-center rounded bg-raised font-mono text-[0.6875rem] text-accent-light">
                {CONTEXT_GLYPHS[ctxId]}
              </span>
              <div>
                <div className="text-xs font-medium text-primary">{t(`contextAdequacy.contexts.${ctxId}.name`)}</div>
                <div className="text-[0.625rem] text-muted">{t(`contextAdequacy.contexts.${ctxId}.desc`)}</div>
              </div>
            </div>
            <span className="hidden text-[0.625rem] text-muted sm:inline">{t('contextAdequacy.noHistory')}</span>
            <span className="text-[0.625rem] font-medium text-warning">{t('contextAdequacy.awaitingE10')}</span>
          </button>
        ))}
      </div>

      <div className="mt-2 flex items-center justify-between border-t border-line/60 pt-2 text-[0.625rem] text-muted">
        <p className="truncate">
          {t('contextAdequacy.selectedFooterPrefix')}{' '}
          <strong className="font-semibold text-primary">{selectedName}</strong>
          {t('contextAdequacy.selectedFooterSuffix')}
        </p>
        <b className="shrink-0 font-mono text-[0.5625rem] font-bold tracking-wider text-primary">
          {t('contextAdequacy.noRanking')}
        </b>
      </div>
    </div>
  )
}

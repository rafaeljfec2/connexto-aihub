import { Activity, Power, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

type ProviderActionsProps = {
  enabled: boolean
  validatePending: boolean
  togglePending: boolean
  actionsDisabled: boolean
  onValidate: () => void
  onToggleEnabled: () => void
}

export function ProviderActions({
  enabled,
  validatePending,
  togglePending,
  actionsDisabled,
  onValidate,
  onToggleEnabled,
}: ProviderActionsProps) {
  const { t } = useTranslation('providers')

  return (
    <div className="provider-actions flex flex-wrap items-center gap-1.5">
      <button
        type="button"
        disabled={actionsDisabled}
        onClick={onValidate}
        className="inline-flex items-center gap-1 rounded-md border border-line bg-surface/80 px-2.5 py-1 text-[0.6875rem] font-medium text-secondary hover:border-cobalt/60 hover:bg-[#1E2638] hover:text-primary transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-50"
      >
        {validatePending ? (
          <>
            <RefreshCw className="size-3 animate-spin text-cobalt" />
            <span>{t('actions.testing')}</span>
          </>
        ) : (
          <>
            <Activity className="size-3 text-cobalt" />
            <span>{t('actions.testProbe')}</span>
          </>
        )}
      </button>

      <button
        type="button"
        disabled={actionsDisabled}
        onClick={onToggleEnabled}
        className={`inline-flex items-center gap-1 rounded-md border px-2.5 py-1 text-[0.6875rem] font-medium transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 ${
          enabled
            ? 'border-line bg-surface/80 text-secondary hover:border-amber-500/50 hover:bg-amber-500/10 hover:text-amber-200'
            : 'border-emerald-500/40 bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20'
        }`}
      >
        {togglePending ? (
          <>
            <RefreshCw className="size-3 animate-spin" />
            <span>{t('actions.updating')}</span>
          </>
        ) : (
          <>
            <Power className="size-3" />
            <span>{enabled ? t('actions.disableProvider') : t('actions.enableProvider')}</span>
          </>
        )}
      </button>
    </div>
  )
}

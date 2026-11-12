import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Panel } from '../../../shared/ui/panel'
import { LocaleSelector } from '../components/LocaleSelector'

export function SettingsPage() {
  const { t } = useTranslation('account')

  return (
    <div className="space-y-4 p-1">
      <div>
        <h1 className="text-lg font-semibold text-primary">{t('settingsTitle')}</h1>
        <p className="text-sm text-secondary">{t('settingsLead')}</p>
      </div>
      <Panel className="max-w-lg space-y-6 border-line/80 p-6">
        <LocaleSelector />
      </Panel>
      <Panel className="max-w-lg space-y-4 border-line/80 p-6">
        <p className="text-sm text-secondary">{t('passwordPanelLead')}</p>
        <Link className="text-sm text-cobalt hover:underline" to="/forgot-password">
          {t('resetPasswordLink')}
        </Link>
      </Panel>
    </div>
  )
}

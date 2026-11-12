import { useTranslation } from 'react-i18next'

export function BackendUnavailableBanner() {
  const { t } = useTranslation('overview')
  return (
    <div className="banner banner-error" role="alert">
      {t('backendUnavailable')}
    </div>
  )
}

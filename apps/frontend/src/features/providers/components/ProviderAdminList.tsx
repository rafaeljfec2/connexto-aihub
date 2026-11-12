import { useTranslation } from 'react-i18next'
import type { AdminProvider } from '../types'
import { ProviderAdminCard } from './ProviderAdminCard'

type ProviderAdminListProps = {
  providers: AdminProvider[]
  isLoading: boolean
  isError: boolean
}

export function ProviderAdminList({ providers, isLoading, isError }: ProviderAdminListProps) {
  const { t } = useTranslation('providers')

  if (isLoading) {
    return <p>{t('admin.loading')}</p>
  }

  if (isError) {
    return (
      <div className="banner banner-error" role="alert">
        {t('admin.unavailable')}
      </div>
    )
  }

  if (providers.length === 0) {
    return <p className="empty-state">{t('admin.empty')}</p>
  }

  return (
    <div className="provider-admin-list">
      {providers.map((provider) => (
        <ProviderAdminCard key={provider.id} provider={provider} />
      ))}
    </div>
  )
}

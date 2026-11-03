import type { AdminProvider } from '../types'
import { ProviderAdminCard } from './ProviderAdminCard'

type ProviderAdminListProps = {
  providers: AdminProvider[]
  isLoading: boolean
  isError: boolean
}

export function ProviderAdminList({ providers, isLoading, isError }: ProviderAdminListProps) {
  if (isLoading) {
    return <p>Loading providers…</p>
  }

  if (isError) {
    return (
      <div className="banner banner-error" role="alert">
        Provider administration is unavailable. Ensure PostgreSQL persistence is enabled and the admin API is reachable.
      </div>
    )
  }

  if (providers.length === 0) {
    return <p className="empty-state">No providers configured yet.</p>
  }

  return (
    <div className="provider-admin-list">
      {providers.map((provider) => (
        <ProviderAdminCard key={provider.id} provider={provider} />
      ))}
    </div>
  )
}

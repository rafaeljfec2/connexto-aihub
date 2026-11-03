import { useQuery } from '@tanstack/react-query'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { listProviders, providersQueryKeys } from '../api/queries'
import { ProviderAdminList } from '../components/ProviderAdminList'

export function ProvidersPage() {
  const providers = useQuery({
    queryKey: providersQueryKeys.list,
    queryFn: listProviders,
    refetchInterval: false,
    retry: 1,
  })

  return (
    <div className="page providers-page">
      <AppPageHeader />
      <ProviderAdminList
        providers={providers.data?.providers ?? []}
        isLoading={providers.isLoading}
        isError={providers.isError}
      />
    </div>
  )
}

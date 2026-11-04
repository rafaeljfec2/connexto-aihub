import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ShieldCheck } from 'lucide-react'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { MasterDetailLayout } from '../../../shared/ui'
import { listProviders, providersQueryKeys } from '../api/queries'
import { ProviderDetailPanel } from '../components/ProviderDetailPanel'
import { ProviderGovernanceMetrics } from '../components/ProviderGovernanceMetrics'
import { ProviderManageDrawer } from '../components/ProviderManageDrawer'
import { ProviderQuotaTable } from '../components/ProviderQuotaTable'
import type { AdminProvider } from '../types'

export function ProvidersPage() {
  const providersQuery = useQuery({
    queryKey: providersQueryKeys.list,
    queryFn: listProviders,
    refetchInterval: false,
    retry: 1,
  })

  const providers = providersQuery.data?.providers ?? []
  const [selectedProvider, setSelectedProvider] = useState<AdminProvider | null>(null)
  const [manageTarget, setManageTarget] = useState<AdminProvider | null>(null)

  const activeSelected =
    selectedProvider !== null
      ? providers.find((p) => p.id === selectedProvider.id) ?? providers[0] ?? null
      : providers[0] ?? null

  return (
    <div className="page providers-page space-y-4">
      <AppPageHeader />

      <ProviderGovernanceMetrics providers={providers} />

      <MasterDetailLayout
        master={
          <ProviderQuotaTable
            providers={providers}
            selectedId={activeSelected?.id ?? null}
            onSelect={(p) => setSelectedProvider(p)}
            onOpenManage={(p) => setManageTarget(p)}
            isLoading={providersQuery.isLoading}
          />
        }
        detail={
          <ProviderDetailPanel
            provider={activeSelected}
            onOpenManage={(p) => setManageTarget(p)}
          />
        }
      />

      <div className="flex items-center gap-2.5 rounded-lg border border-cobalt/20 bg-cobalt/5 p-3 text-[11px] text-secondary">
        <ShieldCheck className="size-4 shrink-0 text-accent-light" />
        <p>
          <strong className="text-primary">Leitura confiável:</strong> cada conexão reporta
          apenas dados autenticados vindos do gateway ou provedor. Quando o provedor não reportar
          franquias em tempo real, indicamos expressamente &quot;Sem leitura&quot; — nunca saldos estimados.
        </p>
      </div>

      <ProviderManageDrawer
        provider={manageTarget}
        open={manageTarget !== null}
        onClose={() => setManageTarget(null)}
      />
    </div>
  )
}

import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { RefreshCw, ShieldCheck } from 'lucide-react'
import { useAppPageMeta } from '../../../app/layouts/AppPageHeader'
import { appPageClassName } from '../../../app/layouts/appShellLayout'
import { getProviderQuotas, listProviders, providersQueryKeys } from '../api/queries'
import { ProviderDetailPanel } from '../components/ProviderDetailPanel'
import { ProviderGovernanceMetrics } from '../components/ProviderGovernanceMetrics'
import { ProviderManageDrawer } from '../components/ProviderManageDrawer'
import { ProviderQuotaTable } from '../components/ProviderQuotaTable'
import { primaryQuotaSnapshot } from '../lib/quotaPresentation'
import type { AdminProvider } from '../types'

export function ProvidersPage() {
  const meta = useAppPageMeta()
  const providersQuery = useQuery({
    queryKey: providersQueryKeys.list,
    queryFn: listProviders,
    refetchInterval: false,
    retry: 1,
  })
  const quotasQuery = useQuery({
    queryKey: providersQueryKeys.quotas,
    queryFn: getProviderQuotas,
    refetchInterval: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retry: 1,
  })

  const providers = providersQuery.data?.providers ?? []
  const snapshots = quotasQuery.data?.snapshots ?? []
  const [selectedProvider, setSelectedProvider] = useState<AdminProvider | null>(null)
  const [manageTarget, setManageTarget] = useState<AdminProvider | null>(null)

  const activeSelected =
    selectedProvider !== null
      ? providers.find((provider) => provider.id === selectedProvider.id) ?? providers[0] ?? null
      : providers[0] ?? null

  const activeSnapshot = useMemo(() => {
    if (!activeSelected) return undefined
    return primaryQuotaSnapshot(activeSelected.id, snapshots)
  }, [activeSelected, snapshots])

  const isRefreshing = providersQuery.isFetching || quotasQuery.isFetching

  return (
    <div className={appPageClassName('providers-page')}>
      <header className="providers-pagehead">
        <div className="min-w-0">
          <p className="providers-page-eyebrow">{meta.eyebrow}</p>
          <h1 className="providers-page-title">{meta.title}</h1>
          {meta.lead ? <p className="providers-page-lead">{meta.lead}</p> : null}
        </div>
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          <span className="overview-backend-tag">DADOS DO BACKEND</span>
          <button
            type="button"
            className="providers-primary-action"
            onClick={() => {
              void providersQuery.refetch()
              void quotasQuery.refetch()
            }}
            disabled={isRefreshing}
            aria-label="Atualizar franquias"
          >
            <RefreshCw className={`mr-1 inline size-3.5 ${isRefreshing ? 'animate-spin' : ''}`} aria-hidden />
            Atualizar
          </button>
          <button
            type="button"
            className="providers-primary-action cursor-not-allowed"
            disabled
            title="Cadastro de novos provedores é feito na configuração do gateway, fora desta tela."
          >
            ＋ Adicionar provedor
          </button>
        </div>
      </header>

      <ProviderGovernanceMetrics providers={providers} snapshots={snapshots} />

      <div className="providers-layout">
        <ProviderQuotaTable
          providers={providers}
          snapshots={snapshots}
          selectedId={activeSelected?.id ?? null}
          onSelect={(provider) => setSelectedProvider(provider)}
          isLoading={providersQuery.isLoading || quotasQuery.isLoading}
        />
        <ProviderDetailPanel
          provider={activeSelected}
          snapshot={activeSnapshot}
          onOpenManage={(provider) => setManageTarget(provider)}
        />
      </div>

      <div className="providers-page-footer">
        <div className="providers-bottominfo">
          <ShieldCheck className="size-3.5 shrink-0 text-accent-light" aria-hidden />
          <p>
            <strong className="text-primary">Leitura confiável:</strong> sem leitura de quota quando
            o provedor não reportar dados — nunca saldo estimado. Utilização por execução permanece
            em Execuções (E09.5), separada desta franquia oficial.
          </p>
        </div>
        <footer className="providers-page-footnote">
          <span>FONTES · PROVIDER-REPORTED · GATEWAY-METERED · MANUAL ALLOWANCE</span>
          <span>CONNEXTO AI HUB / PROVIDER QUOTAS</span>
        </footer>
      </div>

      <ProviderManageDrawer
        provider={manageTarget}
        open={manageTarget !== null}
        onClose={() => setManageTarget(null)}
      />
    </div>
  )
}

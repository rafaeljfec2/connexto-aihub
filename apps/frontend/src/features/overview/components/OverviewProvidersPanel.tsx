import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Panel } from '../../../shared/ui'
import type { ProviderQuotaSnapshot } from '../types/quota'
import { accountName, ProviderQuotaAccountRow } from './ProviderQuotaAccountRow'
import { ProviderQuotaDetailDrawer } from './ProviderQuotaDetailDrawer'

export type OverviewProvidersPanelProps = Readonly<{
  snapshots?: ProviderQuotaSnapshot[]
  isLoading?: boolean
  isError?: boolean
}>

type AccountSnapshots = Readonly<{
  accountId: string
  snapshots: ProviderQuotaSnapshot[]
}>

function groupByAccount(snapshots: ProviderQuotaSnapshot[]): AccountSnapshots[] {
  const groups = new Map<string, ProviderQuotaSnapshot[]>()
  for (const snapshot of snapshots) {
    const current = groups.get(snapshot.account_id) ?? []
    current.push(snapshot)
    groups.set(snapshot.account_id, current)
  }
  return Array.from(groups, ([accountId, accountSnapshots]) => ({
    accountId,
    snapshots: accountSnapshots,
  }))
}

export function OverviewProvidersPanel({
  snapshots = [],
  isLoading = false,
  isError = false,
}: OverviewProvidersPanelProps) {
  const accounts = groupByAccount(snapshots)
  const [selectedAccountId, setSelectedAccountId] = useState<string | null>(null)

  const selected = useMemo(
    () => accounts.find((account) => account.accountId === selectedAccountId) ?? null,
    [accounts, selectedAccountId],
  )

  return (
    <Panel className="overview-panel">
      <div className="overview-panel-head">
        <div>
          <h3 className="text-xs font-semibold text-[#E7ECF5]">Consumo de franquias</h3>
          <p className="mt-0.5 text-[10px] text-[#8D9AAF]">Resumo bruto por conta</p>
        </div>
        <Link to="/providers" className="overview-panel-link whitespace-nowrap">
          Ver franquias ↗
        </Link>
      </div>

      {isLoading && accounts.length === 0 ? <QuotaLoadingState /> : null}
      {!isLoading && isError && accounts.length === 0 ? <QuotaErrorState /> : null}
      {!isLoading && !isError && accounts.length === 0 ? <QuotaEmptyState /> : null}
      {isError && accounts.length > 0 ? <QuotaRefreshWarning /> : null}
      {!isLoading && accounts.length > 0 ? (
        <div className="max-h-80 min-h-0 flex-1 space-y-2 overflow-y-auto pt-2">
          {accounts.map((account) => (
            <ProviderQuotaAccountRow
              key={account.accountId}
              snapshots={account.snapshots}
              selected={selectedAccountId === account.accountId}
              onSelect={(accountId) => setSelectedAccountId(accountId)}
            />
          ))}
        </div>
      ) : null}

      <ProviderQuotaDetailDrawer
        open={selected != null}
        onOpenChange={(open) => {
          if (!open) setSelectedAccountId(null)
        }}
        accountLabel={selected ? accountName(selected.snapshots) : 'Provedor'}
        snapshots={selected?.snapshots ?? []}
      />
    </Panel>
  )
}

function QuotaLoadingState() {
  return (
    <div className="space-y-3 py-3 animate-pulse" data-testid="quota-loading">
      <div className="h-8 rounded bg-raised" />
      <div className="h-8 rounded bg-raised" />
    </div>
  )
}

function QuotaEmptyState() {
  return (
    <div className="py-4 text-center text-xs text-muted" data-testid="quota-empty">
      Nenhuma conta de provedor configurada no momento.
    </div>
  )
}

function QuotaErrorState() {
  return (
    <p className="py-4 text-center text-xs text-warning" role="alert">
      Falha ao consultar os dados oficiais do provedor. Tente atualizar.
    </p>
  )
}

function QuotaRefreshWarning() {
  return (
    <p className="pt-2 text-[10px] text-warning" role="status">
      Falha ao atualizar; exibindo a última leitura disponível.
    </p>
  )
}

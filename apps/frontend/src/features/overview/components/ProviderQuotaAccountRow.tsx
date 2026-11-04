import type { DataScope, ProviderQuotaSnapshot } from '../types/quota'
import { OverviewQuotaMetricRow } from './OverviewQuotaMetricRow'

type ProviderQuotaAccountRowProps = Readonly<{
  snapshots: readonly ProviderQuotaSnapshot[]
}>

const keySummaryMetrics = new Set(['usage', 'limit', 'limit_remaining', 'free_model_daily_requests'])
const accountSummaryMetrics = new Set(['total_credits', 'total_usage'])

function accountName(snapshots: readonly ProviderQuotaSnapshot[]): string {
  const primary = snapshots.find((snapshot) => snapshot.scope === 'api_key') ?? snapshots[0]
  if (primary === undefined) return 'Provedor'
  return [primary.account_label, primary.provider_name, primary.account_id]
    .find((name) => name.trim().length > 0) ?? primary.account_id
}

function initials(label: string): string {
  const parts = label.trim().replace(/[^a-zA-Z0-9]/g, ' ').split(/\s+/).filter(Boolean)
  if (parts.length >= 2) return `${parts[0][0]}${parts[1][0]}`.toUpperCase()
  return label.slice(0, 2).toUpperCase()
}

function scopeLabel(scope: DataScope): string {
  if (scope === 'api_key') return 'Chave API'
  if (scope === 'account') return 'Conta'
  return `Escopo ${scope}`
}

function statusLabel(status: ProviderQuotaSnapshot['status']): string {
  if (status === 'available') return 'Disponível'
  if (status === 'rate_limited') return 'Limite da API atingido'
  if (status === 'timeout') return 'Tempo limite excedido'
  if (status === 'unconfigured_credential') return 'Credencial ausente'
  if (status === 'unauthorized') return 'Sem permissão'
  return 'Sem leitura'
}

function statusMessage(snapshot: ProviderQuotaSnapshot): string {
  if (snapshot.status === 'unconfigured_credential' && snapshot.scope === 'account') {
    return 'Management API Key não configurada'
  }
  if (snapshot.status === 'unconfigured_credential') return 'Credencial da chave API não configurada'
  if (snapshot.status === 'unauthorized') return 'A credencial não tem permissão para esta leitura'
  if (snapshot.status === 'unsupported') return 'Este provedor não oferece uma leitura oficial configurada'
  if (snapshot.status === 'rate_limited') return 'O provedor limitou a consulta; tente atualizar depois'
  if (snapshot.status === 'timeout') return 'A consulta ao provedor excedeu o tempo limite'
  if (snapshot.status === 'error') return 'Não foi possível consultar os dados oficiais'
  return 'O provedor não retornou campos do resumo'
}

function isSummaryMetric(scope: DataScope, metric: string): boolean {
  if (scope === 'api_key') return keySummaryMetrics.has(metric)
  if (scope === 'account') return accountSummaryMetrics.has(metric)
  return false
}

function visibleSnapshots(snapshots: readonly ProviderQuotaSnapshot[]): readonly ProviderQuotaSnapshot[] {
  const hasAccountSnapshot = snapshots.some((snapshot) => snapshot.scope === 'account')
  if (!hasAccountSnapshot) return snapshots
  return snapshots.filter((snapshot) => snapshot.scope !== 'api_key')
}

function availableBalance(snapshot: ProviderQuotaSnapshot): number | undefined {
  if (snapshot.scope !== 'account' || snapshot.status !== 'available') return undefined

  const credits = snapshot.quotas.find((metric) => metric.metric === 'total_credits')
  const usage = snapshot.quotas.find((metric) => metric.metric === 'total_usage')
  if (credits?.limit == null || usage?.used == null || credits.unit !== 'USD' || usage.unit !== 'USD') {
    return undefined
  }

  return credits.limit - usage.used
}

function formatAvailableBalance(value: number): string {
  const amount = value.toLocaleString('pt-BR', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  return `US$ ${amount}`
}

function ScopeSummary({ snapshot }: Readonly<{ snapshot: ProviderQuotaSnapshot }>) {
  const metrics = snapshot.quotas.filter((metric) => isSummaryMetric(snapshot.scope, metric.metric))
  const balance = availableBalance(snapshot)

  return (
    <section className="border-t border-[rgb(180_194_218_/_0.08)] pt-1.5 first:border-t-0 first:pt-0" data-testid={`quota-scope-${snapshot.account_id}-${snapshot.scope}`}>
      <header className="mb-1 flex items-center justify-between gap-2 text-[9px]">
        <span className="font-medium text-[#AEBBD0]">{scopeLabel(snapshot.scope)}</span>
        <span className="text-[#8290A5]">{statusLabel(snapshot.status)}</span>
      </header>
      {snapshot.status === 'available' && metrics.length > 0 ? (
        <div className="space-y-1">
          {metrics.map((metric) => (
            <OverviewQuotaMetricRow
              key={metric.metric}
              accountId={snapshot.account_id}
              snapshotScope={snapshot.scope}
              metric={metric}
            />
          ))}
          {balance != null ? (
            <div className="flex items-baseline justify-between gap-2" data-testid={`quota-balance-${snapshot.account_id}`}>
              <span className="text-[9px] font-medium text-[#CBD5E2]">Saldo disponível</span>
              <span className="text-[9px] text-[#A7B3C3]">{formatAvailableBalance(balance)}</span>
            </div>
          ) : null}
        </div>
      ) : (
        <p className="text-[9px] text-muted" role="status">{statusMessage(snapshot)}</p>
      )}
      <p className="mt-1 text-[8px] text-[#68778C]">Consultado em: {snapshot.source.fetched_at}</p>
    </section>
  )
}

export function ProviderQuotaAccountRow({ snapshots }: ProviderQuotaAccountRowProps) {
  const firstSnapshot = snapshots[0]
  if (firstSnapshot === undefined) return null
  const name = accountName(snapshots)
  const summaries = visibleSnapshots(snapshots)

  return (
    <article
      className="rounded-lg border border-[rgb(180_194_218_/_0.09)] bg-[#182333]/70 p-2"
      data-testid={`provider-summary-${firstSnapshot.account_id}`}
    >
      <header className="mb-1.5 flex items-center gap-2">
        <span className="overview-provider-icon">{initials(name)}</span>
        <span className="truncate text-[11px] font-semibold text-[#E1E8F3]">{name}</span>
      </header>
      <div className="space-y-1.5">
        {summaries.map((snapshot) => (
          <ScopeSummary key={snapshot.scope} snapshot={snapshot} />
        ))}
      </div>
    </article>
  )
}

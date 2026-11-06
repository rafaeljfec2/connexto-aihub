import { AiBrandIcon, hubProviderIconSlug } from '../../../shared/branding'
import {
  goPlusChipPercent,
  goPlusMetersFromSnapshot,
  productUsageTitle,
} from '../lib/goPlusMeters'
import { formatQuotaFetchedAt } from '../lib/formatQuotaDate'
import type { DataScope, ProviderQuotaSnapshot } from '../types/quota'
import { OverviewQuotaMetricRow } from './OverviewQuotaMetricRow'

type ProviderQuotaAccountRowProps = Readonly<{
  snapshots: readonly ProviderQuotaSnapshot[]
  selected?: boolean
  onSelect?: (accountId: string) => void
}>

const keySummaryMetrics = new Set(['usage', 'limit', 'limit_remaining', 'free_model_daily_requests'])
const accountSummaryMetrics = new Set(['total_credits', 'total_usage'])

export function accountName(snapshots: readonly ProviderQuotaSnapshot[]): string {
  const primary =
    snapshots.find((snapshot) => snapshot.scope === 'subscription') ??
    snapshots.find((snapshot) => snapshot.scope === 'api_key') ??
    snapshots[0]
  if (primary === undefined) return 'Provedor'
  const product = primary.workspace_name?.trim()
  const base =
    [primary.account_label, primary.provider_name, primary.account_id].find(
      (name) => name.trim().length > 0,
    ) ?? primary.account_id
  return product ? `${base} (${product})` : base
}

function scopeLabel(scope: DataScope): string {
  if (scope === 'api_key') return 'Chave API'
  if (scope === 'account') return 'Conta'
  if (scope === 'subscription') return 'Assinatura'
  if (scope === 'workspace') return 'Workspace'
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

function goSummaryLine(snapshots: readonly ProviderQuotaSnapshot[]): string | undefined {
  const subscription = snapshots.find(
    (snapshot) =>
      snapshot.status === 'available' &&
      (snapshot.scope === 'subscription' || snapshot.scope === 'workspace'),
  )
  if (!subscription) return undefined
  const meters = goPlusMetersFromSnapshot(subscription)
  const percent = goPlusChipPercent(meters)
  if (percent == null) return undefined
  return `${productUsageTitle(subscription.workspace_name).replace(/ usage$/i, '')} ${percent.toLocaleString('pt-BR')}% used`
}

function ScopeSummary({ snapshot }: Readonly<{ snapshot: ProviderQuotaSnapshot }>) {
  const metrics = snapshot.quotas.filter((metric) => isSummaryMetric(snapshot.scope, metric.metric))
  const balance = availableBalance(snapshot)

  return (
    <section
      className="border-t border-[rgb(180_194_218_/_0.08)] pt-1.5 first:border-t-0 first:pt-0"
      data-testid={`quota-scope-${snapshot.account_id}-${snapshot.scope}`}
    >
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
            <div
              className="flex items-baseline justify-between gap-2"
              data-testid={`quota-balance-${snapshot.account_id}`}
            >
              <span className="text-[9px] font-medium text-[#CBD5E2]">Saldo disponível</span>
              <span className="text-[9px] text-[#A7B3C3]">{formatAvailableBalance(balance)}</span>
            </div>
          ) : null}
        </div>
      ) : (
        <p className="text-[9px] text-muted" role="status">
          {statusMessage(snapshot)}
        </p>
      )}
      <p className="mt-1 text-[8px] text-[#68778C]">
        Consultado em: {formatQuotaFetchedAt(snapshot.source.fetched_at)}
      </p>
    </section>
  )
}

export function ProviderQuotaAccountRow({
  snapshots,
  selected = false,
  onSelect,
}: ProviderQuotaAccountRowProps) {
  const firstSnapshot = snapshots[0]
  if (firstSnapshot === undefined) return null
  const name = accountName(snapshots)
  const summaries = visibleSnapshots(snapshots)
  const goLine = goSummaryLine(snapshots)
  const isGoAccount = goLine != null
  const primary = summaries[0]
  const clickable = onSelect != null

  return (
    <article
      className={`rounded-lg border border-[rgb(180_194_218_/_0.09)] bg-[#182333]/70 p-2 ${
        clickable ? 'cursor-pointer transition-colors hover:bg-white/[0.03]' : ''
      } ${selected ? 'ring-1 ring-cobalt/50' : ''}`}
      data-testid={`provider-summary-${firstSnapshot.account_id}`}
      role={clickable ? 'button' : undefined}
      tabIndex={clickable ? 0 : undefined}
      onClick={
        clickable
          ? () => {
              onSelect(firstSnapshot.account_id)
            }
          : undefined
      }
      onKeyDown={
        clickable
          ? (event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                onSelect(firstSnapshot.account_id)
              }
            }
          : undefined
      }
    >
      <header className="flex items-center gap-2">
        <AiBrandIcon
          slug={hubProviderIconSlug(firstSnapshot.provider_id)}
          label={name}
          size={28}
          className="overview-provider-icon"
        />
        <div className="min-w-0 flex-1">
          <span className="block truncate text-[11px] font-semibold text-[#E1E8F3]">{name}</span>
          {isGoAccount ? (
            <div className="mt-0.5 flex items-center justify-between gap-2 text-[9px]">
              <span className="truncate text-[#AEBBD0]" data-testid={`go-plus-summary-${firstSnapshot.account_id}`}>
                {goLine}
              </span>
              <span className="shrink-0 text-[#8290A5]">
                {primary ? statusLabel(primary.status) : 'Sem leitura'}
              </span>
            </div>
          ) : null}
        </div>
      </header>
      {!isGoAccount ? (
        <div className="mt-1.5 space-y-1.5">
          {summaries.map((snapshot) => (
            <ScopeSummary key={snapshot.scope} snapshot={snapshot} />
          ))}
        </div>
      ) : (
        <p className="mt-1 text-[8px] text-[#68778C]">
          Clique para ver 5h / semanal / mensal
          {primary ? ` · Consultado em: ${formatQuotaFetchedAt(primary.source.fetched_at)}` : ''}
        </p>
      )}
    </article>
  )
}

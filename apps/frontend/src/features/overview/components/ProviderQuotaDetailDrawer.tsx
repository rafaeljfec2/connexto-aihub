import * as DialogPrimitive from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { AiBrandIcon, hubProviderIconSlug } from '../../../shared/branding'
import {
  consoleBudgetsFromSnapshot,
  goPlusMetersFromSnapshot,
  productUsageTitle,
} from '../lib/goPlusMeters'
import { formatQuotaFetchedAt } from '../lib/formatQuotaDate'
import { formatResetsIn } from '../lib/formatResetsIn'
import type { ProviderQuotaSnapshot } from '../types/quota'
import { ConsoleBudgetsPanel } from './ConsoleBudgetsPanel'
import { GoPlusUsagePanel } from './GoPlusUsagePanel'
import { WorkspaceUsageChart } from './WorkspaceUsageChart'
import '../provider-quota-drawer.css'

type ProviderQuotaDetailDrawerProps = Readonly<{
  open: boolean
  onOpenChange: (open: boolean) => void
  accountLabel: string
  snapshots: readonly ProviderQuotaSnapshot[]
}>

function primarySnapshot(
  snapshots: readonly ProviderQuotaSnapshot[],
): ProviderQuotaSnapshot | undefined {
  return (
    snapshots.find((snapshot) => snapshot.scope === 'subscription') ??
    snapshots.find((snapshot) => snapshot.scope === 'account') ??
    snapshots.find((snapshot) => snapshot.scope === 'api_key') ??
    snapshots[0]
  )
}

export function ProviderQuotaDetailDrawer({
  open,
  onOpenChange,
  accountLabel,
  snapshots,
}: ProviderQuotaDetailDrawerProps) {
  const primary = primarySnapshot(snapshots)
  const goMeters = primary ? goPlusMetersFromSnapshot(primary) : []
  const budgets = primary ? consoleBudgetsFromSnapshot(primary) : []
  const status = primary?.status === 'available' ? 'Disponível' : 'Sem leitura'

  return (
    <DialogPrimitive.Root modal={false} open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="provider-quota-drawer-backdrop" />
        <DialogPrimitive.Content
          className="provider-quota-drawer"
          data-testid="provider-quota-drawer"
          onInteractOutside={(event) => event.preventDefault()}
        >
          <header className="provider-quota-drawer-head">
            <div className="flex min-w-0 items-start gap-2.5">
              {primary ? (
                <AiBrandIcon
                  slug={hubProviderIconSlug(primary.provider_id)}
                  label={accountLabel}
                  size={32}
                  className="overview-provider-icon size-8 shrink-0"
                />
              ) : null}
              <div className="min-w-0">
                <p className="provider-quota-drawer-eyebrow">FRANQUIA DO PROVEDOR</p>
                <DialogPrimitive.Title className="provider-quota-drawer-title">
                  {accountLabel}
                </DialogPrimitive.Title>
                <DialogPrimitive.Description className="provider-quota-drawer-description">
                  {primary
                    ? `${primary.scope} · ${status} · Consultado em ${formatQuotaFetchedAt(primary.source.fetched_at)}`
                    : 'Selecione um provedor para inspecionar a franquia oficial.'}
                </DialogPrimitive.Description>
              </div>
            </div>
            <DialogPrimitive.Close className="provider-quota-close" aria-label="Fechar detalhe da franquia">
              <X aria-hidden="true" className="size-4" />
            </DialogPrimitive.Close>
          </header>

          <div className="provider-quota-drawer-scroll">
            {primary?.provider_id === 'antigravity' && primary.status === 'available' ? (
              <div className="space-y-3" data-testid="antigravity-drawer-quotas">
                <div className="flex items-baseline justify-between">
                  <h4 className="text-xs font-semibold text-primary">Cotas por Modelo</h4>
                  <span className="text-[10px] text-muted">
                    {primary.quotas.length} modelo(s) reportado(s)
                  </span>
                </div>
                <div className="space-y-1.5 rounded-md border border-line bg-raised p-2.5">
                  {primary.quotas.map((q) => (
                    <div
                      key={q.metric}
                      className="flex items-center justify-between text-xs py-1 border-b border-line/40 last:border-b-0"
                    >
                      <div>
                        <span className="font-medium text-[#E1E8F3]">{q.label || q.metric}</span>
                        {q.metric !== q.label ? (
                          <span className="ml-1.5 text-[10px] text-muted">{q.metric}</span>
                        ) : null}
                      </div>
                      <div className="text-right">
                        <span className="text-[11px] font-semibold text-[#CBD5E2]">
                          {q.remaining_percent != null
                            ? `${q.remaining_percent.toLocaleString('pt-BR')}% restante`
                            : '—'}
                        </span>
                        {q.resets_at ? (
                          <span className="ml-2 text-[10px] text-muted">
                            {formatResetsIn(q.resets_at) ?? ''}
                          </span>
                        ) : null}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : goMeters.length > 0 && primary ? (
              <GoPlusUsagePanel
                meters={goMeters}
                workspaceName={primary.workspace_name}
                testIdPrefix="go-plus-drawer"
              />
            ) : (
              <p className="text-xs text-muted" role="status">
                {primary?.error?.message ??
                  (primary?.provider_id === 'antigravity'
                    ? 'Sem leitura oficial de cotas Antigravity.'
                    : 'Sem leitura oficial de franquia Go para esta conta.')}
              </p>
            )}
            {primary?.usage_chart && primary.usage_chart.points.length > 0 ? (
              <WorkspaceUsageChart
                chart={primary.usage_chart}
                testIdPrefix="go-plus-drawer"
              />
            ) : null}
            {budgets.length > 0 ? (
              <ConsoleBudgetsPanel budgets={budgets} testIdPrefix="go-plus-drawer-budget" />
            ) : null}
            {goMeters.length > 0 ? (
              <p className="mt-3 text-[10px] text-[#68778C]">
                {productUsageTitle(primary?.workspace_name)} — valores oficiais do provedor, sem estimativa.
              </p>
            ) : null}
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}

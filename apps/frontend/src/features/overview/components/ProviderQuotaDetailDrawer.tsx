import * as DialogPrimitive from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { AiBrandIcon, hubProviderIconSlug } from '../../../shared/branding'
import {
  consoleBudgetsFromSnapshot,
  goPlusMetersFromSnapshot,
  productUsageTitle,
} from '../lib/goPlusMeters'
import { formatQuotaFetchedAt } from '../lib/formatQuotaDate'
import type { ProviderQuotaSnapshot } from '../types/quota'
import { ConsoleBudgetsPanel } from './ConsoleBudgetsPanel'
import { GoPlusUsagePanel } from './GoPlusUsagePanel'
import { WorkspaceUsageChart } from './WorkspaceUsageChart'
import { AntigravityQuotaDrawerSection } from './AntigravityQuotaDrawerSection'
import { CursorUsageDrawerSection } from '../../cursor'
import { CursorQuotaBreakdownPanel } from '../../providers/components/CursorQuotaBreakdownPanel'
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
              <AntigravityQuotaDrawerSection quotas={primary.quotas} />
            ) : primary?.provider_id === 'cursor' ? (
              <>
                <CursorQuotaBreakdownPanel snapshot={primary} testId="cursor-drawer-quota-summary" />
                <CursorUsageDrawerSection providerId={primary.account_id} />
              </>
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

import { appPageClassName } from '../../../app/layouts/appShellLayout'
import { Banner, Panel, RefreshButton } from '../../../shared/ui'
import { AddProviderModal } from '../components/AddProviderModal'
import { ProviderDetailPanel } from '../components/ProviderDetailPanel'
import { ProviderGovernanceMetrics } from '../components/ProviderGovernanceMetrics'
import { ProviderManageDrawer } from '../components/ProviderManageDrawer'
import { ProviderQuotaPageFooter } from '../components/ProviderQuotaPageFooter'
import { ProviderQuotaTable } from '../components/ProviderQuotaTable'
import { ProviderQuotasPageSkeleton } from '../components/ProviderQuotasPageSkeleton'
import type { AdminProvider } from '../types'
import type { ProviderQuotaSnapshot } from '../../overview/types/quota'
import type { AppPageMeta } from '../../../app/layouts/pageMeta'

export type ProviderQuotasPageViewProps = Readonly<{
  meta: AppPageMeta
  providers: AdminProvider[]
  snapshots: ProviderQuotaSnapshot[]
  selectedProvider: AdminProvider | null
  selectedSnapshot?: ProviderQuotaSnapshot
  manageProvider: AdminProvider | null
  isAddModalOpen: boolean
  hasProviders: boolean
  isInitialLoading: boolean
  hasInitialError: boolean
  hasRefetchError: boolean
  isRefreshing: boolean
  isBackgroundRefreshing: boolean
  onRefresh: () => void
  onAddProvider: () => void
  onSelectProvider: (provider: AdminProvider) => void
  onOpenManage: (provider: AdminProvider) => void
  onCloseManage: () => void
  onAddModalChange: (open: boolean) => void
  onProviderAdded: (provider: AdminProvider) => void
}>

export function ProviderQuotasPageView(props: ProviderQuotasPageViewProps) {
  return (
    <div
      className={appPageClassName('providers-page')}
      aria-busy={props.isInitialLoading ? true : undefined}
    >
      <ProvidersPageHeader
        meta={props.meta}
        hasRefetchError={props.hasRefetchError}
        isRefreshing={props.isRefreshing}
        isBackgroundRefreshing={props.isBackgroundRefreshing}
        onRefresh={props.onRefresh}
        onAddProvider={props.onAddProvider}
      />
      <ProvidersPageContent {...props} />
      <ProviderQuotaPageFooter isLoading={props.isInitialLoading} />
      <ProviderPageDialogs
        providers={props.providers}
        manageProvider={props.manageProvider}
        isAddModalOpen={props.isAddModalOpen}
        onCloseManage={props.onCloseManage}
        onAddModalChange={props.onAddModalChange}
        onProviderAdded={props.onProviderAdded}
      />
    </div>
  )
}

function ProvidersPageHeader({
  meta,
  hasRefetchError,
  isRefreshing,
  isBackgroundRefreshing,
  onRefresh,
  onAddProvider,
}: Pick<
  ProviderQuotasPageViewProps,
  'meta' | 'hasRefetchError' | 'isRefreshing' | 'isBackgroundRefreshing' | 'onRefresh' | 'onAddProvider'
>) {
  return (
    <header className="providers-pagehead">
      <ProviderHeaderTitle meta={meta} hasRefetchError={hasRefetchError} />
      <ProviderHeaderActions
        isRefreshing={isRefreshing}
        isBackgroundRefreshing={isBackgroundRefreshing}
        onRefresh={onRefresh}
        onAddProvider={onAddProvider}
      />
    </header>
  )
}

function ProviderHeaderTitle({
  meta,
  hasRefetchError,
}: Pick<ProviderQuotasPageViewProps, 'meta' | 'hasRefetchError'>) {
  return (
    <div className="min-w-0">
      <p className="providers-page-eyebrow">{meta.eyebrow}</p>
      <h1 className="providers-page-title">{meta.title}</h1>
      {meta.lead ? <p className="providers-page-lead">{meta.lead}</p> : null}
      {hasRefetchError ? (
        <p
          className="mt-2 text-[10px] text-warning"
          data-testid="providers-refetch-warning"
          role="status"
        >
          Falha ao atualizar; exibindo os dados carregados anteriormente.
        </p>
      ) : null}
    </div>
  )
}

function ProviderHeaderActions({
  isRefreshing,
  isBackgroundRefreshing,
  onRefresh,
  onAddProvider,
}: Pick<
  ProviderQuotasPageViewProps,
  'isRefreshing' | 'isBackgroundRefreshing' | 'onRefresh' | 'onAddProvider'
>) {
  return (
    <div className="flex shrink-0 flex-wrap items-center gap-2">
      <span className="overview-backend-tag">DADOS DO BACKEND</span>
      <RefreshButton
        className="providers-primary-action"
        onRefresh={onRefresh}
        isRefreshing={isRefreshing || isBackgroundRefreshing}
        disabled={isRefreshing}
        aria-label="Atualizar franquias"
      />
      {isBackgroundRefreshing ? (
        <span className="sr-only" data-testid="providers-refreshing-status" role="status">
          Atualizando franquias
        </span>
      ) : null}
      <button
        type="button"
        className="providers-primary-action"
        onClick={onAddProvider}
        aria-label="Adicionar provedor"
      >
        ＋ Adicionar provedor
      </button>
    </div>
  )
}

function ProvidersPageContent(props: ProviderQuotasPageViewProps) {
  if (props.isInitialLoading) return <ProviderQuotasPageSkeleton />
  if (props.hasInitialError) {
    return <ProviderQuotasInitialError isRefreshing={props.isRefreshing} onRetry={props.onRefresh} />
  }

  return (
    <>
      <ProviderGovernanceMetrics providers={props.providers} snapshots={props.snapshots} />
      <div className="providers-layout" data-testid="providers-page-layout">
        <ProviderQuotaTable
          providers={props.providers}
          snapshots={props.snapshots}
          selectedId={props.selectedProvider?.id ?? null}
          onSelect={props.onSelectProvider}
          onOpenManage={props.onOpenManage}
        />
        <ProviderDetailPanel
          provider={props.selectedProvider}
          snapshot={props.selectedSnapshot}
          hasProviders={props.hasProviders}
          onOpenManage={props.onOpenManage}
        />
      </div>
    </>
  )
}

function ProviderQuotasInitialError({
  isRefreshing,
  onRetry,
}: Readonly<{ isRefreshing: boolean; onRetry: () => void }>) {
  return (
    <>
      <div className="providers-metrics" data-testid="providers-initial-error">
        <Banner variant="error" className="col-span-full !mb-0" role="alert">
          Não foi possível carregar as franquias dos provedores.
        </Banner>
      </div>
      <div className="providers-layout">
        <Panel className="providers-panel min-w-0 items-center justify-center lg:col-span-2">
          <div className="max-w-sm space-y-3 text-center">
            <p className="text-xs text-secondary">
              Verifique a conexão com o AI Hub e tente novamente. Nenhum valor de quota foi estimado.
            </p>
            <RefreshButton
              className="providers-primary-action"
              onRefresh={onRetry}
              isRefreshing={isRefreshing}
              label="Tentar novamente"
            />
          </div>
        </Panel>
      </div>
    </>
  )
}

function ProviderPageDialogs({
  providers,
  manageProvider,
  isAddModalOpen,
  onCloseManage,
  onAddModalChange,
  onProviderAdded,
}: Readonly<{
  providers: AdminProvider[]
  manageProvider: AdminProvider | null
  isAddModalOpen: boolean
  onCloseManage: () => void
  onAddModalChange: (open: boolean) => void
  onProviderAdded: (provider: AdminProvider) => void
}>) {
  return (
    <>
      <ProviderManageDrawer
        provider={manageProvider}
        open={manageProvider !== null}
        onClose={onCloseManage}
      />
      <AddProviderModal
        open={isAddModalOpen}
        onOpenChange={onAddModalChange}
        existingProviders={providers}
        onProviderAdded={onProviderAdded}
      />
    </>
  )
}

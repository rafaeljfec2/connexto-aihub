import { useTranslation } from 'react-i18next'
import { Panel, PanelHeader, Skeleton as SharedSkeleton } from '../../../shared/ui'

function Skeleton({ className }: Readonly<{ className?: string }>) {
  return <SharedSkeleton className={className} motionSafe />
}

export function ProviderQuotasPageSkeleton() {
  const { t } = useTranslation('providers')

  const summaryCards = [
    [t('governance.monitoredTitle'), t('governance.monitoredSubtitle'), 'w-8'],
    [t('governance.nearLimitTitle'), t('governance.nearLimitSubtitle'), 'w-7'],
    [t('governance.atLimitTitle'), t('governance.atLimitSubtitle'), 'w-7'],
    [t('governance.withoutReadingTitle'), t('governance.withoutReadingSubtitle'), 'w-7'],
  ] as const

  const tableColumns = [
    t('table.colProvider'),
    t('table.colFranchise'),
    t('table.colConsumption'),
    t('table.colAvailable'),
    t('table.colSituation'),
    t('table.colActions'),
  ]

  return (
    <>
      <span className="sr-only" data-testid="provider-quotas-skeleton-status" role="status">
        {t('skeleton.loadingSr')}
      </span>
      <ProviderQuotaSummarySkeleton summaryCards={summaryCards} />
      <div className="providers-layout" data-testid="providers-page-layout">
        <ProviderQuotaTableSkeleton tableColumns={tableColumns} title={t('table.title')} subtitle={t('table.subtitle')} />
        <ProviderQuotaDetailsSkeleton t={t} />
      </div>
    </>
  )
}

function ProviderQuotaSummarySkeleton({
  summaryCards,
}: Readonly<{
  summaryCards: ReadonlyArray<readonly [string, string, string]>
}>) {
  return (
    <div className="providers-metrics" data-testid="provider-quotas-page-skeleton" aria-hidden="true">
      {summaryCards.map(([label, detail, valueWidth]) => (
        <div className="providers-metric" key={label} data-testid="provider-quota-summary-card-skeleton">
          <div>
            <span>{label}</span>
            <small>{detail}</small>
          </div>
          <Skeleton className={`h-7 ${valueWidth} rounded-sm`} />
        </div>
      ))}
    </div>
  )
}

function ProviderQuotaTableSkeleton({
  tableColumns,
  title,
  subtitle,
}: Readonly<{ tableColumns: string[]; title: string; subtitle: string }>) {
  return (
    <Panel className="providers-panel min-w-0" aria-hidden="true">
      <PanelHeader
        className="!mb-2 shrink-0"
        title={title}
        subtitle={subtitle}
        action={<Skeleton className="h-[34px] w-52 rounded-md" />}
      />
      <div className="providers-table-body">
        <div className="grid-table-scroll scrollbar-slim">
          <table className="providers-quota-table w-full">
            <thead>
              <tr>
                {tableColumns.map((column) => (
                  <th key={column}>{column}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {Array.from({ length: 4 }, (_, index) => (
                <ProviderQuotaTableSkeletonRow key={index} />
              ))}
            </tbody>
          </table>
        </div>
        <ProviderQuotaPaginationSkeleton />
      </div>
      <footer className="providers-panel-footnote shrink-0">
        <Skeleton className="h-2 w-24" />
        <Skeleton className="h-2 w-20" />
      </footer>
    </Panel>
  )
}

function ProviderQuotaTableSkeletonRow() {
  return (
    <tr data-testid="provider-quota-skeleton-row">
      <td>
        <div className="providers-provider-name">
          <Skeleton className="size-7 shrink-0 rounded-lg" />
          <div className="min-w-0 flex-1 space-y-1.5">
            <Skeleton className="h-2.5 w-20" />
            <Skeleton className="h-2 w-24" />
          </div>
        </div>
      </td>
      <td>
        <Skeleton className="h-2.5 w-28" />
        <Skeleton className="mt-1 h-2 w-24" />
      </td>
      <td>
        <div className="space-y-1.5">
          <div className="flex justify-between gap-2">
            <Skeleton className="h-2 w-12" />
            <Skeleton className="h-2 w-14" />
          </div>
          <Skeleton className="h-2 w-full rounded-full" />
        </div>
      </td>
      <td>
        <Skeleton className="h-2.5 w-12" />
      </td>
      <td>
        <Skeleton className="h-4 w-20 rounded-[4px]" />
      </td>
      <td>
        <Skeleton className="ml-auto h-7 w-16 rounded-md" />
      </td>
    </tr>
  )
}

function ProviderQuotaPaginationSkeleton() {
  return (
    <div className="grid-panel-pagination shrink-0" aria-hidden="true">
      <div className="grid-pagination">
        <Skeleton className="h-2 w-20" />
        <div className="flex items-center gap-1.5">
          <Skeleton className="h-6 w-20 rounded-md" />
          <Skeleton className="size-7 rounded-md" />
          <Skeleton className="h-2 w-10" />
          <Skeleton className="size-7 rounded-md" />
        </div>
      </div>
    </div>
  )
}

function ProviderQuotaDetailsSkeleton({
  t,
}: Readonly<{ t: ReturnType<typeof useTranslation>['t'] }>) {
  return (
    <Panel className="providers-panel min-w-0" aria-hidden="true" data-testid="provider-quota-details-skeleton">
      <div className="providers-detail-head shrink-0">
        <strong>{t('detail.detailTitle')}</strong>
        <small>{t('skeleton.governanceInspection')}</small>
      </div>
      <div className="providers-detail-body">
        <div className="providers-provider-name mt-2 border-b border-line/70 pb-2">
          <Skeleton className="size-8 shrink-0 rounded-lg" />
          <div className="min-w-0 flex-1 space-y-1.5">
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-2 w-28" />
          </div>
        </div>
        <ProviderQuotaCardSkeleton consumptionLabel={t('skeleton.franchiseConsumptionLabel')} />
        <div className="mt-1">
          <ProviderQuotaDetailRowSkeleton label={t('skeleton.rowPrimaryReading')} />
          <ProviderQuotaDetailRowSkeleton label={t('skeleton.rowReset')} />
          <ProviderQuotaDetailRowSkeleton label={t('skeleton.rowSource')} />
          <ProviderQuotaDetailRowSkeleton label={t('skeleton.rowUpdatedAt')} />
          <ProviderQuotaDetailRowSkeleton label={t('skeleton.rowSituation')} />
        </div>
        <div className="providers-source-note space-y-1.5">
          <Skeleton className="h-2 w-full" />
          <Skeleton className="h-2 w-3/4" />
        </div>
        <div className="providers-detail-actions flex flex-col gap-2 pt-1">
          <Skeleton className="h-9 w-full rounded-md" />
          <div className="flex gap-2">
            <Skeleton className="h-7 flex-1 rounded-[7px]" />
            <Skeleton className="h-7 flex-1 rounded-[7px]" />
          </div>
        </div>
      </div>
    </Panel>
  )
}

function ProviderQuotaCardSkeleton({ consumptionLabel }: Readonly<{ consumptionLabel: string }>) {
  return (
    <div className="providers-quota-box">
      <div className="providers-quota-box-label">{consumptionLabel}</div>
      <div className="providers-quota-box-main">
        <Skeleton className="h-6 w-14" />
        <Skeleton className="h-3 w-12" />
      </div>
      <Skeleton className="h-2 w-full rounded-full" />
      <div className="providers-quota-foot">
        <Skeleton className="h-2 w-full" />
        <Skeleton className="h-2 w-full" />
      </div>
    </div>
  )
}

function ProviderQuotaDetailRowSkeleton({ label }: Readonly<{ label: string }>) {
  return (
    <div className="providers-detail-row">
      <span>{label}</span>
      <Skeleton className="h-2.5 w-28" />
    </div>
  )
}

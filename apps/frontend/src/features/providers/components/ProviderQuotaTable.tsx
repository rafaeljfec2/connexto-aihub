import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { formatNumber } from '../../../shared/i18n/format'
import { Edit2 } from 'lucide-react'
import { AiBrandIcon } from '../../../shared/branding'
import { GridPagination, Panel, PanelHeader, ProgressBar } from '../../../shared/ui'
import { useGridPagination } from '../../../shared/hooks/useGridPagination'
import type { ProviderQuotaSnapshot } from '../../overview/types/quota'
import {
  providerConnectionSubtitle,
  providerIconSlug,
} from '../lib/providerPresentation'
import {
  availableBalanceDisplay,
  consumptionCell,
  franchiseSummary,
  primaryQuotaSnapshot,
  providerQuotaSituation,
} from '../lib/quotaPresentation'
import type { AdminProvider } from '../types'

type ProviderQuotaTableProps = {
  providers: AdminProvider[]
  snapshots: ProviderQuotaSnapshot[]
  selectedId: string | null
  onSelect: (provider: AdminProvider) => void
  onOpenManage?: (provider: AdminProvider) => void
}

export function ProviderQuotaTable({
  providers,
  snapshots,
  selectedId,
  onSelect,
  onOpenManage,
}: ProviderQuotaTableProps) {
  const { t } = useTranslation('providers')
  const [search, setSearch] = useState('')

  const filtered = useMemo(
    () =>
      providers.filter((provider) => {
        const query = search.toLowerCase().trim()
        return (
          provider.name.toLowerCase().includes(query) ||
          provider.id.toLowerCase().includes(query) ||
          provider.type.toLowerCase().includes(query)
        )
      }),
    [providers, search],
  )

  const pagination = useGridPagination(filtered, { resetKey: search })

  return (
    <Panel className="providers-panel min-w-0">
      <PanelHeader
        className="!mb-2 shrink-0"
        title={t('table.title')}
        subtitle={t('table.subtitle')}
        action={
          <input
            type="search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder={t('table.searchPlaceholder')}
            className="providers-search"
            aria-label={t('table.searchAria')}
          />
        }
      />

      {filtered.length === 0 ? (
        <div className="py-8 text-center text-xs text-muted">
          {providers.length === 0 ? t('table.emptyNone') : t('table.emptySearch')}
        </div>
      ) : (
        <div className="providers-table-body">
          <div className="grid-table-scroll scrollbar-slim">
          <table className="providers-quota-table w-full">
            <thead>
              <tr>
                <th>{t('table.colProvider')}</th>
                <th>{t('table.colFranchise')}</th>
                <th>{t('table.colConsumption')}</th>
                <th>{t('table.colAvailable')}</th>
                <th>{t('table.colSituation')}</th>
                {onOpenManage ? <th className="text-right">{t('table.colActions')}</th> : null}
              </tr>
            </thead>
            <tbody>
              {pagination.pageItems.map((provider) => {
                const isSelected = provider.id === selectedId
                const snapshot = primaryQuotaSnapshot(provider.id, snapshots)
                const situation = providerQuotaSituation(provider, snapshot)
                const franchise = franchiseSummary(snapshot)
                const consumption = consumptionCell(snapshot)
                const available = availableBalanceDisplay(snapshot)
                return (
                  <tr
                    key={provider.id}
                    className={`cursor-pointer transition-colors hover:bg-white/[0.02] ${
                      isSelected ? 'selected' : ''
                    }`}
                    onClick={() => onSelect(provider)}
                    data-testid={`providers-quota-row-${provider.id}`}
                  >
                    <td>
                      <div className="providers-provider-name">
                        <AiBrandIcon
                          slug={providerIconSlug(provider)}
                          label={provider.name}
                          size={28}
                          className="providers-provider-icon"
                        />
                        <div>
                          <strong className="block text-[0.6875rem] font-semibold text-[#E1E8F3]">
                            {provider.name}
                          </strong>
                          <small className="mt-[3px] block text-[0.5625rem] text-[#909DB0]">
                            {providerConnectionSubtitle(provider)}
                          </small>
                        </div>
                      </div>
                    </td>
                    <td>
                      <div className="providers-quota-title">{franchise.title}</div>
                      <div className="providers-quota-value">{franchise.value}</div>
                    </td>
                    <td>
                      {consumption.kind === 'percent' ? (
                        <div>
                          <div className="providers-progress-head">
                            <span>
                              {t('table.percentUsed', {
                                percent: formatNumber(consumption.percent),
                              })}
                            </span>
                            <span>{consumption.label}</span>
                          </div>
                          <ProgressBar
                            value={consumption.barValue}
                            max={consumption.barMax}
                            variant="warn"
                          />
                        </div>
                      ) : consumption.kind === 'usage' ? (
                        <div>
                          <div className="providers-progress-head">
                            <span>{consumption.label}</span>
                            <span>{consumption.primary}</span>
                          </div>
                          {consumption.barValue != null &&
                          consumption.barMax != null &&
                          consumption.barMax > 0 ? (
                            <ProgressBar
                              value={consumption.barValue}
                              max={consumption.barMax}
                              variant="warn"
                            />
                          ) : (
                            <ProgressBar value={0} max={100} variant="na" />
                          )}
                          {consumption.secondary ? (
                            <div className="providers-quota-value mt-0.5">{consumption.secondary}</div>
                          ) : null}
                        </div>
                      ) : (
                        <div>
                          <div className="providers-progress-head">
                            <span>{consumption.label}</span>
                            <span>—</span>
                          </div>
                          <ProgressBar value={0} max={100} variant="na" />
                        </div>
                      )}
                    </td>
                    <td className="providers-remaining">{available}</td>
                    <td className="whitespace-nowrap">
                      <span
                        className={`providers-state ${situation.state}`}
                        title={situation.label}
                      >
                        <i aria-hidden />
                        {situation.label}
                      </span>
                    </td>
                    {onOpenManage ? (
                      <td className="text-right">
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation()
                            onOpenManage(provider)
                          }}
                          className="inline-flex items-center gap-1.5 rounded border border-line bg-surface/80 px-2.5 py-1 text-[0.6875rem] font-medium text-secondary hover:border-cobalt hover:bg-raised hover:text-primary transition-all cursor-pointer shadow-xs"
                          aria-label={t('table.editAria', { name: provider.name })}
                        >
                          <Edit2 className="size-3 text-cobalt" />
                          {t('table.edit')}
                        </button>
                      </td>
                    ) : null}
                  </tr>
                )
              })}
            </tbody>
          </table>
          </div>
        </div>
      )}

      {filtered.length > 0 ? (
        <GridPagination
          meta={pagination.meta}
          onPreviousPage={pagination.goToPreviousPage}
          onNextPage={pagination.goToNextPage}
          pageSize={pagination.pageSize}
          onPageSizeChange={pagination.setPageSize}
          className="grid-panel-pagination shrink-0"
        />
      ) : null}

      <footer className="providers-panel-footnote shrink-0">
        <span>
          {providers.length === 1
            ? t('table.footnoteProviders', { count: providers.length })
            : t('table.footnoteProvidersPlural', { count: providers.length })}
          {t('table.footnoteUnits')}
        </span>
        <span>{t('table.backendDataTag')}</span>
      </footer>
    </Panel>
  )
}

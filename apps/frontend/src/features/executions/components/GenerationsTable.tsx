import { useTranslation } from 'react-i18next'
import { Button } from '../../../shared/ui'
import { catalogProviderLabel } from '../../catalog/lib/catalogProviderLabel'
import { ExecutionOutcomeBadge } from './ExecutionOutcomeBadge'
import {
  formatGenerationStatus,
  formatGenerationThroughput,
  formatGenerationTime,
  formatGenerationTokens,
  formatOptionalDuration,
  formatProviderReportedCost,
} from '../lib/formatGeneration'
import type { Generation } from '../types'

type GenerationsTableProps = {
  readonly generations: readonly Generation[]
  readonly selectedId: string | null
  readonly onSelect: (generation: Generation) => void
  readonly onFocusRow: (row: HTMLTableRowElement) => void
  readonly isLoading: boolean
  readonly isError: boolean
  readonly errorMessage?: string
  readonly enrichmentNotice: string | null
  readonly hasNextPage: boolean
  readonly isFetchingNextPage: boolean
  readonly onLoadMore: () => void
  readonly loadedCount: number
  readonly originTag: string
}

const COLUMN_KEYS = [
  'date',
  'model',
  'provider',
  'app',
  'input',
  'output',
  'reportedCost',
  'listCost',
  'usage',
  'speed',
  'routingOverhead',
  'firstToken',
  'finishReason',
  'outcome',
] as const

export function GenerationsTable({
  generations,
  selectedId,
  onSelect,
  onFocusRow,
  isLoading,
  isError,
  errorMessage,
  enrichmentNotice,
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
  loadedCount,
  originTag,
}: GenerationsTableProps) {
  const { t } = useTranslation('executions')
  const notReported = t('format.notReported')

  if (isLoading) {
    return <p className="py-8 text-center text-xs text-muted animate-pulse">{t('table.loading')}</p>
  }
  if (isError && loadedCount === 0) {
    return (
      <p className="banner banner-error" role="alert">
        {errorMessage ?? t('table.loadError')}
      </p>
    )
  }
  if (generations.length === 0) {
    return <p className="empty-state py-10 text-center">{t('table.empty')}</p>
  }

  return (
    <section
      className="min-h-0 flex-1 overflow-hidden rounded-[9px] border border-line bg-surface"
      aria-label={t('table.ariaLabel')}
    >
      <div className="flex items-center justify-between gap-3 border-b border-line px-3 py-2.5">
        <h2 className="text-[0.6875rem] font-semibold text-primary">{t('table.title')}</h2>
        <div className="flex items-center gap-3 font-mono text-[0.5625rem] text-muted">
          <span>
            {t('table.recordsCount', { count: generations.length, originTag })}
          </span>
          {enrichmentNotice ? (
            <span className="text-warning" role="status">
              {enrichmentNotice}
            </span>
          ) : null}
        </div>
      </div>
      {isError ? (
        <p className="px-3 py-2 text-xs text-warning" role="status">
          {t('table.refreshPartialError')}
        </p>
      ) : null}
      <div className="scrollbar-slim max-h-[min(64vh,620px)] overflow-auto">
        <table className="generations-table">
          <caption className="sr-only">{t('table.caption')}</caption>
          <thead>
            <tr>
              {COLUMN_KEYS.map((columnKey) => (
                <th
                  key={columnKey}
                  scope="col"
                  className={
                    columnKey === 'input' || columnKey === 'output' ? 'text-right' : undefined
                  }
                >
                  {t(`table.columns.${columnKey}`)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {generations.map((generation) => {
              const selected = generation.execution_id === selectedId
              const modelLabel = generation.model_id ?? notReported
              return (
                <tr
                  key={generation.execution_id}
                  tabIndex={0}
                  aria-selected={selected}
                  aria-label={t('table.rowAriaLabel', {
                    model: modelLabel,
                    provider: catalogProviderLabel(generation.provider_id),
                    outcome: generation.outcome,
                  })}
                  className={selected ? 'generation-row generation-row-selected' : 'generation-row'}
                  onFocus={(event) => onFocusRow(event.currentTarget)}
                  onClick={(event) => {
                    onFocusRow(event.currentTarget)
                    onSelect(generation)
                  }}
                  onKeyDown={(event) => {
                    if (event.key !== 'Enter' && event.key !== ' ') return
                    event.preventDefault()
                    onSelect(generation)
                  }}
                >
                  <td className="font-mono text-muted">{formatGenerationTime(generation.started_at)}</td>
                  <td>
                    <span className="generation-model">{modelLabel}</span>
                  </td>
                  <td>
                    <span className="generation-provider">{catalogProviderLabel(generation.provider_id)}</span>
                  </td>
                  <td className={generation.telemetry.client_app ? undefined : 'generation-muted'}>
                    {generation.telemetry.client_app ?? notReported}
                  </td>
                  <td className="text-right font-mono">{formatGenerationTokens(generation.usage.input_tokens)}</td>
                  <td className="text-right font-mono">{formatGenerationTokens(generation.usage.output_tokens)}</td>
                  <td className="generation-cost" title={t('table.reportedCostTitle')}>
                    {formatProviderReportedCost(
                      generation.usage.provider_reported_cost,
                      generation.usage.currency,
                    )}
                  </td>
                  <td className="generation-cost" title={t('table.listCostTitle')}>
                    {formatProviderReportedCost(generation.usage.provider_list_cost, generation.usage.currency)}
                  </td>
                  <td>{formatGenerationStatus(generation.usage.status)}</td>
                  <td className="font-mono">{formatGenerationThroughput(generation)}</td>
                  <td className="font-mono">{formatOptionalDuration(generation.telemetry.routing_overhead_ns)}</td>
                  <td className="font-mono">{formatOptionalDuration(generation.telemetry.time_to_first_token_ns)}</td>
                  <td className="font-mono">{generation.telemetry.finish_reason ?? notReported}</td>
                  <td>
                    <ExecutionOutcomeBadge outcome={generation.outcome} />
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {hasNextPage ? (
        <div className="flex justify-center border-t border-line px-3 py-2">
          <Button variant="ghost" size="sm" onClick={onLoadMore} disabled={isFetchingNextPage}>
            {isFetchingNextPage ? t('table.loadingMore') : t('table.loadMore')}
          </Button>
        </div>
      ) : null}
    </section>
  )
}

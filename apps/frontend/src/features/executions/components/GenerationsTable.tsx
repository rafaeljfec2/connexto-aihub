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
  if (isLoading) {
    return <p className="py-8 text-center text-xs text-muted animate-pulse">Loading generations…</p>
  }
  if (isError && loadedCount === 0) {
    return <p className="banner banner-error" role="alert">{errorMessage ?? 'Unable to load generations.'}</p>
  }
  if (generations.length === 0) {
    return <p className="empty-state py-10 text-center">No generations match the current filters.</p>
  }

  return (
    <section
      className="min-h-0 flex-1 overflow-hidden rounded-[9px] border border-line bg-surface"
      aria-label="Generation history"
    >
      <div className="flex items-center justify-between gap-3 border-b border-line px-3 py-2.5">
        <h2 className="text-[11px] font-semibold text-primary">Generations</h2>
        <div className="flex items-center gap-3 font-mono text-[9px] text-muted">
          <span>
            {generations.length.toLocaleString('pt-BR')} REGISTROS · {originTag}
          </span>
          {enrichmentNotice ? <span className="text-warning" role="status">{enrichmentNotice}</span> : null}
        </div>
      </div>
      {isError ? <p className="px-3 py-2 text-xs text-warning" role="status">Some results could not be refreshed.</p> : null}
      <div className="scrollbar-slim max-h-[min(64vh,620px)] overflow-auto">
        <table className="generations-table">
          <caption className="sr-only">Content-free gateway generation history. Search covers loaded results.</caption>
          <thead>
            <tr>
              <th scope="col">Date</th>
              <th scope="col">Model</th>
              <th scope="col">Provider</th>
              <th scope="col">App</th>
              <th scope="col" className="text-right">Input</th>
              <th scope="col" className="text-right">Output</th>
              <th scope="col">Reported cost</th>
              <th scope="col">List cost</th>
              <th scope="col">Usage</th>
              <th scope="col">Speed</th>
              <th scope="col">Routing overhead</th>
              <th scope="col">First token</th>
              <th scope="col">Finish reason</th>
              <th scope="col">Outcome</th>
            </tr>
          </thead>
          <tbody>
            {generations.map((generation) => {
              const selected = generation.execution_id === selectedId
              return (
                <tr
                  key={generation.execution_id}
                  tabIndex={0}
                  aria-selected={selected}
                  aria-label={`${generation.model_id ?? 'Model not reported'}, ${catalogProviderLabel(generation.provider_id)}, ${generation.outcome}`}
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
                  <td><span className="generation-model">{generation.model_id ?? 'Not reported'}</span></td>
                  <td><span className="generation-provider">{catalogProviderLabel(generation.provider_id)}</span></td>
                  <td className="generation-muted">Not reported</td>
                  <td className="text-right font-mono">{formatGenerationTokens(generation.usage.input_tokens)}</td>
                  <td className="text-right font-mono">{formatGenerationTokens(generation.usage.output_tokens)}</td>
                  <td className="generation-cost" title="OpenCode usage.cost (charged amount reported by provider)">
                    {formatProviderReportedCost(generation.usage.provider_reported_cost, generation.usage.currency)}
                  </td>
                  <td className="generation-cost" title="OpenCode usage.opencode.list_cost (list price)">
                    {formatProviderReportedCost(generation.usage.provider_list_cost, generation.usage.currency)}
                  </td>
                  <td>{formatGenerationStatus(generation.usage.status)}</td>
                  <td className="font-mono">{formatGenerationThroughput(generation)}</td>
                  <td className="font-mono">{formatOptionalDuration(generation.telemetry.routing_overhead_ns)}</td>
                  <td className="font-mono">{formatOptionalDuration(generation.telemetry.time_to_first_token_ns)}</td>
                  <td className="font-mono">{generation.telemetry.finish_reason ?? 'Not reported'}</td>
                  <td><ExecutionOutcomeBadge outcome={generation.outcome} /></td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {hasNextPage ? (
        <div className="flex justify-center border-t border-line px-3 py-2">
          <Button variant="ghost" size="sm" onClick={onLoadMore} disabled={isFetchingNextPage}>
            {isFetchingNextPage ? 'Loading…' : 'Load more'}
          </Button>
        </div>
      ) : null}
    </section>
  )
}

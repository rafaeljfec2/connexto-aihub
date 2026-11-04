import type { ExecutionRecord } from '../types'
import {
  formatDurationNs,
  formatRecordTime,
  recordMatchesSearch,
} from '../lib/formatExecution'
import { ExecutionOutcomeBadge } from './ExecutionOutcomeBadge'
import { cn } from '../../../shared/ui/cn'

type ExecutionsTableProps = {
  records: ExecutionRecord[]
  search: string
  selectedId: string | null
  onSelect: (executionId: string) => void
  isLoading: boolean
  isError: boolean
  errorMessage?: string
}

export function ExecutionsTable({
  records,
  search,
  selectedId,
  onSelect,
  isLoading,
  isError,
  errorMessage,
}: ExecutionsTableProps) {
  if (isLoading) {
    return <p className="empty-state px-4 py-8">Loading execution records…</p>
  }

  if (isError) {
    return (
      <div className="banner banner-error m-4" role="alert">
        {errorMessage ??
          'Execution records are unavailable. Verify gateway access and the records store.'}
      </div>
    )
  }

  const visible = records.filter((record) => recordMatchesSearch(record, search))

  if (visible.length === 0) {
    return (
      <p className="empty-state px-4 py-10 text-center">
        No execution records match the current filters.
      </p>
    )
  }

  return (
    <div className="table-scroll executions-table-wrap">
      <table className="data-table executions-table">
        <thead>
          <tr>
            <th scope="col">Time</th>
            <th scope="col">Model / provider</th>
            <th scope="col">Phase</th>
            <th scope="col">Outcome</th>
            <th scope="col">Duration</th>
          </tr>
        </thead>
        <tbody>
          {visible.map((record) => (
            <tr
              key={record.execution_id}
              className={cn(selectedId === record.execution_id && 'executions-row-selected')}
              onClick={() => onSelect(record.execution_id)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  onSelect(record.execution_id)
                }
              }}
              tabIndex={0}
              role="button"
              aria-pressed={selectedId === record.execution_id}
            >
              <td className="font-mono text-[10px] text-muted">
                {formatRecordTime(record.completed_at)}
              </td>
              <td>
                <div className="text-[11px] font-semibold text-[#E4EAF4]">
                  {record.model_id ?? '—'}
                </div>
                <div className="text-[10px] text-[#C3CDDD]">{record.provider_id}</div>
              </td>
              <td className="font-mono text-[10px]">{record.phase}</td>
              <td>
                <ExecutionOutcomeBadge outcome={record.outcome} />
              </td>
              <td className="font-mono text-[10px]">{formatDurationNs(record.duration_ns)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

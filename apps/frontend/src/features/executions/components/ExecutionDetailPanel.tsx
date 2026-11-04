import { ExecutionOutcomeBadge } from './ExecutionOutcomeBadge'
import type { ExecutionRecord } from '../types'
import {
  formatDurationNs,
  formatOutcomeLabel,
  formatRecordTime,
  toAuditSnapshot,
} from '../lib/formatExecution'
import { Link } from 'react-router-dom'
import { Button } from '../../../shared/ui/button'

type ExecutionDetailPanelProps = {
  record: ExecutionRecord | undefined
  isLoading: boolean
  isError: boolean
  errorMessage?: string
  selectedId: string | null
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="executions-detail-row">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

export function ExecutionDetailPanel({
  record,
  isLoading,
  isError,
  errorMessage,
  selectedId,
}: ExecutionDetailPanelProps) {
  if (!selectedId) {
    return (
      <aside className="executions-detail-panel">
        <div className="executions-panel-head">
          <strong>Execution details</strong>
          <span className="font-mono text-[9px] uppercase text-muted">None selected</span>
        </div>
        <p className="empty-state px-3 py-6">Select a row to inspect metadata.</p>
      </aside>
    )
  }

  if (isLoading) {
    return (
      <aside className="executions-detail-panel">
        <div className="executions-panel-head">
          <strong>Execution details</strong>
        </div>
        <p className="empty-state px-3 py-6">Loading record…</p>
      </aside>
    )
  }

  if (isError || !record) {
    return (
      <aside className="executions-detail-panel">
        <div className="executions-panel-head">
          <strong>Execution details</strong>
        </div>
        <div className="banner banner-error m-3" role="alert">
          {errorMessage ?? 'Could not load execution metadata.'}
        </div>
      </aside>
    )
  }

  const auditJson = JSON.stringify(toAuditSnapshot(record), null, 2)

  return (
    <aside className="executions-detail-panel">
      <div className="executions-panel-head">
        <strong>Execution details</strong>
        <span className="font-mono text-[9px] uppercase text-muted">Selected</span>
      </div>

      <div className="executions-id-block">
        <small className="font-mono text-[9px] uppercase text-muted">Execution ID</small>
        <strong className="break-all font-mono text-[10px] text-[#CED8E8]">
          {record.execution_id}
        </strong>
      </div>

      <div className="executions-detail-rows">
        <DetailRow label="Outcome" value={formatOutcomeLabel(record.outcome)} />
        <DetailRow label="Phase" value={record.phase} />
        <DetailRow label="Model" value={record.model_id ?? '—'} />
        <DetailRow label="Provider" value={record.provider_id} />
        <DetailRow label="Request ID" value={record.request_id ?? '—'} />
        <DetailRow label="Started" value={formatRecordTime(record.started_at)} />
        <DetailRow label="Completed" value={formatRecordTime(record.completed_at)} />
        <DetailRow label="Duration" value={formatDurationNs(record.duration_ns)} />
        <DetailRow label="Cause" value={record.cause ?? '—'} />
        <DetailRow
          label="Upstream status"
          value={record.upstream_status === null ? '—' : String(record.upstream_status)}
        />
        <DetailRow label="Response committed" value={record.response_committed ? 'Yes' : 'No'} />
        <DetailRow label="Usage (E06)" value="Not reported (null)" />
      </div>

      <div className="executions-detail-outcome">
        <ExecutionOutcomeBadge outcome={record.outcome} />
      </div>

      <Button variant="ghost" size="sm" className="executions-catalog-link" asChild>
        <Link to="/catalog">Explore model catalog ↗</Link>
      </Button>

      <div className="executions-audit-block">
        <small className="font-mono text-[9px] uppercase text-muted">Audit snapshot</small>
        <code>{auditJson}</code>
      </div>
    </aside>
  )
}

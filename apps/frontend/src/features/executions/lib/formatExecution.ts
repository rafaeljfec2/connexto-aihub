import i18n from 'i18next'
import { formatDateTime } from '../../../shared/i18n/format'
import type { ExecutionRecord } from '../types'

const ALLOWED_AUDIT_KEYS: ReadonlyArray<keyof ExecutionRecord> = [
  'schema_version',
  'execution_id',
  'request_id',
  'provider_id',
  'model_id',
  'started_at',
  'completed_at',
  'duration_ns',
  'outcome',
  'phase',
  'cause',
  'upstream_status',
  'response_committed',
  'usage',
]

export function formatDurationNs(durationNs: number): string {
  if (!Number.isFinite(durationNs) || durationNs < 0) {
    return i18n.t('executions:format.emDash')
  }
  if (durationNs < 1_000) {
    return i18n.t('executions:format.duration.ns', { value: durationNs })
  }
  const micros = durationNs / 1_000
  if (micros < 1_000) {
    return i18n.t('executions:format.duration.micros', { value: micros.toFixed(0) })
  }
  const millis = micros / 1_000
  if (millis < 1_000) {
    return i18n.t('executions:format.duration.millis', {
      value: millis.toFixed(millis < 10 ? 1 : 0),
    })
  }
  const seconds = millis / 1_000
  return i18n.t('executions:format.duration.seconds', { value: seconds.toFixed(2) })
}

export function formatRecordTime(iso: string): string {
  const parsed = Date.parse(iso)
  if (Number.isNaN(parsed)) {
    return iso
  }
  return formatDateTime(parsed, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}

export function formatOutcomeLabel(outcome: string): string {
  switch (outcome) {
    case 'completed':
      return i18n.t('executions:format.outcomes.completed')
    case 'failed':
      return i18n.t('executions:format.outcomes.failed')
    case 'canceled':
      return i18n.t('executions:format.outcomes.canceled')
    case 'timed_out':
      return i18n.t('executions:format.outcomes.timed_out')
    default:
      return outcome
  }
}

export function recordMatchesSearch(record: ExecutionRecord, query: string): boolean {
  const normalized = query.trim().toLowerCase()
  if (normalized.length === 0) {
    return true
  }
  const haystack = [
    record.execution_id,
    record.request_id ?? '',
    record.provider_id,
    record.model_id ?? '',
    record.outcome,
    record.phase,
    record.cause ?? '',
    record.upstream_status === null ? '' : String(record.upstream_status),
  ]
    .join(' ')
    .toLowerCase()
  return haystack.includes(normalized)
}

export function toAuditSnapshot(record: ExecutionRecord): Record<string, unknown> {
  const snapshot: Record<string, unknown> = {}
  for (const key of ALLOWED_AUDIT_KEYS) {
    snapshot[key] = record[key]
  }
  return snapshot
}

export function isAllowedAuditKey(key: string): key is keyof ExecutionRecord {
  return (ALLOWED_AUDIT_KEYS as readonly string[]).includes(key)
}

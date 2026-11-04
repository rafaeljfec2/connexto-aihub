import { fetchJson } from '../../../shared/api/client'
import type { ExecutionRecord, ExecutionRecordsFilter, ExecutionRecordsResponse } from '../types'

export const executionsQueryKeys = {
  list: (filter: ExecutionRecordsFilter) => ['executions', 'list', filter] as const,
  detail: (executionId: string) => ['executions', 'detail', executionId] as const,
}

export function buildExecutionsListPath(filter: ExecutionRecordsFilter): string {
  const params = new URLSearchParams()
  if (filter.limit !== undefined) {
    params.set('limit', String(filter.limit))
  }
  if (filter.provider_id) {
    params.set('provider_id', filter.provider_id)
  }
  if (filter.model_id) {
    params.set('model_id', filter.model_id)
  }
  if (filter.outcome) {
    params.set('outcome', filter.outcome)
  }
  if (filter.started_after) {
    params.set('started_after', filter.started_after)
  }
  if (filter.started_before) {
    params.set('started_before', filter.started_before)
  }
  if (filter.cursor) {
    params.set('cursor', filter.cursor)
  }
  const query = params.toString()
  return query.length > 0 ? `/v1/records/executions?${query}` : '/v1/records/executions'
}

export async function listExecutionRecords(
  filter: ExecutionRecordsFilter,
): Promise<ExecutionRecordsResponse> {
  return fetchJson<ExecutionRecordsResponse>(buildExecutionsListPath(filter))
}

export async function getExecutionRecord(executionId: string): Promise<ExecutionRecord> {
  return fetchJson<ExecutionRecord>(
    `/v1/records/executions/${encodeURIComponent(executionId)}`,
  )
}

export const EXECUTION_OUTCOMES = ['completed', 'failed', 'canceled', 'timed_out'] as const

export type ExecutionOutcome = (typeof EXECUTION_OUTCOMES)[number]

export type ExecutionRecord = {
  schema_version: number
  execution_id: string
  request_id: string | null
  provider_id: string
  model_id: string | null
  started_at: string
  completed_at: string
  duration_ns: number
  outcome: ExecutionOutcome | string
  phase: string
  cause: string | null
  upstream_status: number | null
  response_committed: boolean
  usage: null
}

export type ExecutionRecordsResponse = {
  records: ExecutionRecord[]
  count: number
  next_cursor: string | null
  degraded: boolean
}

export type ExecutionRecordsFilter = {
  limit?: number
  provider_id?: string
  model_id?: string
  outcome?: ExecutionOutcome | ''
  started_after?: string
  started_before?: string
  cursor?: string
}

export type ExecutionsUiFilter = {
  provider_id: string
  model_id: string
  outcome: ExecutionOutcome | 'all'
  search: string
}

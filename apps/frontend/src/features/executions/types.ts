export const EXECUTION_OUTCOMES = ['completed', 'failed', 'canceled', 'timed_out'] as const

export type ExecutionOutcome = (typeof EXECUTION_OUTCOMES)[number]

export type GenerationUsage = {
  readonly status: 'reported' | 'partial' | 'unavailable' | 'invalid' | string
  readonly source: string
  readonly input_tokens: number | null
  readonly output_tokens: number | null
  readonly provider_reported_cost: string | null
  readonly currency: string | null
}

export type GenerationTelemetry = {
  readonly streaming: boolean | null
  readonly provider_latency_ns: number | null
  readonly routing_overhead_ns: number | null
  readonly time_to_first_token_ns: number | null
  readonly output_stream_duration_ns: number | null
  readonly finish_reason: string | null
}

export type GenerationProviderEvent = {
  readonly sequence: number
  readonly type: string
  readonly observed_at: string
  readonly provider_id: string
  readonly model_id: string
  readonly upstream_status: number | null
  readonly duration_ns: number | null
}

export type Generation = {
  readonly execution_id: string
  readonly request_id: string | null
  readonly provider_id: string
  readonly model_id: string | null
  readonly started_at: string
  readonly completed_at: string
  readonly duration_ns: number
  readonly outcome: ExecutionOutcome | string
  readonly phase: string
  readonly cause: string | null
  readonly upstream_status: number | null
  readonly usage: GenerationUsage
  readonly telemetry: GenerationTelemetry
  readonly enrichment_status: 'available' | 'partial' | 'unavailable' | string
}

export type GenerationDetail = Generation & {
  readonly provider_session_id: string | null
  readonly provider_responses: readonly GenerationProviderEvent[]
  readonly degraded: boolean
}

export type GenerationsListResponse = {
  readonly generations: readonly Generation[]
  readonly count: number
  readonly next_cursor: string | null
  readonly degraded: boolean
}

export type GenerationsFilter = {
  readonly limit?: number
  readonly provider_id?: string
  readonly model_id?: string
  readonly outcome?: ExecutionOutcome | ''
  readonly started_after?: string
  readonly started_before?: string
  readonly cursor?: string
}

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

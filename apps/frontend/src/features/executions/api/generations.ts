import { fetchJson } from '../../../shared/api/client'
import type { GenerationDetail, GenerationsFilter, GenerationsListResponse } from '../types'

export const generationsQueryKeys = {
  list: (filter: Omit<GenerationsFilter, 'cursor'>) => ['generations', 'list', filter] as const,
  detail: (executionId: string) => ['generations', 'detail', executionId] as const,
}

export function buildGenerationsListPath(filter: GenerationsFilter): string {
  const params = new URLSearchParams()
  if (filter.limit !== undefined) params.set('limit', String(filter.limit))
  if (filter.provider_id) params.set('provider_id', filter.provider_id)
  if (filter.model_id) params.set('model_id', filter.model_id)
  if (filter.outcome) params.set('outcome', filter.outcome)
  if (filter.started_after) params.set('started_after', filter.started_after)
  if (filter.started_before) params.set('started_before', filter.started_before)
  if (filter.cursor) params.set('cursor', filter.cursor)
  const query = params.toString()
  return query.length > 0 ? `/v1/generations?${query}` : '/v1/generations'
}

export function listGenerations(filter: GenerationsFilter): Promise<GenerationsListResponse> {
  return fetchJson<GenerationsListResponse>(buildGenerationsListPath(filter))
}

export function getGeneration(executionId: string): Promise<GenerationDetail> {
  return fetchJson<GenerationDetail>(`/v1/generations/${encodeURIComponent(executionId)}`)
}

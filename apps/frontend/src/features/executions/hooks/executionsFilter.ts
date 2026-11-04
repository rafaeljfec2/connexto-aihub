import { ApiError } from '../../../shared/api/client'
import type { ExecutionOutcome } from '../types'

export function executionsErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError) {
    return error.message
  }
  if (error instanceof Error && error.message.length > 0) {
    return error.message
  }
  return fallback
}

export function toApiFilter(ui: {
  provider_id: string
  model_id: string
  outcome: ExecutionOutcome | 'all'
}): {
  provider_id?: string
  model_id?: string
  outcome?: ExecutionOutcome
} {
  return {
    provider_id: ui.provider_id.trim() || undefined,
    model_id: ui.model_id.trim() || undefined,
    outcome: ui.outcome === 'all' ? undefined : ui.outcome,
  }
}

export const DEFAULT_PAGE_LIMIT = 25

export const defaultExecutionsUiFilter = {
  provider_id: '',
  model_id: '',
  outcome: 'all' as const,
  search: '',
}

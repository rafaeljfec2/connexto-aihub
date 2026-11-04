import type { Generation } from '../types'
import { formatDurationNs, formatRecordTime } from './formatExecution'

export function generationMatchesSearch(generation: Generation, query: string): boolean {
  const normalized = query.trim().toLowerCase()
  if (normalized.length === 0) return true
  const searchable = [
    generation.execution_id,
    generation.request_id ?? '',
    generation.provider_id,
    generation.model_id ?? '',
    generation.outcome,
    generation.phase,
    generation.cause ?? '',
  ]
    .join(' ')
    .toLowerCase()
  return searchable.includes(normalized)
}

export function formatGenerationTokens(value: number | null): string {
  if (value === null || !Number.isSafeInteger(value) || value < 0) return 'Not reported'
  return new Intl.NumberFormat().format(value)
}

export function formatProviderReportedCost(cost: string | null, currency: string | null): string {
  if (cost === null || !/^-?\d+(\.\d+)?$/.test(cost)) return 'Not reported'
  return currency === null ? `${cost} · currency not reported` : `${currency} ${cost}`
}

export function formatGenerationThroughput(generation: Generation): string {
  const tokens = generation.usage.output_tokens
  const duration = generation.telemetry.output_stream_duration_ns
  if (tokens === null || duration === null || duration <= 0 || tokens < 0) return 'Not reported'
  const throughput = tokens / (duration / 1_000_000_000)
  if (!Number.isFinite(throughput)) return 'Not reported'
  return `${throughput.toFixed(1)} tok/s`
}

export function formatOptionalDuration(durationNs: number | null): string {
  return durationNs === null ? 'Not reported' : formatDurationNs(durationNs)
}

export function formatGenerationTime(value: string): string {
  const parsed = Date.parse(value)
  return Number.isNaN(parsed) ? 'Not reported' : formatRecordTime(value)
}

export function formatGenerationStatus(status: string): string {
  switch (status) {
    case 'reported':
      return 'Reported'
    case 'partial':
      return 'Partial'
    case 'invalid':
      return 'Invalid'
    case 'unavailable':
      return 'Unavailable'
    default:
      return 'Not reported'
  }
}

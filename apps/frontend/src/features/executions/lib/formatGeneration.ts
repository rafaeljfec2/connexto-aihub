import i18n from 'i18next'
import { formatNumber } from '../../../shared/i18n/format'
import type { Generation } from '../types'
import { formatDurationNs, formatRecordTime } from './formatExecution'

function notReported(): string {
  return i18n.t('executions:format.notReported')
}

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
  if (value === null || !Number.isSafeInteger(value) || value < 0) return notReported()
  return formatNumber(value)
}

export function formatProviderReportedCost(cost: string | null, currency: string | null): string {
  if (cost === null || !/^-?\d+(\.\d+)?$/.test(cost)) return notReported()
  return currency === null
    ? i18n.t('executions:format.currencyNotReported', { cost })
    : i18n.t('executions:format.currencyValue', { currency, cost })
}

export function formatGenerationThroughput(generation: Generation): string {
  const tokens = generation.usage.output_tokens
  const duration = generation.telemetry.output_stream_duration_ns
  if (tokens === null || duration === null || duration <= 0 || tokens < 0) return notReported()
  const throughput = tokens / (duration / 1_000_000_000)
  if (!Number.isFinite(throughput)) return notReported()
  return i18n.t('executions:format.throughput', { value: throughput.toFixed(1) })
}

export function formatOptionalDuration(durationNs: number | null): string {
  return durationNs === null ? notReported() : formatDurationNs(durationNs)
}

export function formatGenerationTime(value: string): string {
  const parsed = Date.parse(value)
  return Number.isNaN(parsed) ? notReported() : formatRecordTime(value)
}

export function formatGenerationStatus(status: string): string {
  switch (status) {
    case 'reported':
      return i18n.t('executions:format.usageStatus.reported')
    case 'partial':
      return i18n.t('executions:format.usageStatus.partial')
    case 'invalid':
      return i18n.t('executions:format.usageStatus.invalid')
    case 'unavailable':
      return i18n.t('executions:format.usageStatus.unavailable')
    default:
      return notReported()
  }
}

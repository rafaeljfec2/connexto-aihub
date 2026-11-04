import { describe, expect, it } from 'vitest'
import type { ExecutionRecord } from '../types'
import {
  formatDurationNs,
  formatOutcomeLabel,
  isAllowedAuditKey,
  recordMatchesSearch,
  toAuditSnapshot,
} from './formatExecution'

const sampleRecord: ExecutionRecord = {
  schema_version: 1,
  execution_id: 'a'.repeat(32),
  request_id: 'req-123',
  provider_id: 'opencode-go',
  model_id: 'vendor/model',
  started_at: '2026-11-03T12:00:00.000Z',
  completed_at: '2026-11-03T12:00:01.000Z',
  duration_ns: 184_000_000,
  outcome: 'completed',
  phase: 'forwarding',
  cause: null,
  upstream_status: 200,
  response_committed: true,
  usage: null,
}

describe('formatExecution', () => {
  it('formats duration in milliseconds', () => {
    expect(formatDurationNs(184_000_000)).toBe('184 ms')
  })

  it('labels known outcomes', () => {
    expect(formatOutcomeLabel('failed')).toBe('Failed')
  })

  it('filters loaded rows by search tokens', () => {
    expect(recordMatchesSearch(sampleRecord, 'vendor')).toBe(true)
    expect(recordMatchesSearch(sampleRecord, 'missing-token')).toBe(false)
  })

  it('matches cause and upstream status in client search', () => {
    const failed: ExecutionRecord = {
      ...sampleRecord,
      outcome: 'failed',
      cause: 'transport',
      upstream_status: 502,
    }
    expect(recordMatchesSearch(failed, 'transport')).toBe(true)
    expect(recordMatchesSearch(failed, '502')).toBe(true)
  })

  it('audit snapshot exposes only schema fields', () => {
    const snapshot = toAuditSnapshot(sampleRecord)
    expect(Object.keys(snapshot).sort()).toEqual(
      [
        'cause',
        'completed_at',
        'duration_ns',
        'execution_id',
        'model_id',
        'outcome',
        'phase',
        'provider_id',
        'request_id',
        'response_committed',
        'schema_version',
        'started_at',
        'upstream_status',
        'usage',
      ].sort(),
    )
    expect(snapshot).not.toHaveProperty('prompt')
    expect(snapshot).not.toHaveProperty('response')
  })

  it('rejects unknown audit keys', () => {
    expect(isAllowedAuditKey('execution_id')).toBe(true)
    expect(isAllowedAuditKey('prompt')).toBe(false)
  })
})

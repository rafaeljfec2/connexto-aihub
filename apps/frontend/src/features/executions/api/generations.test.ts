import { describe, expect, it } from 'vitest'
import { buildGenerationsListPath } from './generations'

describe('buildGenerationsListPath', () => {
  it('maps the supported filters and cursor to the generation API', () => {
    const path = buildGenerationsListPath({
      limit: 50,
      provider_id: 'opencode-go',
      model_id: 'vendor/model',
      outcome: 'timed_out',
      started_after: '2026-11-03T00:00:00.000Z',
      started_before: '2026-11-04T23:59:59.999Z',
      cursor: 'cursor-2',
    })

    expect(path).toContain('/v1/generations?')
    const query = new URLSearchParams(path.split('?')[1])
    expect(query.get('limit')).toBe('50')
    expect(query.get('provider_id')).toBe('opencode-go')
    expect(query.get('model_id')).toBe('vendor/model')
    expect(query.get('outcome')).toBe('timed_out')
    expect(query.get('started_after')).toBe('2026-11-03T00:00:00.000Z')
    expect(query.get('started_before')).toBe('2026-11-04T23:59:59.999Z')
    expect(query.get('cursor')).toBe('cursor-2')
  })

  it('omits unset filters', () => {
    expect(buildGenerationsListPath({ limit: 50 })).toBe('/v1/generations?limit=50')
  })
})

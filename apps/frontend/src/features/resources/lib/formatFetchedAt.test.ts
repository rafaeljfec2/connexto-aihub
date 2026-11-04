import { describe, expect, it } from 'vitest'
import { formatFetchedAtUtc } from './formatFetchedAt'

describe('formatFetchedAtUtc', () => {
  it('formats epoch ms as UTC without T separator', () => {
    expect(formatFetchedAtUtc(Date.UTC(2026, 10, 4, 12, 30, 45))).toBe('2026-11-04 12:30:45')
  })
})

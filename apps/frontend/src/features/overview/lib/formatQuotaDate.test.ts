import { describe, expect, it } from 'vitest'
import {
  formatQuotaCalendarDay,
  formatQuotaChartAxisDay,
  formatQuotaFetchedAt,
} from './formatQuotaDate'

describe('formatQuotaDate', () => {
  it('formats fetched_at as pt-BR datetime', () => {
    const formatted = formatQuotaFetchedAt('2026-11-04T16:00:00.000Z')
    expect(formatted).toMatch(/04/)
    expect(formatted).toMatch(/11/)
    expect(formatted).toMatch(/2026/)
    expect(formatted).not.toContain('T')
  })

  it('formats calendar days in pt-BR with UTC month boundaries', () => {
    expect(formatQuotaCalendarDay('2026-11-02')).toMatch(/2 de novembro de 2026/i)
    expect(formatQuotaChartAxisDay('2026-11-30')).toMatch(/30 de nov/i)
  })

  it('returns raw string when date-only is invalid', () => {
    expect(formatQuotaCalendarDay('bad')).toBe('bad')
  })
})

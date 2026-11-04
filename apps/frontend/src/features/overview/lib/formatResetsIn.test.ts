import { describe, expect, it } from 'vitest'
import { formatResetsIn } from './formatResetsIn'

describe('formatResetsIn', () => {
  const now = new Date('2026-11-04T16:00:00.000Z')

  it('formats hours and minutes for a same-day reset', () => {
    expect(formatResetsIn('2026-11-04T18:31:46.000Z', now)).toBe('Reinicia em 2h 31m')
  })

  it('formats days and hours for a multi-day reset', () => {
    expect(formatResetsIn('2026-11-09T00:00:00.000Z', now)).toBe('Reinicia em 4d 8h')
  })

  it('returns Reinicia em breve when the timestamp is in the past', () => {
    expect(formatResetsIn('2026-11-04T15:00:00.000Z', now)).toBe('Reinicia em breve')
  })

  it('returns undefined for invalid timestamps', () => {
    expect(formatResetsIn('not-a-date', now)).toBeUndefined()
  })
})

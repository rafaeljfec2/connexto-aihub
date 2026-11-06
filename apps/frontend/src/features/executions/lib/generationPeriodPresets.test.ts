import { describe, expect, it } from 'vitest'
import { inferPeriodPreset, resolvePeriodPresetRange } from './generationPeriodPresets'

describe('generationPeriodPresets', () => {
  const today = new Date('2026-11-05T15:00:00.000Z')

  it('resolves the last 30 days inclusive of today', () => {
    expect(resolvePeriodPresetRange('30d', today)).toEqual({
      start_date: '2026-10-07',
      end_date: '2026-11-05',
    })
  })

  it('infers preset from matching ranges', () => {
    const range = resolvePeriodPresetRange('7d', today)
    expect(inferPeriodPreset(range.start_date, range.end_date, today)).toBe('7d')
    expect(inferPeriodPreset('', '')).toBe('all')
  })
})

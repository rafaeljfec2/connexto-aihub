import { describe, expect, it } from 'vitest'
import { buildAdmissionsAreaPath, buildAdmissionsLinePath } from './consumptionChart'

describe('consumptionChart paths', () => {
  it('returns null when the daily series is empty', () => {
    expect(buildAdmissionsAreaPath([])).toBeNull()
    expect(buildAdmissionsLinePath([])).toBeNull()
  })

  it('builds closed area and open line paths for a non-empty series', () => {
    const daily = [
      { period_start: '2026-11-10T00:00:00Z', admitted_count: 2 },
      { period_start: '2026-11-11T00:00:00Z', admitted_count: 8 },
    ]

    const area = buildAdmissionsAreaPath(daily)
    const line = buildAdmissionsLinePath(daily)

    expect(area).toMatch(/^M .+ C .+ Z$/)
    expect(line).toMatch(/^M .+ C /)
    expect(area?.endsWith(' Z')).toBe(true)
    expect(line?.includes(' Z')).toBe(false)
  })

  it('closes a single-day area across the full plot width', () => {
    const area = buildAdmissionsAreaPath([
      { period_start: '2026-11-12T00:00:00Z', admitted_count: 4 },
    ])

    expect(area).toContain('L 560.0 116')
    expect(area).toContain('L 0.0 116')
    expect(area?.endsWith(' Z')).toBe(true)
  })
})

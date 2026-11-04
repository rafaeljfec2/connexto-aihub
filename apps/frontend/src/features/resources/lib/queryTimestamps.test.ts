import { describe, expect, it } from 'vitest'
import { maxDataUpdatedAt } from './queryTimestamps'

describe('maxDataUpdatedAt', () => {
  it('returns undefined when no valid timestamps', () => {
    expect(maxDataUpdatedAt(0, NaN)).toBeUndefined()
  })

  it('returns the latest valid timestamp', () => {
    expect(maxDataUpdatedAt(100, 250, 200)).toBe(250)
  })
})

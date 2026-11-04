import { describe, expect, it } from 'vitest'
import { formatQuotaValue } from './formatQuotaValue'

describe('formatQuotaValue', () => {
  it('formats request counts as integers', () => {
    expect(formatQuotaValue(0, 'requests')).toBe('0 requests')
    expect(formatQuotaValue(1000, 'requests')).toBe('1.000 requests')
  })

  it('preserves provider-reported USD decimal precision', () => {
    expect(formatQuotaValue(4.84186443, 'USD')).toBe('US$ 4,84186443')
    expect(formatQuotaValue(0, 'USD')).toBe('US$ 0,00')
  })
})

import { describe, expect, it } from 'vitest'
import { formatTokenCount, isUnknownToken } from './formatTokens'

describe('formatTokenCount', () => {
  it('renders null as unknown label', () => {
    expect(formatTokenCount(null)).toBe('Desconhecido')
    expect(isUnknownToken(null)).toBe(true)
  })

  it('renders explicit zero as zero', () => {
    expect(formatTokenCount(0)).toBe('0')
    expect(isUnknownToken(0)).toBe(false)
  })

  it('formats positive counts', () => {
    expect(formatTokenCount(1200)).toBe('1.200')
  })
})

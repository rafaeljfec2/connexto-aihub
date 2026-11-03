import { describe, expect, it } from 'vitest'
import { statusClass } from './statusClass'

describe('statusClass', () => {
  it('maps healthy states', () => {
    expect(statusClass('healthy')).toBe('status-healthy')
    expect(statusClass('up')).toBe('status-healthy')
  })

  it('maps muted states', () => {
    expect(statusClass('disabled')).toBe('status-muted')
  })

  it('defaults unknown bad states to unhealthy styling', () => {
    expect(statusClass('unreachable')).toBe('status-unhealthy')
  })
})

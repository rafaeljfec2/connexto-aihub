import { describe, expect, it } from 'vitest'
import { ApiError } from './client'

describe('ApiError', () => {
  it('preserves server error type', () => {
    const error = new ApiError(502, { type: 'upstream_failure', message: 'failed' })
    expect(error.type).toBe('upstream_failure')
    expect(error.message).toBe('failed')
  })
})

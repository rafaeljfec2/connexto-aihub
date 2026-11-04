import { ApiError } from '../../../shared/api/client'
import { describe, expect, it } from 'vitest'
import { resourcesErrorMessage } from './resourcesError'

describe('resourcesErrorMessage', () => {
  it('surfaces ApiError messages', () => {
    expect(resourcesErrorMessage(new ApiError(400, { type: 'invalid_request', message: 'bad range' }), 'fallback')).toBe(
      'bad range',
    )
  })

  it('falls back for unknown errors', () => {
    expect(resourcesErrorMessage(undefined, 'fallback')).toBe('fallback')
  })
})

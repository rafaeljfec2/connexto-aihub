import { describe, expect, it } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import { executionsErrorMessage, toApiFilter } from '../hooks/executionsFilter'

describe('executionsFilter helpers', () => {
  describe('toApiFilter', () => {
    it('omits empty values and maps all to all', () => {
      expect(
        toApiFilter({ provider_id: '  ', model_id: '', outcome: 'all' }),
      ).toEqual({})
    })

    it('passes trimmed provider, model, and outcome', () => {
      expect(
        toApiFilter({
          provider_id: ' opencode-go ',
          model_id: 'vendor/model',
          outcome: 'failed',
        }),
      ).toEqual({
        provider_id: 'opencode-go',
        model_id: 'vendor/model',
        outcome: 'failed',
      })
    })
  })

  describe('executionsErrorMessage', () => {
    it('returns ApiError message from gateway', () => {
      const error = new ApiError(400, {
        type: 'invalid_request',
        message: 'outcome is not supported',
      })
      expect(executionsErrorMessage(error, 'fallback')).toBe('outcome is not supported')
    })

    it('uses fallback for unknown errors', () => {
      expect(executionsErrorMessage(null, 'fallback')).toBe('fallback')
    })
  })
})

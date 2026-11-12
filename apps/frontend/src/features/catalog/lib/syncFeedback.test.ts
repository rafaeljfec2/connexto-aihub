import { describe, expect, it } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import { formatCatalogSyncFeedback } from './syncFeedback'

describe('formatCatalogSyncFeedback', () => {
  it('maps sync_in_progress conflict to warning', () => {
    const feedback = formatCatalogSyncFeedback(
      new ApiError(409, { type: 'sync_in_progress', message: 'a synchronization is already in progress' }),
    )
    expect(feedback?.tone).toBe('warning')
    expect(feedback?.message).toContain('sincronização')
  })

  it('maps provider_disabled conflict to warning', () => {
    const feedback = formatCatalogSyncFeedback(
      new ApiError(409, { type: 'provider_disabled', message: 'provider is disabled' }),
    )
    expect(feedback?.tone).toBe('warning')
  })
})

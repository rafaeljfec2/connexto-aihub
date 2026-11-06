import { describe, expect, it } from 'vitest'
import { validateGenerationFilterDraft } from './validateGenerationFilters'

describe('validateGenerationFilterDraft', () => {
  it('rejects inverted date ranges', () => {
    const result = validateGenerationFilterDraft({
      provider_id: '',
      model_id: '',
      outcome: 'all',
      start_date: '2026-11-05',
      end_date: '2026-11-04',
    })
    expect(result.ok).toBe(false)
  })
})

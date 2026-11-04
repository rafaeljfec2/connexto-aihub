import { describe, expect, it } from 'vitest'
import { applyCatalogColumnFilters, defaultCatalogColumnFilters } from './catalogColumnFilters'
import type { CatalogModel } from '../types'

const sample: CatalogModel = {
  provider: 'openai',
  model_id: 'gpt-4.1-mini',
  name: 'GPT-4.1 mini',
  capabilities: {},
  presence: 'catalog',
  freshness: 'fresh',
  provider_state: 'ok',
  execution_verification: 'verified',
  listing_state: 'Listado',
  listing_reasons: [],
}

describe('applyCatalogColumnFilters', () => {
  it('filters by provider column', () => {
    const models = [
      sample,
      { ...sample, provider: 'anthropic', model_id: 'claude-3', name: 'Claude 3' },
    ]
    expect(
      applyCatalogColumnFilters(models, {
        ...defaultCatalogColumnFilters,
        provider: 'openai',
      }),
    ).toHaveLength(1)
  })

  it('filters by model name substring', () => {
    expect(
      applyCatalogColumnFilters([sample], {
        ...defaultCatalogColumnFilters,
        model: 'mini',
      }),
    ).toHaveLength(1)
  })
})

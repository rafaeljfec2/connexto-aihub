import { describe, expect, it } from 'vitest'
import { formatPricingCell, formatUsdPerMillion } from './formatCatalogPricing'
import type { CatalogModelPricing } from '../types'

const samplePricing: CatalogModelPricing = {
  currency: 'USD',
  unit: 'per_1m_tokens',
  input: 10,
  output: 50,
  cache_read: 0.25,
  source: 'provider_catalog',
  complete: true,
}

describe('formatUsdPerMillion', () => {
  it('formats USD amounts in pt-BR', () => {
    expect(formatUsdPerMillion(10)).toMatch(/US\$\s*10,00/)
    expect(formatUsdPerMillion(0.15)).toMatch(/US\$\s*0,15/)
  })
})

describe('formatPricingCell', () => {
  it('returns Não informado when pricing is absent', () => {
    expect(formatPricingCell(undefined, 'input')).toBe('Não informado')
  })

  it('returns em dash for missing optional fields', () => {
    expect(formatPricingCell({ ...samplePricing, cache_write: undefined }, 'cache_write')).toBe('—')
  })

  it('formats known fields', () => {
    expect(formatPricingCell(samplePricing, 'input')).toMatch(/10,00/)
    expect(formatPricingCell(samplePricing, 'output')).toMatch(/50,00/)
  })
})

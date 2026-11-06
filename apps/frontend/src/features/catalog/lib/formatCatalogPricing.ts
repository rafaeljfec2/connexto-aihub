import type { CatalogModelPricing } from '../types'

export type CatalogPricingField = 'input' | 'output' | 'cache_read' | 'cache_write'

export function formatUsdPerMillion(value: number): string {
  const amount = value.toLocaleString('pt-BR', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  return `US$ ${amount}`
}

/** Table/detail cell: formatted price, em dash when field absent, or Não informado when no pricing object. */
export function formatPricingCell(
  pricing: CatalogModelPricing | null | undefined,
  field: CatalogPricingField,
): string {
  if (!pricing) {
    return 'Não informado'
  }

  const value = pricing[field]
  if (value == null) {
    return '—'
  }

  return formatUsdPerMillion(value)
}

export function formatCatalogContextLimit(context: number | null | undefined): string {
  if (context == null) {
    return 'Não informada'
  }
  return context.toLocaleString('pt-BR')
}

export function hasCatalogPricing(pricing: CatalogModelPricing | null | undefined): boolean {
  return pricing != null && pricing.complete
}

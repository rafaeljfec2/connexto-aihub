import type { ProviderUsageProviderRow } from '../types'

export type UsageDistributionSlice = {
  providerId: string
  value: number
  percent: number
}

export function buildProviderUsageDistribution(
  rows: ProviderUsageProviderRow[],
): UsageDistributionSlice[] {
  if (rows.length === 0) {
    return []
  }

  const tokenTotals = rows.map((row) => row.total_tokens)
  const allTokensKnown = tokenTotals.every((value) => value !== null)
  const useTokens = allTokensKnown && tokenTotals.some((value) => (value ?? 0) > 0)

  const weights = rows.map((row) => {
    if (useTokens) {
      return row.total_tokens ?? 0
    }
    return row.executions
  })

  const totalWeight = weights.reduce((sum, value) => sum + value, 0)
  if (totalWeight <= 0) {
    return rows.map((row) => ({
      providerId: row.provider_id,
      value: useTokens ? (row.total_tokens ?? 0) : row.executions,
      percent: 0,
    }))
  }

  return rows.map((row, index) => ({
    providerId: row.provider_id,
    value: weights[index] ?? 0,
    percent: Math.round(((weights[index] ?? 0) / totalWeight) * 1000) / 10,
  }))
}

export function maxDataUpdatedAt(...times: number[]): number | undefined {
  const valid = times.filter((value) => Number.isFinite(value) && value > 0)
  if (valid.length === 0) {
    return undefined
  }
  return Math.max(...valid)
}

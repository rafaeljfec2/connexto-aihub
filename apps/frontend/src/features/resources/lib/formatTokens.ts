import type { NullableTokenCount } from '../types'

export function formatTokenCount(value: NullableTokenCount | undefined): string {
  if (value === null || value === undefined) {
    return 'Unknown'
  }
  return new Intl.NumberFormat('en-US').format(value)
}

export function isUnknownToken(value: NullableTokenCount | undefined): boolean {
  return value === null || value === undefined
}

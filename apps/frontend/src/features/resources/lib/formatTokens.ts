import i18n from '../../../shared/i18n'
import { formatNumber } from '../../../shared/i18n/format'
import type { NullableTokenCount } from '../types'

export function formatTokenCount(value: NullableTokenCount | undefined): string {
  if (value === null || value === undefined) {
    return i18n.t('resources:tokens.unknown')
  }
  return formatNumber(value)
}

export function isUnknownToken(value: NullableTokenCount | undefined): boolean {
  return value === null || value === undefined
}

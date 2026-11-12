import i18n from 'i18next'
import { intlLocaleFor, type SupportedLocale } from './locale'

function activeIntlLocale(): string {
  const lang = i18n.language
  if (lang === 'en' || lang.startsWith('en-')) {
    return intlLocaleFor('en')
  }
  return intlLocaleFor('pt-BR')
}

export function formatNumber(
  value: number,
  options?: Intl.NumberFormatOptions,
  locale?: SupportedLocale,
): string {
  const intlLocale = locale ? intlLocaleFor(locale) : activeIntlLocale()
  return new Intl.NumberFormat(intlLocale, options).format(value)
}

export function formatPercent(value: number, locale?: SupportedLocale): string {
  const intlLocale = locale ? intlLocaleFor(locale) : activeIntlLocale()
  return new Intl.NumberFormat(intlLocale, {
    style: 'percent',
    maximumFractionDigits: 1,
  }).format(value)
}

export function formatDateTime(
  value: Date | number | string,
  options?: Intl.DateTimeFormatOptions,
  locale?: SupportedLocale,
): string {
  const intlLocale = locale ? intlLocaleFor(locale) : activeIntlLocale()
  const date = value instanceof Date ? value : new Date(value)
  return new Intl.DateTimeFormat(intlLocale, options).format(date)
}

export function formatCurrencyUsd(value: number, locale?: SupportedLocale): string {
  const intlLocale = locale ? intlLocaleFor(locale) : activeIntlLocale()
  return new Intl.NumberFormat(intlLocale, {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
}

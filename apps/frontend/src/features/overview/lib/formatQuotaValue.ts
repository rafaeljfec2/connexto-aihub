export function formatQuotaValue(value: number, unit?: string): string {
  const isCount = unit === 'requests'
  const formatted = value.toLocaleString('pt-BR', {
    minimumFractionDigits: isCount ? 0 : 2,
    maximumFractionDigits: isCount ? 0 : 20,
  })
  if (unit === 'USD') return `US$ ${formatted}`
  if (unit) return `${formatted} ${unit}`
  return formatted
}

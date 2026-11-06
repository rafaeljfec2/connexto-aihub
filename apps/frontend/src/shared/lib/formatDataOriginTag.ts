export type RuntimeDataOriginTagVariant = 'executions' | 'catalog'

const demonstrativeLabel: Record<RuntimeDataOriginTagVariant, string> = {
  executions: 'DADOS DE EXEMPLO',
  catalog: 'DADOS ILUSTRATIVOS',
}

const unavailableLabel: Record<RuntimeDataOriginTagVariant, string> = {
  executions: 'HISTÓRICO INDISPONÍVEL',
  catalog: 'CATÁLOGO INDISPONÍVEL',
}

export function formatDataOriginTag(
  dataOrigin?: string,
  variant: RuntimeDataOriginTagVariant = 'executions',
): string {
  if (dataOrigin === 'demonstrative') {
    return demonstrativeLabel[variant]
  }
  if (dataOrigin === 'partial') {
    return 'DADOS PARCIAIS'
  }
  if (dataOrigin === 'unavailable') {
    return unavailableLabel[variant]
  }
  return 'DADOS OPERACIONAIS'
}

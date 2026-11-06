export function formatDataOriginTag(dataOrigin?: string): string {
  if (dataOrigin === 'demonstrative') {
    return 'DADOS DE EXEMPLO'
  }
  if (dataOrigin === 'partial') {
    return 'DADOS PARCIAIS'
  }
  if (dataOrigin === 'unavailable') {
    return 'HISTÓRICO INDISPONÍVEL'
  }
  return 'DADOS OPERACIONAIS'
}

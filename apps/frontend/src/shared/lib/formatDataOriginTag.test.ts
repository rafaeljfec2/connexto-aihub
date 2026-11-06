import { describe, expect, it } from 'vitest'
import { formatDataOriginTag } from './formatDataOriginTag'

describe('formatDataOriginTag', () => {
  it('uses route-specific copy for demonstrative and unavailable origins', () => {
    expect(formatDataOriginTag('demonstrative', 'executions')).toBe('DADOS DE EXEMPLO')
    expect(formatDataOriginTag('demonstrative', 'catalog')).toBe('DADOS ILUSTRATIVOS')
    expect(formatDataOriginTag('unavailable', 'executions')).toBe('HISTÓRICO INDISPONÍVEL')
    expect(formatDataOriginTag('unavailable', 'catalog')).toBe('CATÁLOGO INDISPONÍVEL')
  })

  it('shares partial and operational labels across routes', () => {
    expect(formatDataOriginTag('partial', 'catalog')).toBe('DADOS PARCIAIS')
    expect(formatDataOriginTag(undefined, 'executions')).toBe('DADOS OPERACIONAIS')
  })
})

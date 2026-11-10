import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { IntelligencePage } from './IntelligencePage'

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/intelligence']}>
      <IntelligencePage />
    </MemoryRouter>,
  )
}

describe('IntelligencePage', () => {
  it('renders page header with eyebrow, title, and badges', () => {
    renderPage()

    expect(screen.getByText('INTELLIGENCE / LEARNING ENGINE')).toBeTruthy()
    expect(screen.getByRole('heading', { level: 1, name: 'Aprendizagem & decisões' })).toBeTruthy()
    expect(screen.getByText('E08 · ATIVO')).toBeTruthy()
    expect(screen.getByText('E10 · PLANEJADO')).toBeTruthy()
  })

  it('renders notice banner with waiting status', () => {
    renderPage()

    expect(screen.getByText(/Learning Engine \(E10\) em planejamento/i)).toBeTruthy()
    expect(screen.getByText('AGUARDANDO E10')).toBeTruthy()
  })

  it('renders current routing and learning status cards', () => {
    renderPage()

    expect(screen.getByText('ROTEAMENTO ATUAL')).toBeTruthy()
    expect(screen.getByText('Determinístico & Auto')).toBeTruthy()
    expect(screen.getByText('REQUEST UNDERSTANDING')).toBeTruthy()
    expect(screen.getByText('Estrutural')).toBeTruthy()
    expect(screen.getByText('AVALIAÇÃO DE RESULTADOS')).toBeTruthy()
    expect(screen.getByText('Não ativada')).toBeTruthy()
    expect(screen.getByText('POLÍTICAS APRENDIDAS')).toBeTruthy()
    expect(screen.getByText('Indisponíveis')).toBeTruthy()
  })

  it('updates selected context when clicking context rows', () => {
    renderPage()

    const reasoningButton = screen.getByRole('button', { name: /Raciocínio e análise/i })
    fireEvent.click(reasoningButton)

    expect(screen.getByText('SEM RANKING')).toBeTruthy()
    expect(screen.getByText((content, element) => {
      return (
        element?.tagName.toLowerCase() === 'p' &&
        content.includes('Contexto selecionado:') &&
        content.includes('Categorias ilustrativas')
      )
    })).toBeTruthy()
  })

  it('renders decision flow steps and architectural callouts', () => {
    renderPage()

    expect(screen.getByText('Como a decisão acontece hoje')).toBeTruthy()
    expect(screen.getByText('Validar e projetar requisitos')).toBeTruthy()
    expect(screen.getByText('Verificar elegibilidade de candidatos')).toBeTruthy()
    expect(screen.getByText('Seleção determinística & virtual auto')).toBeTruthy()
    expect(screen.getByText('Avaliar resultados por contexto')).toBeTruthy()

    expect(screen.getByText('Sinais disponíveis para evoluir o E10')).toBeTruthy()
    expect(screen.getByText('Evolução do módulo')).toBeTruthy()
    expect(screen.getByText(/Princípio de produto:/i)).toBeTruthy()
  })
})

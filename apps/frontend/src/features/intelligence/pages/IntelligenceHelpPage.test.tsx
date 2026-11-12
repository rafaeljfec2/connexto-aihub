import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { IntelligenceHelpPage } from './IntelligenceHelpPage'

describe('IntelligenceHelpPage', () => {
  it('explains deterministic routing rules', () => {
    render(
      <MemoryRouter initialEntries={['/intelligence/help']}>
        <IntelligenceHelpPage />
      </MemoryRouter>,
    )

    expect(screen.getByRole('heading', { level: 1, name: 'Como o roteamento decide' })).toBeTruthy()
    expect(screen.getByRole('heading', { level: 2, name: 'Quem entra na disputa' })).toBeTruthy()
    expect(screen.getByRole('heading', { level: 2, name: 'Quem ganha' })).toBeTruthy()
    expect(screen.getByRole('heading', { level: 2, name: 'aihub/auto' })).toBeTruthy()
    expect(screen.getByText(/não lê o texto da mensagem/i)).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Voltar' }).getAttribute('href')).toBe('/intelligence')
  })
})
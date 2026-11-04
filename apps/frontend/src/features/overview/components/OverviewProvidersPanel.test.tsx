import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { OverviewProvidersPanel } from './OverviewProvidersPanel'

describe('OverviewProvidersPanel', () => {
  it('renders unmonitored provider with Sem leitura and correct description', () => {
    render(
      <MemoryRouter>
        <OverviewProvidersPanel
          providers={[{ id: 'azure-openai', enabled: true, state: 'operational' }]}
          quotas={[]}
        />
      </MemoryRouter>,
    )

    expect(screen.getByText('Consumo de franquias')).toBeTruthy()
    expect(screen.getByText('Azure OpenAI')).toBeTruthy()
    expect(screen.getByText('Franquia ainda não monitorada')).toBeTruthy()
    expect(screen.getByText('Sem leitura')).toBeTruthy()
  })

  it('renders provider with active quota utilization', () => {
    render(
      <MemoryRouter>
        <OverviewProvidersPanel
          providers={[{ id: 'opencode-go', enabled: true, state: 'operational' }]}
          quotas={[
            {
              policy_id: 1,
              scope: 'provider',
              provider_id: 'opencode-go',
              daily_limit: 1000,
              enabled: true,
              period_start: '2026-11-03T00:00:00Z',
              reset_at: '2026-11-04T00:00:00Z',
              admitted_count: 680,
              remaining_requests: 320,
              utilization_percent: 68,
            },
          ]}
        />
      </MemoryRouter>,
    )

    expect(screen.getByText('OpenCode')).toBeTruthy()
    expect(screen.getByText('68% consumido · cota diária')).toBeTruthy()
    expect(screen.getByText('32% disponível')).toBeTruthy()
  })

  it('renders provider with disabled quota as Não aplicável', () => {
    render(
      <MemoryRouter>
        <OverviewProvidersPanel
          providers={[{ id: 'openai', enabled: true, state: 'operational' }]}
          quotas={[
            {
              policy_id: 2,
              scope: 'provider',
              provider_id: 'openai',
              daily_limit: 500,
              enabled: false,
              period_start: '2026-11-03T00:00:00Z',
              reset_at: '2026-11-04T00:00:00Z',
              admitted_count: 0,
              remaining_requests: 500,
              utilization_percent: 0,
            },
          ]}
        />
      </MemoryRouter>,
    )

    expect(screen.getByText('OpenAI')).toBeTruthy()
    expect(screen.getByText('Política desabilitada')).toBeTruthy()
    expect(screen.getByText('Não aplicável')).toBeTruthy()
  })
})

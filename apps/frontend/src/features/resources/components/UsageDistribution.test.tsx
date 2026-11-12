import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { UsageDistribution } from './UsageDistribution'

describe('UsageDistribution', () => {
  it('renders empty state when distribution has no providers', () => {
    render(
      <UsageDistribution
        distribution={{ from: '', to: '', by_provider: [], by_model: [] }}
        usage={undefined}
        isLoading={false}
        isError={false}
      />,
    )

    expect(screen.getByText(/Nenhum uso por provedor/i)).toBeTruthy()
  })

  it('renders unavailable copy when the query failed', () => {
    render(<UsageDistribution distribution={undefined} usage={undefined} isError />)

    expect(screen.getByText(/Distribuição de uso indisponível/i)).toBeTruthy()
  })
})

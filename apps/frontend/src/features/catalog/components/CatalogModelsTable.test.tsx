import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { CatalogModelsTable } from './CatalogModelsTable'

describe('CatalogModelsTable', () => {
  it('renders empty state', () => {
    render(<CatalogModelsTable models={[]} isLoading={false} isError={false} />)
    expect(screen.getByText('No models match the current filters.')).toBeTruthy()
  })
})

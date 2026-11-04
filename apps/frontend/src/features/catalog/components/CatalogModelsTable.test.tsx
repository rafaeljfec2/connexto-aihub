import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { defaultCatalogColumnFilters } from '../lib/catalogColumnFilters'
import { CatalogModelsTable } from './CatalogModelsTable'

describe('CatalogModelsTable', () => {
  it('renders empty state when no rows on current page', () => {
    render(
      <CatalogModelsTable
        models={[]}
        isLoading={false}
        isError={false}
        columnFilters={defaultCatalogColumnFilters}
        onColumnFilterChange={vi.fn()}
        providerOptions={[]}
        listingOptions={[]}
      />,
    )
    expect(screen.getByText('Nenhum modelo corresponde aos filtros das colunas.')).toBeTruthy()
  })
})

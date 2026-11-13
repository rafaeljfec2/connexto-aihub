import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { defaultCatalogColumnFilters } from '../lib/catalogColumnFilters'
import type { CatalogModel } from '../types'
import { CatalogModelsTable } from './CatalogModelsTable'

const sampleModel: CatalogModel = {
  provider: 'openrouter',
  model_id: 'anthropic/claude-fable-5.1',
  name: 'Anthropic: Claude Fable 5.1',
  capabilities: {},
  presence: 'listed',
  freshness: 'fresh',
  provider_state: 'ok',
  execution_verification: 'unverified',
  listing_state: 'Encontrado',
  listing_reasons: [],
}

const tableProps = {
  columnFilters: defaultCatalogColumnFilters,
  onColumnFilterChange: vi.fn(),
  providerOptions: ['openrouter'],
  listingOptions: ['Encontrado'],
}

describe('CatalogModelsTable', () => {
  it('keeps a stable hook order when loading finishes', () => {
    const { rerender } = render(
      <CatalogModelsTable models={[]} isLoading isError={false} {...tableProps} />,
    )

    expect(screen.getByText('Carregando modelos do catálogo…')).toBeTruthy()

    rerender(
      <CatalogModelsTable
        models={[sampleModel]}
        isLoading={false}
        isError={false}
        {...tableProps}
      />,
    )

    expect(screen.getByText(sampleModel.name)).toBeTruthy()
  })

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

  it('copies the model identifier without selecting the row', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const onSelectModel = vi.fn()

    render(
      <CatalogModelsTable
        models={[sampleModel]}
        isLoading={false}
        isError={false}
        onSelectModel={onSelectModel}
        columnFilters={defaultCatalogColumnFilters}
        onColumnFilterChange={vi.fn()}
        providerOptions={['openrouter']}
        listingOptions={['Encontrado']}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /Copiar identificador anthropic\/claude-fable-5.1/i }))

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith('anthropic/claude-fable-5.1')
    })
    expect(onSelectModel).not.toHaveBeenCalled()
  })
})

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { AppSidebar } from './AppSidebar'

function renderSidebar(mobileOpen = false, dataOrigin = 'live') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ data_origin: dataOrigin }))))

  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <AppSidebar
          collapsed={false}
          mobileOpen={mobileOpen}
          connectionState="live"
          onToggleCollapsed={vi.fn()}
          onCloseMobile={vi.fn()}
        />
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe('AppSidebar focus behavior', () => {
  it('does not focus the sidebar container when the page mounts', async () => {
    const { container } = renderSidebar()

    const sidebar = container.querySelector('aside')
    expect(sidebar).not.toBeNull()

    await waitFor(() => {
      expect(document.activeElement).not.toBe(sidebar)
    })
  })

  it('focuses the first navigation link when opening the mobile drawer', async () => {
    const { container } = renderSidebar(true)

    const firstLink = container.querySelector('aside a')
    expect(firstLink).not.toBeNull()

    await waitFor(() => {
      expect(document.activeElement).toBe(firstLink)
    })
  })

  it('shows the demonstrative environment label when runtime metadata reports it', async () => {
    renderSidebar(false, 'demonstrative')

    expect(await screen.findByText('AMBIENTE DEMONSTRATIVO')).toBeTruthy()
  })

  it('shows unavailable navigation items without adding dead links', () => {
    renderSidebar()

    expect(screen.getByRole('link', { name: 'Inteligência' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Configurações' }).hasAttribute('disabled')).toBe(true)
    expect(screen.queryByRole('link', { name: 'Playground' })).toBeNull()
  })
})

import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { RefreshButton } from './refresh-button'

describe('RefreshButton', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('spins the refresh icon while pending and keeps spinning for the minimum duration', async () => {
    vi.useFakeTimers()
    let resolveRefresh: () => void = () => {}
    const onRefresh = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveRefresh = resolve
        }),
    )

    render(<RefreshButton onRefresh={onRefresh} minSpinMs={450} />)

    const button = screen.getByRole('button', { name: 'Atualizar' })
    const icon = button.querySelector('svg')
    expect(icon?.className.baseVal ?? icon?.getAttribute('class') ?? '').not.toContain('animate-spin')

    fireEvent.click(button)

    expect(onRefresh).toHaveBeenCalledTimes(1)
    expect(button.hasAttribute('disabled')).toBe(true)
    expect(icon?.getAttribute('class') ?? '').toContain('animate-spin')

    await act(async () => {
      resolveRefresh()
    })

    expect(button.hasAttribute('disabled')).toBe(true)
    expect(icon?.getAttribute('class') ?? '').toContain('animate-spin')

    act(() => {
      vi.advanceTimersByTime(450)
    })

    expect(button.hasAttribute('disabled')).toBe(false)
    expect(icon?.getAttribute('class') ?? '').not.toContain('animate-spin')
  })

  it('keeps label unchanged and recovers when refresh rejects', async () => {
    const onRefresh = vi.fn().mockRejectedValue(new Error('network error'))

    render(<RefreshButton onRefresh={onRefresh} minSpinMs={0} />)

    const button = screen.getByRole('button', { name: 'Atualizar' })
    await act(async () => {
      fireEvent.click(button)
    })

    expect(button.hasAttribute('disabled')).toBe(false)
    expect(screen.getByText('Atualizar')).toBeDefined()
  })
})

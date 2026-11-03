import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { StatusCard } from './StatusCard'

describe('StatusCard', () => {
  it('renders loading state', () => {
    const { container } = render(<StatusCard title="System liveness" isLoading />)
    expect(container.querySelector('.animate-pulse')).toBeTruthy()
  })

  it('renders status with semantic class', () => {
    render(<StatusCard title="System liveness" isLoading={false} status="healthy" />)
    const status = screen.getByText('HEALTHY')
    expect(status.className).toContain('status-healthy')
  })
})

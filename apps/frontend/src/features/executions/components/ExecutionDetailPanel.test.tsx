import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ExecutionDetailPanel } from './ExecutionDetailPanel'

describe('ExecutionDetailPanel', () => {
  it('shows API error message when detail load fails', () => {
    render(
      <ExecutionDetailPanel
        record={undefined}
        isLoading={false}
        isError
        errorMessage="execution record not found"
        selectedId={'a'.repeat(32)}
      />,
    )
    expect(screen.getByRole('alert').textContent).toContain('execution record not found')
  })
})

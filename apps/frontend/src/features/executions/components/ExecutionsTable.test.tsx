import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { ExecutionRecord } from '../types'
import { ExecutionsTable } from './ExecutionsTable'

const record: ExecutionRecord = {
  schema_version: 1,
  execution_id: 'b'.repeat(32),
  request_id: 'req-1',
  provider_id: 'opencode-go',
  model_id: 'vendor/model',
  started_at: '2026-11-03T10:00:00.000Z',
  completed_at: '2026-11-03T10:00:01.000Z',
  duration_ns: 50_000_000,
  outcome: 'completed',
  phase: 'forwarding',
  cause: null,
  upstream_status: 200,
  response_committed: true,
  usage: null,
}

describe('ExecutionsTable', () => {
  it('renders empty state when no rows match search', () => {
    render(
      <ExecutionsTable
        records={[record]}
        search="no-match"
        selectedId={null}
        onSelect={() => undefined}
        isLoading={false}
        isError={false}
      />,
    )
    expect(screen.getByText(/No execution records match/)).toBeTruthy()
  })

  it('does not render prompt or response columns', () => {
    const { container } = render(
      <ExecutionsTable
        records={[record]}
        search=""
        selectedId={null}
        onSelect={() => undefined}
        isLoading={false}
        isError={false}
      />,
    )
    expect(container.textContent).not.toMatch(/prompt/i)
    expect(container.textContent).not.toMatch(/response body/i)
    expect(screen.getByText('vendor/model')).toBeTruthy()
  })
})

import { describe, expect, it } from 'vitest'
import { cursorCollectReady, messageForCursorCollectReadiness } from './cursorCollectReadiness'

describe('cursorCollectReadiness', () => {
  it('requires authenticated session and non-auth_required collect', () => {
    expect(
      cursorCollectReady(
        { status: 'authenticated', accountId: 'cursor' },
        { status: 'available', accountId: 'cursor' },
      ),
    ).toBe(true)
    expect(
      cursorCollectReady(
        { status: 'authenticated', accountId: 'cursor' },
        { status: 'authentication_required', accountId: 'cursor' },
      ),
    ).toBe(false)
  })

  it('maps probe failure to a red message', () => {
    const msg = messageForCursorCollectReadiness(
      { status: 'authenticated', accountId: 'cursor', touchedAt: '2026-01-01' },
      { status: 'authentication_required', accountId: 'cursor' },
    )
    expect(msg.success).toBe(false)
    expect(msg.message).toMatch(/authentication_required/)
  })
})

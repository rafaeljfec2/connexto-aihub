import { describe, expect, it } from 'vitest'
import { cursorAuthStartSuccessMessage } from './cursorAuthStartMessage'

describe('cursorAuthStartSuccessMessage', () => {
  it('explains headless Docker when no desktop window', () => {
    const message = cursorAuthStartSuccessMessage({
      status: 'authentication_pending',
      sessionId: 's1',
      expiresAt: '2026-11-07T00:00:00Z',
      interactionUrl: 'http://127.0.0.1:9090/v1/providers/cursor/auth/s1/interaction',
      browserMode: 'headless',
    })
    expect(message).toContain('nenhuma janela Chrome')
    expect(message).toContain('pnpm dev')
  })

  it('mentions Chrome window for headed mode', () => {
    const message = cursorAuthStartSuccessMessage({
      status: 'authentication_pending',
      sessionId: 's1',
      expiresAt: '2026-11-07T00:00:00Z',
      interactionUrl: 'http://proxy/interaction',
      browserMode: 'headed',
    })
    expect(message).toContain('Sign in — Google Chrome')
  })
})

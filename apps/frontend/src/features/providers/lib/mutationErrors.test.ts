import { describe, expect, it } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import { cursorProxyAuthErrorMessage } from './mutationErrors'

describe('cursorProxyAuthErrorMessage', () => {
  it('surfaces Hub message for proxy_cursor_auth_failed', () => {
    const err = new ApiError(502, {
      type: 'proxy_cursor_auth_failed',
      message:
        'Não foi possível abrir o Chromium no proxy. Detalhe do proxy: profile appears to be in use',
    })
    expect(cursorProxyAuthErrorMessage(err)).toContain('Detalhe do proxy')
    expect(cursorProxyAuthErrorMessage(err)).not.toMatch(/Antigravity/i)
  })

  it('passes through legacy upstream detail strings', () => {
    const err = new ApiError(502, {
      type: 'proxy_cursor_auth_failed',
      message: 'the connexto proxy rejected the Cursor usage auth request (upstream busy)',
    })
    expect(cursorProxyAuthErrorMessage(err)).toContain('upstream busy')
  })
})

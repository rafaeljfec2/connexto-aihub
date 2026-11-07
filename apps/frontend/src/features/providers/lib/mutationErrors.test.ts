import { describe, expect, it } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import { cursorProxyAuthErrorMessage } from './mutationErrors'

describe('cursorProxyAuthErrorMessage', () => {
  it('maps proxy_cursor_auth_failed Chromium hints to PT-BR', () => {
    const err = new ApiError(502, {
      type: 'proxy_cursor_auth_failed',
      message:
        'Cursor usage auth requires Chromium on the Connexto Proxy (configure CHROMIUM_EXECUTABLE_PATH and restart the proxy).',
    })
    expect(cursorProxyAuthErrorMessage(err)).toMatch(/Chromium indisponível no proxy/)
    expect(cursorProxyAuthErrorMessage(err)).not.toMatch(/Antigravity/i)
  })

  it('maps generic proxy_cursor_auth_failed with detail', () => {
    const err = new ApiError(502, {
      type: 'proxy_cursor_auth_failed',
      message: 'the connexto proxy rejected the Cursor usage auth request (upstream busy)',
    })
    expect(cursorProxyAuthErrorMessage(err)).toContain('Falha ao iniciar login web do Cursor no proxy.')
    expect(cursorProxyAuthErrorMessage(err)).toContain('upstream busy')
  })
})

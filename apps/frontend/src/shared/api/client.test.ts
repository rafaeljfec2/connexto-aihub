import { afterEach, describe, expect, it, vi } from 'vitest'
import { setUiAuthSystem } from '../../features/auth/api/uiAuthSystem'
import { fetchJson } from './client'

describe('fetchJson 401 refresh', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    setUiAuthSystem('unknown')
  })

  it('retries once after a successful refresh', async () => {
    setUiAuthSystem('users')
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(new Response(null, { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ok: true }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    const result = await fetchJson<{ ok: boolean }>('/api/overview')
    expect(result.ok).toBe(true)
    expect(fetchMock).toHaveBeenCalledTimes(3)
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/ui/refresh')
  })

  it('does not refresh in operator auth system', async () => {
    setUiAuthSystem('operator')
    const fetchMock = vi.fn().mockResolvedValueOnce(new Response(null, { status: 401 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(fetchJson('/api/overview')).rejects.toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('does not refresh for login endpoint', async () => {
    setUiAuthSystem('users')
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ error: { type: 'unauthorized', message: 'x' } }), { status: 401 }),
      )
    vi.stubGlobal('fetch', fetchMock)

    await expect(fetchJson('/api/ui/login', { method: 'POST' })).rejects.toBeDefined()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})

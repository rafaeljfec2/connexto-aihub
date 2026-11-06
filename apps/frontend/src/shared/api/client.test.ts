import { describe, expect, it } from 'vitest'
import { ApiError } from './client'

describe('ApiError', () => {
  it('preserves server error type', () => {
    const error = new ApiError(502, { type: 'upstream_failure', message: 'failed' })
    expect(error.type).toBe('upstream_failure')
    expect(error.message).toBe('failed')
  })
})

describe('fetchJson', () => {
  it('parses gateway error envelope', async () => {
    const { fetchJson, ApiError } = await import('./client')
    globalThis.fetch = (() =>
      Promise.resolve({
        ok: false,
        status: 502,
        statusText: 'Bad Gateway',
        json: () =>
          Promise.resolve({
            error: { type: 'proxy_antigravity_auth_failed', message: 'upstream said no' },
          }),
      })) as unknown as typeof fetch

    const err = await fetchJson('/api/x').catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    const apiErr = err as InstanceType<typeof ApiError>
    expect(apiErr.status).toBe(502)
    expect(apiErr.type).toBe('proxy_antigravity_auth_failed')
    expect(apiErr.message).toBe('upstream said no')
  })
})

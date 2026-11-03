import { describe, expect, it, vi, afterEach } from 'vitest'
import { streamResponses } from './streamResponses'

describe('streamResponses', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('rejects when the abort signal fires during fetch', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((_input: RequestInfo, init?: RequestInit) =>
        new Promise<Response>((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () => {
            reject(new DOMException('Aborted', 'AbortError'))
          })
        }),
      ),
    )

    const controller = new AbortController()
    const promise = streamResponses({
      settings: {
        providerId: 'opencode_go',
        modelId: 'test-model',
        systemPrompt: 'sys',
        temperature: 0.5,
      },
      messages: [{ id: '1', role: 'user', content: 'hi', timestamp: 1 }],
      signal: controller.signal,
      onEvent: () => undefined,
    })

    controller.abort()
    await expect(promise).rejects.toMatchObject({ name: 'AbortError' })
  })
})

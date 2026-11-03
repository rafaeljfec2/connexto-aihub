import { describe, expect, it } from 'vitest'
import { initialPlaygroundState, playgroundReducer } from './playgroundReducer'

describe('playgroundReducer', () => {
  it('transitions through send, delta, and complete', () => {
    let state = initialPlaygroundState
    state = playgroundReducer(state, {
      type: 'send_start',
      userMessage: { id: 'u1', role: 'user', content: 'Hi', timestamp: 1 },
      assistantMessage: { id: 'a1', role: 'assistant', content: '', timestamp: 2, status: 'streaming' },
    })
    expect(state.phase).toBe('submitting')

    state = playgroundReducer(state, { type: 'first_token', messageId: 'a1', ttftMs: 120 })
    expect(state.phase).toBe('streaming')

    state = playgroundReducer(state, { type: 'append_delta', messageId: 'a1', delta: 'Hello' })
    state = playgroundReducer(state, {
      type: 'complete',
      messageId: 'a1',
      durationMs: 500,
      usage: { input_tokens: 1, output_tokens: 2, total_tokens: 3 },
    })

    expect(state.phase).toBe('completed')
    expect(state.messages[1]?.content).toBe('Hello')
    expect(state.messages[1]?.metrics?.totalTokens).toBe(3)
  })

  it('marks cancellation without dropping prior content', () => {
    let state = initialPlaygroundState
    state = playgroundReducer(state, {
      type: 'send_start',
      userMessage: { id: 'u1', role: 'user', content: 'Hi', timestamp: 1 },
      assistantMessage: { id: 'a1', role: 'assistant', content: 'Partial', timestamp: 2, status: 'streaming' },
    })
    state = playgroundReducer(state, { type: 'cancel', messageId: 'a1' })
    expect(state.phase).toBe('cancelled')
    expect(state.messages[1]?.content).toBe('Partial')
    expect(state.messages[1]?.status).toBe('cancelled')
  })
})

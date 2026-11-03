import { useCallback, useReducer, useRef } from 'react'
import { ApiError } from '../../../shared/api/client'
import { streamResponses } from '../api/streamResponses'
import type { ResponsesSseEvent } from '../lib/responsesSseParser'
import type { ChatMessage } from '../types'
import { initialPlaygroundState, playgroundReducer } from './playgroundReducer'

function createMessage(role: ChatMessage['role'], content: string): ChatMessage {
  return {
    id: `${role}-${crypto.randomUUID()}`,
    role,
    content,
    timestamp: Date.now(),
  }
}

export function usePlayground() {
  const [state, dispatch] = useReducer(playgroundReducer, initialPlaygroundState)
  const abortRef = useRef<AbortController | null>(null)

  const cancel = useCallback(() => {
    abortRef.current?.abort()
  }, [])

  const sendMessage = useCallback(async () => {
    const trimmed = state.draft.trim()
    if (!trimmed) {
      return
    }
    if (!state.settings.providerId || !state.settings.modelId) {
      dispatch({
        type: 'set_error',
        errorMessage: 'Select a provider and model before sending.',
      })
      return
    }

    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller

    const userMessage = createMessage('user', trimmed)
    const assistantMessage: ChatMessage = {
      ...createMessage('assistant', ''),
      status: 'streaming',
    }
    const history = [...state.messages, userMessage]

    dispatch({ type: 'send_start', userMessage, assistantMessage })

    const startedAt = performance.now()
    let firstTokenAt: number | undefined
    let streamCompleted = false
    let streamFailed = false

    const handleEvent = (event: ResponsesSseEvent): void => {
      if (event.kind === 'text_delta') {
        if (firstTokenAt === undefined) {
          firstTokenAt = performance.now()
          dispatch({
            type: 'first_token',
            messageId: assistantMessage.id,
            ttftMs: Math.round(firstTokenAt - startedAt),
          })
        }
        dispatch({ type: 'append_delta', messageId: assistantMessage.id, delta: event.delta })
      }
      if (event.kind === 'completed') {
        streamCompleted = true
        dispatch({
          type: 'complete',
          messageId: assistantMessage.id,
          durationMs: Math.round(performance.now() - startedAt),
          usage: event.usage,
        })
      }
      if (event.kind === 'error') {
        streamFailed = true
        dispatch({
          type: 'fail',
          messageId: assistantMessage.id,
          errorMessage: event.message,
        })
      }
    }

    try {
      await streamResponses({
        settings: state.settings,
        messages: history,
        signal: controller.signal,
        onEvent: handleEvent,
      })
      if (!streamCompleted && !streamFailed && !controller.signal.aborted) {
        dispatch({
          type: 'complete',
          messageId: assistantMessage.id,
          durationMs: Math.round(performance.now() - startedAt),
        })
      }
    } catch (error) {
      if (controller.signal.aborted) {
        dispatch({ type: 'cancel', messageId: assistantMessage.id })
        return
      }
      const message = error instanceof ApiError ? error.message : error instanceof Error ? error.message : 'Request failed'
      dispatch({ type: 'fail', messageId: assistantMessage.id, errorMessage: message })
    } finally {
      if (abortRef.current === controller) {
        abortRef.current = null
      }
    }
  }, [state.draft, state.messages, state.settings])

  return {
    state,
    dispatch,
    sendMessage,
    cancel,
    isStreaming: state.phase === 'submitting' || state.phase === 'streaming',
  }
}

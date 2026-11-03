import { ApiError } from '../../../shared/api/client'
import { buildResponsesBody } from '../lib/buildResponsesBody'
import { feedSseBuffer, type ResponsesSseEvent } from '../lib/responsesSseParser'
import type { ChatMessage, PlaygroundSettings } from '../types'

export type StreamResponsesParams = {
  settings: PlaygroundSettings
  messages: ChatMessage[]
  signal: AbortSignal
  onEvent: (event: ResponsesSseEvent) => void
}

export async function streamResponses(params: StreamResponsesParams): Promise<void> {
  const body = buildResponsesBody({
    model: params.settings.modelId,
    systemPrompt: params.settings.systemPrompt,
    messages: params.messages,
    temperature: params.settings.temperature,
    maxOutputTokens: params.settings.maxOutputTokens,
  })

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (params.settings.providerId) {
    headers['X-AIHub-Provider'] = params.settings.providerId
  }

  const response = await fetch('/v1/responses', {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
    signal: params.signal,
  })

  if (!response.ok) {
    let errorBody = { type: 'request_failed', message: response.statusText }
    try {
      errorBody = (await response.json()) as typeof errorBody
    } catch {
      // keep default
    }
    throw new ApiError(response.status, errorBody)
  }

  if (!response.body) {
    throw new Error('Streaming body is unavailable')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) {
      break
    }
    const chunk = decoder.decode(value, { stream: true })
    const fed = feedSseBuffer(buffer, chunk)
    buffer = fed.rest
    for (const event of fed.events) {
      params.onEvent(event)
    }
  }

  if (buffer.trim()) {
    const fed = feedSseBuffer(buffer, '\n\n')
    for (const event of fed.events) {
      params.onEvent(event)
    }
  }
}

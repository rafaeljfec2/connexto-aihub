export type ResponsesSseEvent =
  | { kind: 'created' }
  | { kind: 'text_delta'; delta: string }
  | {
      kind: 'completed'
      usage?: {
        input_tokens?: number
        output_tokens?: number
        total_tokens?: number
      }
    }
  | { kind: 'error'; type: string; message: string }
  | { kind: 'ignored'; eventName: string }

export function parseSseBlock(block: string): ResponsesSseEvent | undefined {
  const lines = block.split('\n').map((line) => line.trimEnd())
  let eventName = 'message'
  const dataLines: string[] = []

  for (const line of lines) {
    if (line.startsWith('event:')) {
      eventName = line.slice('event:'.length).trim()
      continue
    }
    if (line.startsWith('data:')) {
      dataLines.push(line.slice('data:'.length).trim())
    }
  }

  if (dataLines.length === 0) {
    return undefined
  }

  const rawData = dataLines.join('\n')
  let payload: Record<string, unknown> = {}
  try {
    payload = JSON.parse(rawData) as Record<string, unknown>
  } catch {
    return { kind: 'error', type: 'parse_error', message: 'Invalid SSE JSON payload' }
  }

  if (eventName === 'response.created' || payload.type === 'response.created') {
    return { kind: 'created' }
  }

  if (eventName === 'response.output_text.delta') {
    const delta = payload.delta
    return typeof delta === 'string' ? { kind: 'text_delta', delta } : { kind: 'ignored', eventName }
  }

  if (eventName === 'response.completed' || payload.type === 'response.completed') {
    const usage = payload.usage
    if (usage && typeof usage === 'object' && !Array.isArray(usage)) {
      const record = usage as Record<string, unknown>
      return {
        kind: 'completed',
        usage: {
          input_tokens: typeof record.input_tokens === 'number' ? record.input_tokens : undefined,
          output_tokens: typeof record.output_tokens === 'number' ? record.output_tokens : undefined,
          total_tokens: typeof record.total_tokens === 'number' ? record.total_tokens : undefined,
        },
      }
    }
    return { kind: 'completed' }
  }

  if (eventName === 'error' || payload.type === 'error') {
    const message = typeof payload.message === 'string' ? payload.message : 'Streaming error'
    const type = typeof payload.type === 'string' ? payload.type : eventName
    return { kind: 'error', type, message }
  }

  return { kind: 'ignored', eventName }
}

export function feedSseBuffer(
  buffer: string,
  chunk: string,
): { events: ResponsesSseEvent[]; rest: string } {
  const combined = buffer + chunk
  const parts = combined.split('\n\n')
  const rest = parts.pop() ?? ''
  const events: ResponsesSseEvent[] = []

  for (const part of parts) {
    const trimmed = part.trim()
    if (!trimmed) {
      continue
    }
    const parsed = parseSseBlock(trimmed)
    if (parsed) {
      events.push(parsed)
    }
  }

  return { events, rest }
}

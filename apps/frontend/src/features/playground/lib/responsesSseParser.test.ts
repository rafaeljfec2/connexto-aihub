import { describe, expect, it } from 'vitest'
import { feedSseBuffer, parseSseBlock } from './responsesSseParser'

const SAMPLE_EVENTS = [
  'event: response.created\ndata: {"type":"response.created"}\n\n',
  'event: response.output_text.delta\ndata: {"delta":"hel"}\n\n',
  'event: response.output_text.delta\ndata: {"delta":"lo"}\n\n',
  'event: response.completed\ndata: {"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}\n\n',
].join('')

describe('responsesSseParser', () => {
  it('parses a single SSE block', () => {
    const event = parseSseBlock('event: response.output_text.delta\ndata: {"delta":"hello"}')
    expect(event).toEqual({ kind: 'text_delta', delta: 'hello' })
  })

  it('parses fragmented chunks across block boundaries', () => {
    let buffer = ''
    const deltas: string[] = []
    const chunkSize = 17
    for (let index = 0; index < SAMPLE_EVENTS.length; index += chunkSize) {
      const chunk = SAMPLE_EVENTS.slice(index, index + chunkSize)
      const fed = feedSseBuffer(buffer, chunk)
      buffer = fed.rest
      for (const event of fed.events) {
        if (event.kind === 'text_delta') {
          deltas.push(event.delta)
        }
      }
    }
    const tail = feedSseBuffer(buffer, '\n\n')
    for (const event of tail.events) {
      if (event.kind === 'text_delta') {
        deltas.push(event.delta)
      }
    }
    expect(deltas.join('')).toBe('hello')
  })

  it('handles multiple events in one chunk', () => {
    const chunk =
      'event: response.output_text.delta\ndata: {"delta":"a"}\n\n' +
      'event: response.output_text.delta\ndata: {"delta":"b"}\n\n'
    const { events } = feedSseBuffer('', chunk)
    expect(events.filter((event) => event.kind === 'text_delta')).toHaveLength(2)
  })

  it('surfaces parse errors for invalid JSON', () => {
    const event = parseSseBlock('event: response.output_text.delta\ndata: {not-json')
    expect(event?.kind).toBe('error')
  })
})

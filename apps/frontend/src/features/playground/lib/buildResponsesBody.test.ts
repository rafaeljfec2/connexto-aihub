import { describe, expect, it } from 'vitest'
import { buildResponsesBody } from './buildResponsesBody'

describe('buildResponsesBody', () => {
  it('builds OpenAI Responses payload with instructions and input', () => {
    const body = buildResponsesBody({
      model: 'deepseek-v4.1-flash',
      systemPrompt: 'Be concise',
      temperature: 0.4,
      messages: [
        {
          id: '1',
          role: 'user',
          content: 'Hello',
          timestamp: 1,
        },
        {
          id: '2',
          role: 'assistant',
          content: 'Hi',
          timestamp: 2,
        },
      ],
    })

    expect(body.stream).toBe(true)
    expect(body.model).toBe('deepseek-v4.1-flash')
    expect(body.instructions).toBe('Be concise')
    expect(body.input).toHaveLength(2)
    expect(body.input[0]?.content[0]?.type).toBe('input_text')
    expect(body.input[1]?.content[0]?.type).toBe('output_text')
  })
})

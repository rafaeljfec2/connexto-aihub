import type { ChatMessage } from '../types'

export type ResponsesRequestBody = {
  model: string
  instructions: string
  input: Array<{
    role: 'user' | 'assistant'
    content: Array<{ type: 'input_text' | 'output_text'; text: string }>
  }>
  stream: true
  temperature?: number
  max_output_tokens?: number
}

export function buildResponsesBody(params: {
  model: string
  systemPrompt: string
  messages: ChatMessage[]
  temperature: number
  maxOutputTokens?: number
}): ResponsesRequestBody {
  const input = params.messages
    .filter((message) => message.role === 'user' || message.role === 'assistant')
    .map((message) => ({
      role: message.role,
      content: [
        {
          type: message.role === 'user' ? ('input_text' as const) : ('output_text' as const),
          text: message.content,
        },
      ],
    }))

  const body: ResponsesRequestBody = {
    model: params.model,
    instructions: params.systemPrompt,
    input,
    stream: true,
    temperature: params.temperature,
  }

  if (params.maxOutputTokens !== undefined && params.maxOutputTokens > 0) {
    body.max_output_tokens = params.maxOutputTokens
  }

  return body
}

export type ChatRole = 'user' | 'assistant'

export type ChatMessage = {
  id: string
  role: ChatRole
  content: string
  timestamp: number
  metrics?: {
    ttftMs?: number
    totalDurationMs?: number
    inputTokens?: number
    outputTokens?: number
    totalTokens?: number
  }
  status?: 'streaming' | 'complete' | 'error' | 'cancelled'
}

export type PlaygroundRunPhase = 'idle' | 'submitting' | 'streaming' | 'completed' | 'error' | 'cancelled'

export type PlaygroundSettings = {
  providerId: string
  modelId: string
  systemPrompt: string
  temperature: number
  maxOutputTokens?: number
}

export type PlaygroundState = {
  settings: PlaygroundSettings
  messages: ChatMessage[]
  draft: string
  phase: PlaygroundRunPhase
  errorMessage?: string
}

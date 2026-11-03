import type { ChatMessage, PlaygroundRunPhase, PlaygroundSettings, PlaygroundState } from '../types'

export type PlaygroundAction =
  | { type: 'patch_settings'; patch: Partial<PlaygroundSettings> }
  | { type: 'set_draft'; draft: string }
  | { type: 'clear_conversation' }
  | { type: 'send_start'; userMessage: ChatMessage; assistantMessage: ChatMessage }
  | { type: 'first_token'; messageId: string; ttftMs: number }
  | { type: 'append_delta'; messageId: string; delta: string }
  | {
      type: 'complete'
      messageId: string
      durationMs: number
      usage?: { input_tokens?: number; output_tokens?: number; total_tokens?: number }
    }
  | { type: 'fail'; messageId: string; errorMessage: string }
  | { type: 'set_error'; errorMessage: string }
  | { type: 'cancel'; messageId: string }

export const initialPlaygroundState: PlaygroundState = {
  settings: {
    providerId: '',
    modelId: '',
    systemPrompt: 'You are a helpful assistant.',
    temperature: 0.7,
  },
  messages: [],
  draft: '',
  phase: 'idle',
}

function updateMessage(
  messages: ChatMessage[],
  messageId: string,
  patch: Partial<ChatMessage>,
): ChatMessage[] {
  return messages.map((message) => (message.id === messageId ? { ...message, ...patch } : message))
}

export function playgroundReducer(state: PlaygroundState, action: PlaygroundAction): PlaygroundState {
  switch (action.type) {
    case 'patch_settings':
      return {
        ...state,
        settings: { ...state.settings, ...action.patch },
        errorMessage: undefined,
      }
    case 'set_draft':
      return { ...state, draft: action.draft }
    case 'clear_conversation':
      return {
        ...state,
        messages: [],
        draft: '',
        phase: 'idle',
        errorMessage: undefined,
      }
    case 'send_start':
      return {
        ...state,
        messages: [...state.messages, action.userMessage, action.assistantMessage],
        draft: '',
        phase: 'submitting',
        errorMessage: undefined,
      }
    case 'first_token':
      return {
        ...state,
        phase: 'streaming',
        messages: updateMessage(state.messages, action.messageId, {
          metrics: { ttftMs: action.ttftMs },
          status: 'streaming',
        }),
      }
    case 'append_delta':
      return {
        ...state,
        phase: 'streaming',
        messages: state.messages.map((message) =>
          message.id === action.messageId
            ? { ...message, content: message.content + action.delta, status: 'streaming' }
            : message,
        ),
      }
    case 'complete': {
      const phase: PlaygroundRunPhase = 'completed'
      return {
        ...state,
        phase,
        messages: updateMessage(state.messages, action.messageId, {
          status: 'complete',
          metrics: {
            ttftMs: state.messages.find((message) => message.id === action.messageId)?.metrics?.ttftMs,
            totalDurationMs: action.durationMs,
            inputTokens: action.usage?.input_tokens,
            outputTokens: action.usage?.output_tokens,
            totalTokens: action.usage?.total_tokens,
          },
        }),
      }
    }
    case 'fail':
      return {
        ...state,
        phase: 'error',
        errorMessage: action.errorMessage,
        messages: updateMessage(state.messages, action.messageId, {
          status: 'error',
        }),
      }
    case 'set_error':
      return {
        ...state,
        phase: 'error',
        errorMessage: action.errorMessage,
      }
    case 'cancel':
      return {
        ...state,
        phase: 'cancelled',
        messages: updateMessage(state.messages, action.messageId, {
          status: 'cancelled',
        }),
      }
    default:
      return state
  }
}

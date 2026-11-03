import type { ChatMessage } from '../types'
import { PlaygroundMessageItem } from './PlaygroundMessageItem'

type PlaygroundMessageListProps = {
  messages: ChatMessage[]
}

export function PlaygroundMessageList({ messages }: PlaygroundMessageListProps) {
  if (messages.length === 0) {
    return (
      <div className="playground-empty" role="status">
        <p>No messages yet. Select a provider and model, then send a prompt.</p>
      </div>
    )
  }

  return (
    <ul className="playground-messages" aria-live="polite">
      {messages.map((message) => (
        <li key={message.id}>
          <PlaygroundMessageItem message={message} />
        </li>
      ))}
    </ul>
  )
}

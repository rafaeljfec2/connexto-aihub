import type { ChatMessage } from '../types'

type PlaygroundMessageItemProps = {
  message: ChatMessage
}

export function PlaygroundMessageItem({ message }: PlaygroundMessageItemProps) {
  return (
    <article className={`playground-message playground-message-${message.role}`}>
      <header>
        <strong>{message.role === 'user' ? 'You' : 'Assistant'}</strong>
        {message.status === 'streaming' ? <span className="streaming-dot">Streaming…</span> : null}
        {message.status === 'cancelled' ? <span className="muted">Cancelled</span> : null}
        {message.status === 'error' ? <span className="field-error">Error</span> : null}
      </header>
      <p>{message.content || (message.status === 'streaming' ? '…' : '')}</p>
      {message.metrics ? (
        <footer className="playground-metrics">
          {message.metrics.ttftMs !== undefined ? <span>TTFT: {message.metrics.ttftMs}ms</span> : null}
          {message.metrics.totalDurationMs !== undefined ? (
            <span>Duration: {message.metrics.totalDurationMs}ms</span>
          ) : null}
          {message.metrics.totalTokens !== undefined ? (
            <span>Tokens: {message.metrics.totalTokens}</span>
          ) : null}
        </footer>
      ) : null}
    </article>
  )
}

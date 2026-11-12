import { useTranslation } from 'react-i18next'
import type { ChatMessage } from '../types'

type PlaygroundMessageItemProps = {
  message: ChatMessage
}

export function PlaygroundMessageItem({ message }: PlaygroundMessageItemProps) {
  const { t } = useTranslation('playground')

  return (
    <article className={`playground-message playground-message-${message.role}`}>
      <header>
        <strong>{message.role === 'user' ? t('message.you') : t('message.assistant')}</strong>
        {message.status === 'streaming' ? (
          <span className="streaming-dot">{t('message.streaming')}</span>
        ) : null}
        {message.status === 'cancelled' ? <span className="muted">{t('message.cancelled')}</span> : null}
        {message.status === 'error' ? <span className="field-error">{t('message.error')}</span> : null}
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

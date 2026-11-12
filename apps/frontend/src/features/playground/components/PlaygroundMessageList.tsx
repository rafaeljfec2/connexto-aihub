import { useTranslation } from 'react-i18next'
import type { ChatMessage } from '../types'
import { PlaygroundMessageItem } from './PlaygroundMessageItem'

type PlaygroundMessageListProps = {
  messages: ChatMessage[]
}

export function PlaygroundMessageList({ messages }: PlaygroundMessageListProps) {
  const { t } = useTranslation('playground')

  if (messages.length === 0) {
    return (
      <div className="playground-empty" role="status">
        <p>{t('message.emptyList')}</p>
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

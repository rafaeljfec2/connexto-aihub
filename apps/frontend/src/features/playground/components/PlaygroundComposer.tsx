import { useTranslation } from 'react-i18next'

type PlaygroundComposerProps = {
  draft: string
  disabled: boolean
  isStreaming: boolean
  onDraftChange: (value: string) => void
  onSend: () => void
  onStop: () => void
}

export function PlaygroundComposer({
  draft,
  disabled,
  isStreaming,
  onDraftChange,
  onSend,
  onStop,
}: PlaygroundComposerProps) {
  const { t } = useTranslation('playground')

  return (
    <div className="playground-composer">
      <label className="sr-only" htmlFor="playground-prompt">
        {t('composer.messageLabel')}
      </label>
      <textarea
        id="playground-prompt"
        rows={4}
        value={draft}
        disabled={disabled && !isStreaming}
        placeholder={t('composer.placeholder')}
        onChange={(event) => onDraftChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
            event.preventDefault()
            if (!isStreaming) {
              onSend()
            }
          }
        }}
      />
      <div className="playground-composer-actions">
        {isStreaming ? (
          <button type="button" className="ghost-button" onClick={onStop}>
            {t('composer.stopGeneration')}
          </button>
        ) : (
          <button type="button" disabled={disabled || draft.trim().length === 0} onClick={onSend}>
            {t('composer.send')}
          </button>
        )}
      </div>
    </div>
  )
}

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
  return (
    <div className="playground-composer">
      <label className="sr-only" htmlFor="playground-prompt">
        Message
      </label>
      <textarea
        id="playground-prompt"
        rows={4}
        value={draft}
        disabled={disabled && !isStreaming}
        placeholder="Write a prompt to test the selected model…"
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
            Stop generation
          </button>
        ) : (
          <button type="button" disabled={disabled || draft.trim().length === 0} onClick={onSend}>
            Send
          </button>
        )}
      </div>
    </div>
  )
}

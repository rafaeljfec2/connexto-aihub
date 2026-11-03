import type { PlaygroundSettings } from '../types'

type PlaygroundSettingsPanelProps = {
  settings: PlaygroundSettings
  disabled: boolean
  onChange: (patch: Partial<PlaygroundSettings>) => void
}

export function PlaygroundSettingsPanel({ settings, disabled, onChange }: PlaygroundSettingsPanelProps) {
  return (
    <div className="playground-settings">
      <label>
        System instructions
        <textarea
          rows={3}
          value={settings.systemPrompt}
          disabled={disabled}
          onChange={(event) => onChange({ systemPrompt: event.target.value })}
        />
      </label>
      <div className="playground-settings-row">
        <label>
          Temperature
          <input
            type="number"
            min={0}
            max={2}
            step={0.1}
            value={settings.temperature}
            disabled={disabled}
            onChange={(event) => onChange({ temperature: Number(event.target.value) })}
          />
        </label>
        <label>
          Max output tokens
          <input
            type="number"
            min={1}
            step={1}
            value={settings.maxOutputTokens ?? ''}
            disabled={disabled}
            placeholder="Optional"
            onChange={(event) => {
              const raw = event.target.value.trim()
              onChange({ maxOutputTokens: raw.length > 0 ? Number(raw) : undefined })
            }}
          />
        </label>
      </div>
    </div>
  )
}

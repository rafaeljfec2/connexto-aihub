import type { PlaygroundSettings } from '../types'

type PlaygroundProviderModelSelectProps = {
  settings: PlaygroundSettings
  providers: string[]
  modelsForProvider: string[]
  disabled: boolean
  onChange: (patch: Partial<PlaygroundSettings>) => void
}

export function PlaygroundProviderModelSelect({
  settings,
  providers,
  modelsForProvider,
  disabled,
  onChange,
}: PlaygroundProviderModelSelectProps) {
  return (
    <div className="playground-select-row">
      <label>
        Provider
        <select
          value={settings.providerId}
          disabled={disabled || providers.length === 0}
          onChange={(event) => {
            const providerId = event.target.value
            onChange({ providerId, modelId: '' })
          }}
        >
          <option value="">Select provider</option>
          {providers.map((provider) => (
            <option key={provider} value={provider}>
              {provider}
            </option>
          ))}
        </select>
      </label>
      <label>
        Model
        <select
          value={settings.modelId}
          disabled={disabled || !settings.providerId || modelsForProvider.length === 0}
          onChange={(event) => onChange({ modelId: event.target.value })}
        >
          <option value="">Select model</option>
          {modelsForProvider.map((modelId) => (
            <option key={modelId} value={modelId}>
              {modelId}
            </option>
          ))}
        </select>
      </label>
    </div>
  )
}

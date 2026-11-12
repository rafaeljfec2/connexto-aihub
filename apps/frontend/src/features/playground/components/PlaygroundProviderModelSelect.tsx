import { useTranslation } from 'react-i18next'
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
  const { t } = useTranslation('playground')

  return (
    <div className="playground-select-row">
      <label>
        {t('select.provider')}
        <select
          value={settings.providerId}
          disabled={disabled || providers.length === 0}
          onChange={(event) => {
            const providerId = event.target.value
            onChange({ providerId, modelId: '' })
          }}
        >
          <option value="">{t('select.selectProvider')}</option>
          {providers.map((provider) => (
            <option key={provider} value={provider}>
              {provider}
            </option>
          ))}
        </select>
      </label>
      <label>
        {t('select.model')}
        <select
          value={settings.modelId}
          disabled={disabled || !settings.providerId || modelsForProvider.length === 0}
          onChange={(event) => onChange({ modelId: event.target.value })}
        >
          <option value="">{t('select.selectModel')}</option>
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

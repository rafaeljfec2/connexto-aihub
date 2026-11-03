import { useEffect, useMemo } from 'react'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { PlaygroundComposer } from '../components/PlaygroundComposer'
import { PlaygroundMessageList } from '../components/PlaygroundMessageList'
import { PlaygroundProviderModelSelect } from '../components/PlaygroundProviderModelSelect'
import { PlaygroundSettingsPanel } from '../components/PlaygroundSettingsPanel'
import { PlaygroundStatusBar } from '../components/PlaygroundStatusBar'
import { usePlayground } from '../hooks/usePlayground'
import {
  defaultSelection,
  groupCatalogModels,
  usePlaygroundCatalogModels,
} from '../hooks/usePlaygroundCatalog'

export function PlaygroundPage() {
  const catalog = usePlaygroundCatalogModels()
  const { state, dispatch, sendMessage, cancel, isStreaming } = usePlayground()

  const grouped = useMemo(
    () => groupCatalogModels(catalog.data?.data ?? []),
    [catalog.data?.data],
  )
  const providers = useMemo(() => [...grouped.keys()].sort(), [grouped])
  const modelsForProvider = useMemo(() => {
    if (!state.settings.providerId) {
      return []
    }
    return (grouped.get(state.settings.providerId) ?? []).map((model) => model.model_id)
  }, [grouped, state.settings.providerId])

  useEffect(() => {
    if (state.settings.providerId || providers.length === 0) {
      return
    }
    const defaults = defaultSelection(grouped)
    dispatch({ type: 'patch_settings', patch: defaults })
  }, [dispatch, grouped, providers.length, state.settings.providerId])

  useEffect(() => {
    if (!state.settings.providerId || !state.settings.modelId) {
      return
    }
    if (modelsForProvider.includes(state.settings.modelId)) {
      return
    }
    dispatch({
      type: 'patch_settings',
      patch: { modelId: modelsForProvider[0] ?? '' },
    })
  }, [dispatch, modelsForProvider, state.settings.modelId, state.settings.providerId])

  return (
    <div className="page playground-page">
      <AppPageHeader actions={<PlaygroundStatusBar phase={state.phase} />} />

      {catalog.isError ? (
        <div className="banner banner-error" role="alert">
          Catalog models unavailable. Open Catalog or sync providers before using the playground.
        </div>
      ) : null}

      {state.errorMessage ? (
        <div className="banner banner-error" role="alert">
          {state.errorMessage}
        </div>
      ) : null}

      <section className="playground-panel">
        <PlaygroundProviderModelSelect
          settings={state.settings}
          providers={providers}
          modelsForProvider={modelsForProvider}
          disabled={isStreaming}
          onChange={(patch) => dispatch({ type: 'patch_settings', patch })}
        />
        <PlaygroundSettingsPanel
          settings={state.settings}
          disabled={isStreaming}
          onChange={(patch) => dispatch({ type: 'patch_settings', patch })}
        />
        <button
          type="button"
          className="ghost-button"
          disabled={isStreaming}
          onClick={() => dispatch({ type: 'clear_conversation' })}
        >
          Clear conversation
        </button>
      </section>

      <PlaygroundMessageList messages={state.messages} />

      <PlaygroundComposer
        draft={state.draft}
        disabled={catalog.isLoading || catalog.isError}
        isStreaming={isStreaming}
        onDraftChange={(draft) => dispatch({ type: 'set_draft', draft })}
        onSend={() => void sendMessage()}
        onStop={cancel}
      />
    </div>
  )
}

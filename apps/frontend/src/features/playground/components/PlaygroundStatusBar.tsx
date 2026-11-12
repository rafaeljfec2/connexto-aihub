import { useTranslation } from 'react-i18next'
import type { PlaygroundRunPhase } from '../types'

type PlaygroundStatusBarProps = {
  phase: PlaygroundRunPhase
}

export function PlaygroundStatusBar({ phase }: PlaygroundStatusBarProps) {
  const { t } = useTranslation('playground')
  const labelKey = `status.${phase}` as const

  return (
    <div className="playground-status" role="status" aria-live="polite">
      {t(labelKey)}
    </div>
  )
}

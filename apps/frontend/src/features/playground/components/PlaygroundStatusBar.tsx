import type { PlaygroundRunPhase } from '../types'

type PlaygroundStatusBarProps = {
  phase: PlaygroundRunPhase
}

const LABELS: Record<PlaygroundRunPhase, string> = {
  idle: 'Ready',
  submitting: 'Sending request…',
  streaming: 'Streaming response…',
  completed: 'Completed',
  error: 'Error',
  cancelled: 'Cancelled',
}

export function PlaygroundStatusBar({ phase }: PlaygroundStatusBarProps) {
  return (
    <div className="playground-status" role="status" aria-live="polite">
      {LABELS[phase]}
    </div>
  )
}

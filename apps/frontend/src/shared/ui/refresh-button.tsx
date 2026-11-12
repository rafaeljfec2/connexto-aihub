import { useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { cn } from './cn'

export type RefreshButtonProps = {
  onRefresh: () => Promise<unknown> | void
  isRefreshing?: boolean
  label?: string
  iconOnly?: boolean
  iconClassName?: string
  className?: string
  disabled?: boolean
  minSpinMs?: number
  'aria-label'?: string
  title?: string
  'data-testid'?: string
}

/**
 * Shared refresh trigger button that keeps the `RefreshCw` icon spinning
 * during refresh (with a brief minimum spin duration so fast local requests
 * still give visible feedback).
 */
export function RefreshButton({
  onRefresh,
  isRefreshing = false,
  label = 'Atualizar',
  iconOnly = false,
  iconClassName = 'size-3.5',
  className,
  disabled = false,
  minSpinMs = 450,
  'aria-label': ariaLabel,
  title,
  'data-testid': dataTestId,
}: RefreshButtonProps) {
  const [isPending, setIsPending] = useState(false)
  const [isMinSpinActive, setIsMinSpinActive] = useState(false)

  useEffect(() => {
    if (!isMinSpinActive) {
      return
    }
    const timer = window.setTimeout(() => setIsMinSpinActive(false), minSpinMs)
    return () => window.clearTimeout(timer)
  }, [isMinSpinActive, minSpinMs])

  const isBusy = isRefreshing || isPending || isMinSpinActive

  async function handleClick(): Promise<void> {
    if (isRefreshing || isPending || disabled) {
      return
    }
    setIsPending(true)
    setIsMinSpinActive(true)
    try {
      await onRefresh()
    } catch {
      // Caller queries surface their own error state; keep button recoverable.
    } finally {
      setIsPending(false)
    }
  }

  return (
    <button
      type="button"
      className={cn(
        'inline-flex items-center justify-center gap-1.5 cursor-pointer disabled:cursor-not-allowed disabled:opacity-60',
        className,
      )}
      onClick={() => void handleClick()}
      disabled={disabled || isBusy}
      aria-label={ariaLabel}
      aria-busy={isBusy || undefined}
      title={title ?? label}
      data-testid={dataTestId}
    >
      <RefreshCw
        className={cn(iconClassName, 'shrink-0', isBusy && 'animate-spin')}
        aria-hidden
      />
      {!iconOnly ? <span>{label}</span> : null}
    </button>
  )
}

import { cn } from '../ui/cn'
import { resolveAiIconUrl, type AiIconVariant } from './resolveAiIconUrl'

export type AiBrandIconProps = {
  slug: string
  label: string
  size?: number
  className?: string
  variant?: AiIconVariant
  /** When no SVG resolves: render two-letter initials (`initials`) or nothing (`none`). */
  fallback?: 'initials' | 'none'
}

function initialsFromLabel(label: string): string {
  const trimmed = label.trim()
  if (trimmed.length >= 2) {
    return trimmed.slice(0, 2).toUpperCase()
  }
  if (trimmed.length === 1) {
    return trimmed.toUpperCase()
  }
  return '?'
}

export function AiBrandIcon({
  slug,
  label,
  size = 28,
  className,
  variant = 'auto',
  fallback = 'initials',
}: AiBrandIconProps) {
  const url = resolveAiIconUrl(slug, { variant })
  const style = { width: size, height: size }

  if (url) {
    return (
      <span
        className={cn('ai-brand-icon inline-grid place-items-center overflow-hidden', className)}
        style={style}
        aria-hidden
      >
        <img
          src={url}
          alt=""
          width={size}
          height={size}
          className="size-full object-contain"
          draggable={false}
          data-ai-icon-slug={slug}
        />
      </span>
    )
  }

  if (fallback === 'none') {
    return null
  }

  return (
    <span
      className={cn(
        'ai-brand-icon ai-brand-icon--fallback inline-grid place-items-center font-mono font-semibold uppercase',
        className,
      )}
      style={{ ...style, fontSize: Math.max(9, Math.round(size * 0.36)) }}
      aria-hidden
      data-ai-icon-slug={slug}
      data-ai-icon-fallback="initials"
    >
      {initialsFromLabel(label)}
    </span>
  )
}

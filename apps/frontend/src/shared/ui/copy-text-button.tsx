import { useEffect, useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { copyTextToClipboard } from '../lib/copyText'
import { cn } from './cn'

export type CopyTextButtonProps = {
  text: string
  label?: string
  className?: string
  /** Stop click from bubbling (e.g. table row selection). Default true. */
  stopPropagation?: boolean
}

export function CopyTextButton({
  text,
  label = 'Copiar identificador',
  className,
  stopPropagation = true,
}: CopyTextButtonProps) {
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!copied) {
      return
    }
    const timer = window.setTimeout(() => setCopied(false), 1600)
    return () => window.clearTimeout(timer)
  }, [copied])

  return (
    <button
      type="button"
      className={cn('copy-text-button', copied && 'is-copied', className)}
      aria-label={copied ? 'Copiado' : label}
      title={copied ? 'Copiado' : label}
      onClick={(event) => {
        if (stopPropagation) {
          event.stopPropagation()
          event.preventDefault()
        }
        void copyTextToClipboard(text).then((ok) => {
          if (ok) {
            setCopied(true)
          }
        })
      }}
    >
      {copied ? <Check className="size-3" aria-hidden /> : <Copy className="size-3" aria-hidden />}
    </button>
  )
}

import * as DialogPrimitive from '@radix-ui/react-dialog'
import type { ReactNode } from 'react'
import { cn } from './cn'

type DialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  children: ReactNode
  /** Merged into the dialog panel (e.g. max-height, padding). */
  contentClassName?: string
  /** Wrapper around children (default spacing below header). */
  bodyClassName?: string
  /** When false, the body does not scroll (content must fit or handle overflow inside). */
  bodyScroll?: boolean
}

export function Dialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  contentClassName,
  bodyClassName,
  bodyScroll = true,
}: DialogProps) {
  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/70" />
        <DialogPrimitive.Content
          className={cn(
            // Top-anchored so tall content is not clipped by vertical centering on short viewports
            'fixed left-1/2 top-[max(0.5rem,min(5vh,2rem))] z-50 flex max-h-[calc(100dvh-1rem)] w-[min(100%,32rem)] -translate-x-1/2 flex-col overflow-hidden rounded-xl border border-line bg-[#171E29] shadow-2xl outline-none sm:max-h-[min(92vh,42rem)]',
            contentClassName,
          )}
        >
          <div className="flex shrink-0 items-start justify-between gap-2 border-b border-line/50 px-4 pb-2 pt-4">
            <div className="min-w-0">
              <DialogPrimitive.Title className="text-base font-semibold text-primary">
                {title}
              </DialogPrimitive.Title>
              {description ? (
                <DialogPrimitive.Description className="mt-0.5 break-words text-xs text-muted sm:text-sm">
                  {description}
                </DialogPrimitive.Description>
              ) : null}
            </div>
            <DialogPrimitive.Close
              className="inline-flex size-7 shrink-0 items-center justify-center rounded-md border border-line text-secondary hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cobalt/40"
              aria-label="Close dialog"
            >
              ✕
            </DialogPrimitive.Close>
          </div>
          <div
            className={cn(
              'min-h-0 flex-1 overflow-x-hidden px-4 pb-4 pt-3',
              bodyScroll ? 'overflow-y-auto' : 'overflow-y-hidden',
              bodyClassName,
            )}
          >
            {children}
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}

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
}

export function Dialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  contentClassName,
  bodyClassName,
}: DialogProps) {
  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/70" />
        <DialogPrimitive.Content
          className={cn(
            'fixed left-1/2 top-1/2 z-50 w-[min(100%,32rem)] max-h-[min(90vh,40rem)] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-[#171E29] p-5 shadow-2xl outline-none',
            contentClassName,
          )}
        >
          <div className="flex items-start justify-between gap-3">
            <div>
              <DialogPrimitive.Title className="text-base font-semibold text-primary">
                {title}
              </DialogPrimitive.Title>
              {description ? (
                <DialogPrimitive.Description className="mt-1 text-sm text-muted">
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
          <div className={cn('mt-4', bodyClassName)}>{children}</div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}

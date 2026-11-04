import { useEffect, useRef } from 'react'
import { FocusScope } from '@radix-ui/react-focus-scope'
import type { OperationalConnectionState } from '../../shared/realtime/useOperationalEvents'
import { cn } from '../../shared/ui/cn'
import { AppSidebarNavigation } from './AppSidebarNavigation'
import { AppSidebarStatus } from './AppSidebarStatus'

type AppSidebarProps = {
  readonly collapsed: boolean
  readonly mobileOpen: boolean
  readonly connectionState: OperationalConnectionState
  readonly onToggleCollapsed: () => void
  readonly onCloseMobile: () => void
}

export function AppSidebar({
  collapsed,
  mobileOpen,
  connectionState,
  onToggleCollapsed: _onToggleCollapsed,
  onCloseMobile,
}: AppSidebarProps) {
  const sidebarRef = useRef<HTMLElement>(null)
  const restoreFocusRef = useRef<HTMLElement | null>(null)

  useEffect(() => {
    if (!mobileOpen) {
      return
    }

    restoreFocusRef.current =
      document.activeElement instanceof HTMLElement ? document.activeElement : null

    const focusFirst = (): void => {
      const root = sidebarRef.current
      if (!root) {
        return
      }
      root.querySelector<HTMLElement>(
        'a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"])',
      )?.focus()
    }

    const frame = requestAnimationFrame(focusFirst)
    const onKeyDown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape') {
        event.preventDefault()
        onCloseMobile()
      }
    }

    document.addEventListener('keydown', onKeyDown)
    return () => {
      cancelAnimationFrame(frame)
      document.removeEventListener('keydown', onKeyDown)
      restoreFocusRef.current?.focus()
      restoreFocusRef.current = null
    }
  }, [mobileOpen, onCloseMobile])

  return (
    <>
      {mobileOpen ? (
        <button
          type="button"
          className="fixed inset-0 z-40 bg-black/60 lg:hidden"
          aria-label="Close navigation"
          onClick={onCloseMobile}
        />
      ) : null}
      <FocusScope
        trapped={mobileOpen}
        loop
        asChild
        onMountAutoFocus={(event) => event.preventDefault()}
      >
        <aside
          ref={sidebarRef}
          id="app-sidebar"
          aria-modal={mobileOpen ? true : undefined}
          className={cn(
            'z-50 flex h-full flex-col overflow-y-auto scrollbar-slim border-r border-line bg-sidebar px-3.5 pb-4 pt-[22px] transition-[width,transform] duration-200',
            collapsed ? 'w-[72px]' : 'w-[224px]',
            'fixed inset-y-0 left-0 lg:static lg:translate-x-0',
            mobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
          )}
        >
          <div className={cn('mb-6 px-2.5', collapsed && 'px-0 text-center')}>
            {!collapsed ? (
              <div>
                <div className="text-[15px] font-semibold tracking-[-0.03em]">connexto</div>
                <div className="mt-0.5 font-mono text-[9px] uppercase tracking-[0.13em] text-muted">
                  AI HUB · CONTROL PLANE
                </div>
              </div>
            ) : null}
          </div>

          {!collapsed ? (
            <div className="mx-1 mb-6 flex items-center gap-2.5 rounded-[10px] border border-line bg-white/[0.025] px-3 py-2.5">
              <span className="size-[7px] shrink-0 rounded-full bg-cobalt shadow-[0_0_0_4px_rgba(120,150,232,0.12)]" />
              <div>
                <strong className="block text-[11px] font-semibold">Workspace principal</strong>
                <small className="mt-0.5 block text-[10px] text-muted">Instância de desenvolvimento</small>
              </div>
            </div>
          ) : null}

          <AppSidebarNavigation collapsed={collapsed} onNavigate={onCloseMobile} />
          <AppSidebarStatus collapsed={collapsed} connectionState={connectionState} />
        </aside>
      </FocusScope>
    </>
  )
}

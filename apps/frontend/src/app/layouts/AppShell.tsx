import { useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import type { OperationalConnectionState } from '../../shared/realtime/useOperationalEvents'
import { AppSidebar } from './AppSidebar'
import { AppTopBar } from './AppTopBar'
import { resolvePageMeta } from './pageMeta'

const SIDEBAR_COLLAPSED_KEY = 'connexto:sidebar-collapsed'

function getInitialSidebarCollapsed(): boolean {
  if (typeof window === 'undefined') {
    return true
  }
  try {
    const saved = window.localStorage.getItem(SIDEBAR_COLLAPSED_KEY)
    if (saved !== null) {
      return saved === 'true'
    }
  } catch {
    // Ignore storage read errors in restricted contexts
  }
  // Default to slim (collapsed = true) per user requirement
  return true
}

type AppShellProps = {
  connectionState: OperationalConnectionState
  onReconnect: () => void
}

export function AppShell({ connectionState, onReconnect }: AppShellProps) {
  const location = useLocation()
  const meta = resolvePageMeta(location.pathname)
  const [collapsed, setCollapsed] = useState<boolean>(getInitialSidebarCollapsed)
  const [mobileOpen, setMobileOpen] = useState(false)

  useEffect(() => {
    setMobileOpen(false)
  }, [location.pathname])

  const handleToggleCollapsed = () => {
    setCollapsed((previous) => {
      const next = !previous
      try {
        window.localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(next))
      } catch {
        // Ignore storage write errors
      }
      return next
    })
  }

  return (
    <div className="grid h-full w-full overflow-hidden bg-graphite lg:grid-cols-[auto_1fr]">
      <AppSidebar
        collapsed={collapsed}
        mobileOpen={mobileOpen}
        connectionState={connectionState}
        onToggleCollapsed={handleToggleCollapsed}
        onCloseMobile={() => setMobileOpen(false)}
      />
      <div
        className="flex h-full min-w-0 flex-col overflow-hidden"
        aria-hidden={mobileOpen ? true : undefined}
      >
        <AppTopBar
          crumb={meta.crumb}
          connectionState={connectionState}
          sidebarCollapsed={collapsed}
          onReconnect={onReconnect}
          onOpenMobileNav={() => setMobileOpen(true)}
          onToggleSidebar={handleToggleCollapsed}
        />
        <main className="min-h-0 flex-1 overflow-y-auto px-4 py-6 lg:px-8 lg:py-7">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

import { useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import type { OperationalConnectionState } from '../../shared/realtime/useOperationalEvents'
import { AppSidebar } from './AppSidebar'
import { AppTopBar } from './AppTopBar'
import { resolvePageMeta } from './pageMeta'

type AppShellProps = {
  connectionState: OperationalConnectionState
  onReconnect: () => void
}

export function AppShell({ connectionState, onReconnect }: AppShellProps) {
  const location = useLocation()
  const meta = resolvePageMeta(location.pathname)
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)

  useEffect(() => {
    setMobileOpen(false)
  }, [location.pathname])

  return (
    <div className="grid min-h-svh bg-graphite lg:grid-cols-[auto_1fr]">
      <AppSidebar
        collapsed={collapsed}
        mobileOpen={mobileOpen}
        connectionState={connectionState}
        onToggleCollapsed={() => setCollapsed((value) => !value)}
        onCloseMobile={() => setMobileOpen(false)}
      />
      <div
        className="flex min-w-0 flex-col lg:min-h-svh"
        aria-hidden={mobileOpen ? true : undefined}
      >
        <AppTopBar
          crumb={meta.crumb}
          connectionState={connectionState}
          onReconnect={onReconnect}
          onOpenMobileNav={() => setMobileOpen(true)}
        />
        <main className="min-h-0 flex-1 overflow-auto px-4 py-6 lg:px-8 lg:py-7">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

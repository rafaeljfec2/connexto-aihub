import { Activity, Radio } from 'lucide-react'
import { Link, Outlet } from 'react-router-dom'
import type { OperationalConnectionState } from '../hooks/useOperationalEvents'

type AppShellProps = {
  connectionState: OperationalConnectionState
  onReconnect: () => void
}

export function AppShell({ connectionState, onReconnect }: AppShellProps) {
  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="brand">
          <Activity size={18} />
          <span>Connexto AI Hub</span>
        </div>
        <div className="header-actions">
          <span className={`sse-badge sse-${connectionState}`}>
            <Radio size={14} />
            {connectionState === 'live' ? 'Live SSE' : connectionState === 'connecting' ? 'Connecting…' : 'SSE degraded'}
          </span>
          <button type="button" className="ghost-button" onClick={onReconnect}>
            Reconnect SSE
          </button>
        </div>
      </header>
      <div className="app-body">
        <nav className="sidebar">
          <Link to="/">Overview</Link>
        </nav>
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

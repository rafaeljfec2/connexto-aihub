import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Bell, ChevronDown, LogOut, Menu, PanelLeftClose, PanelLeftOpen, Radio } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { logoutOperator, operatorSessionQueryKey } from '../../features/auth/api/operatorSession'
import { useOperatorSession } from '../../features/auth/hooks/useOperatorSession'
import type { OperationalConnectionState } from '../../shared/realtime/useOperationalEvents'
import { Badge } from '../../shared/ui/badge'
import { Button } from '../../shared/ui/button'
import { cn } from '../../shared/ui/cn'

type AppTopBarProps = {
  crumb: string
  connectionState: OperationalConnectionState
  sidebarCollapsed?: boolean
  onReconnect: () => void
  onOpenMobileNav: () => void
  onToggleSidebar?: () => void
}

function connectionLabel(state: OperationalConnectionState): string {
  if (state === 'live') {
    return 'Live SSE'
  }
  if (state === 'connecting') {
    return 'Connecting…'
  }
  return 'SSE degraded'
}

export function AppTopBar({
  crumb,
  connectionState,
  sidebarCollapsed,
  onReconnect,
  onOpenMobileNav,
  onToggleSidebar,
}: AppTopBarProps) {
  const live = connectionState === 'live'
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const session = useOperatorSession()

  const logoutMutation = useMutation({
    mutationFn: logoutOperator,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: operatorSessionQueryKey })
      if (session.data?.mode === 'passphrase') {
        navigate('/login', { replace: true })
      }
    },
  })

  const showLogout =
    session.data?.authenticated === true &&
    (session.data.mode === 'passphrase' || session.data.mode === 'loopback_trust')

  return (
    <header className="flex h-[62px] shrink-0 items-center justify-between border-b border-line bg-surface/80 px-4 backdrop-blur-sm lg:px-8">
      <div className="flex min-w-0 items-center gap-3">
        <Button
          variant="icon"
          size="sm"
          className="lg:hidden"
          onClick={onOpenMobileNav}
          aria-label="Open navigation"
        >
          <Menu className="size-4" />
        </Button>
        {onToggleSidebar ? (
          <Button
            variant="icon"
            size="sm"
            className="hidden text-[#94A1B3] hover:text-[#E0E7F1] lg:inline-flex"
            onClick={onToggleSidebar}
            aria-label={sidebarCollapsed ? 'Expand navigation sidebar' : 'Collapse navigation sidebar'}
            title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {sidebarCollapsed ? (
              <PanelLeftOpen className="size-4" />
            ) : (
              <PanelLeftClose className="size-4" />
            )}
          </Button>
        ) : null}
        <p className="truncate text-[11px] text-[#94A1B3]">
          Workspace <span className="mx-2 text-[#58677B]">/</span>{' '}
          <b className="font-medium text-[#E0E7F1]">{crumb}</b>
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-3 lg:gap-4">
        <div className="hidden items-center gap-2 rounded-[7px] border border-line bg-white/[0.025] px-2.5 py-1.5 text-[10px] text-[#BEC8D6] sm:flex">
          <span className={cn('size-1.5 rounded-full', live ? 'bg-healthy' : 'bg-warning')} />
          Development
          <ChevronDown className="size-3 opacity-60" aria-hidden />
        </div>
        <span
          className={cn(
            'inline-flex items-center gap-1.5 font-mono text-[10px]',
            live ? 'text-healthy' : connectionState === 'connecting' ? 'text-warning' : 'text-failure',
          )}
        >
          <Radio className="size-3.5" aria-hidden />
          {connectionLabel(connectionState)}
        </span>
        {connectionState !== 'live' ? (
          <Button variant="secondary" size="sm" onClick={onReconnect}>
            Reconnect
          </Button>
        ) : null}
        <Badge variant="mono" className="hidden md:inline-flex">
          Backend data
        </Badge>
        {showLogout ? (
          <Button
            variant="ghost"
            size="sm"
            disabled={logoutMutation.isPending}
            onClick={() => logoutMutation.mutate()}
          >
            <LogOut className="size-3.5" aria-hidden />
            Sign out
          </Button>
        ) : null}
        <Button variant="icon" aria-label="Notifications">
          <Bell className="size-3.5 opacity-70" />
        </Button>
      </div>
    </header>
  )
}

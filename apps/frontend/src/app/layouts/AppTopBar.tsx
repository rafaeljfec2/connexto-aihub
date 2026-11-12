import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Bell, ChevronDown, LogOut, Menu, PanelLeftClose, PanelLeftOpen, Radio } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { logoutOperator, operatorSessionQueryKey } from '../../features/auth/api/operatorSession'
import { useOperatorSession } from '../../features/auth/hooks/useOperatorSession'
import { getRuntimeMeta, overviewQueryKeys } from '../../features/overview/api/queries'
import type { OperationalConnectionState } from '../../shared/realtime/useOperationalEvents'
import { Button } from '../../shared/ui/button'
import { cn } from '../../shared/ui/cn'
import { APP_SHELL_TOP_BAR_CLASS } from './appShellLayout'

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

function formatEnvironmentName(env?: string): string {
  if (env === 'production') return 'Produção'
  if (env === 'staging' || env === 'homologation') return 'Homologação'
  return 'Desenvolvimento'
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

  const runtimeMeta = useQuery({
    queryKey: overviewQueryKeys.runtimeMeta,
    queryFn: getRuntimeMeta,
    refetchInterval: false,
    retry: 1,
  })

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

  const environmentLabel = formatEnvironmentName(runtimeMeta.data?.environment)
  const isDemonstrative = runtimeMeta.data?.data_origin === 'demonstrative'
  const originLabel = isDemonstrative ? 'DADOS DE EXEMPLO' : 'DADOS OPERACIONAIS'

  return (
    <header className={APP_SHELL_TOP_BAR_CLASS}>
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
        <p className="truncate text-[0.6875rem] text-[#94A1B3]">
          Workspace <span className="mx-2 text-[#58677B]">/</span>{' '}
          <b className="font-medium text-[#E0E7F1]">{crumb}</b>
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-3 lg:gap-4">
        <div className="hidden items-center gap-2 rounded-[7px] border border-line bg-white/[0.025] px-2.5 py-1.5 text-[0.625rem] text-[#BEC8D6] sm:flex">
          <span className="size-1.5 rounded-full bg-cobalt" />
          {environmentLabel}
          <ChevronDown className="size-3 text-[#8592A7]" aria-hidden />
        </div>
        <span className="hidden font-mono text-[0.625rem] tracking-[0.05em] text-[#8A98AB] md:inline">
          {originLabel}
        </span>
        <span
          className={cn(
            'inline-flex items-center gap-1.5 font-mono text-[0.625rem]',
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
        <Button variant="icon" aria-label="Notificações">
          <Bell className="size-3.5 opacity-70" />
        </Button>
      </div>
    </header>
  )
}

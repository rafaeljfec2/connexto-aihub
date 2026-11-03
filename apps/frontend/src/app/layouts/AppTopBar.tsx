import { Bell, ChevronDown, Menu, Radio } from 'lucide-react'
import type { OperationalConnectionState } from '../../shared/realtime/useOperationalEvents'
import { Badge } from '../../shared/ui/badge'
import { Button } from '../../shared/ui/button'
import { cn } from '../../shared/ui/cn'

type AppTopBarProps = {
  crumb: string
  connectionState: OperationalConnectionState
  onReconnect: () => void
  onOpenMobileNav: () => void
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
  onReconnect,
  onOpenMobileNav,
}: AppTopBarProps) {
  const live = connectionState === 'live'

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
        <Button variant="icon" aria-label="Notifications">
          <Bell className="size-3.5 opacity-70" />
        </Button>
      </div>
    </header>
  )
}

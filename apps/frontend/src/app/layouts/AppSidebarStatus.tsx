import { useQuery } from '@tanstack/react-query'
import type { OperationalConnectionState } from '../../shared/realtime/useOperationalEvents'
import { cn } from '../../shared/ui/cn'
import { getRuntimeMeta, overviewQueryKeys } from '../../features/overview/api/queries'

type AppSidebarStatusProps = {
  readonly collapsed: boolean
  readonly connectionState: OperationalConnectionState
}

function statusCopy(state: OperationalConnectionState): {
  readonly label: string
  readonly detail: string
  readonly dotClass: string
} {
  if (state === 'live') {
    return { label: 'Gateway operacional', detail: 'CONEXÃO OPERACIONAL ATIVA', dotClass: 'bg-healthy' }
  }
  if (state === 'connecting') {
    return { label: 'Gateway conectando', detail: 'CONECTANDO EVENT STREAM', dotClass: 'bg-warning' }
  }
  return { label: 'Gateway degradado', detail: 'RECONEXÃO AGENDADA', dotClass: 'bg-failure' }
}

export function AppSidebarStatus({ collapsed, connectionState }: AppSidebarStatusProps) {
  const status = statusCopy(connectionState)
  const runtimeMeta = useQuery({
    queryKey: overviewQueryKeys.runtimeMeta,
    queryFn: getRuntimeMeta,
    refetchInterval: false,
    retry: 1,
  })
  const detail = runtimeMeta.data?.data_origin === 'demonstrative'
    ? 'AMBIENTE DEMONSTRATIVO'
    : status.detail

  return (
    <div className="pt-3">
      <div className={cn('flex items-center gap-2 border-t border-line pt-3.5', collapsed && 'justify-center')}>
        <span className={cn('size-2 shrink-0 rounded-full', status.dotClass)} />
        {!collapsed ? (
          <div>
            <span className="block text-[11px] text-[#BEC8D6]">{status.label}</span>
            <small className="mt-0.5 block font-mono text-[9px] uppercase text-[#8290A5]">{detail}</small>
          </div>
        ) : null}
      </div>
    </div>
  )
}

import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import type { OperationalConnectionState } from '../../shared/realtime/useOperationalEvents'
import { cn } from '../../shared/ui/cn'
import { getRuntimeMeta, overviewQueryKeys } from '../../features/overview/api/queries'

type AppSidebarStatusProps = {
  readonly collapsed: boolean
  readonly connectionState: OperationalConnectionState
}

export function AppSidebarStatus({ collapsed, connectionState }: AppSidebarStatusProps) {
  const { t } = useTranslation('common')
  const runtimeMeta = useQuery({
    queryKey: overviewQueryKeys.runtimeMeta,
    queryFn: getRuntimeMeta,
    refetchInterval: false,
    retry: 1,
  })

  const status =
    connectionState === 'live'
      ? {
          label: t('shell.gatewayOperational'),
          detail: t('shell.connectionLive'),
          dotClass: 'bg-healthy',
        }
      : connectionState === 'connecting'
        ? {
            label: t('shell.gatewayConnecting'),
            detail: t('shell.connectionConnecting'),
            dotClass: 'bg-warning',
          }
        : {
            label: t('shell.gatewayDegraded'),
            detail: t('shell.connectionReconnect'),
            dotClass: 'bg-failure',
          }

  const detail =
    runtimeMeta.data?.data_origin === 'demonstrative' ? t('shell.demonstrativeEnv') : status.detail

  return (
    <div className="pt-3">
      <div className={cn('flex items-center gap-2 border-t border-line pt-3.5', collapsed && 'justify-center')}>
        <span className={cn('size-2 shrink-0 rounded-full', status.dotClass)} />
        {!collapsed ? (
          <div>
            <span className="block text-[0.6875rem] text-[#BEC8D6]">{status.label}</span>
            <small className="mt-0.5 block font-mono text-[0.5625rem] uppercase text-[#8290A5]">{detail}</small>
          </div>
        ) : null}
      </div>
    </div>
  )
}

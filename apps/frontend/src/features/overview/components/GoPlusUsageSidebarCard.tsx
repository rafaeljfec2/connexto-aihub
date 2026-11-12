import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { cn } from '../../../shared/ui/cn'
import { formatNumber } from '../../../shared/i18n/format'
import { getProviderQuotas, overviewQueryKeys } from '../api/queries'
import {
  formatGoProductLabel,
  goPlusChipPercent,
  goPlusMetersFromSnapshot,
  selectOpenCodeGoSnapshot,
} from '../lib/goPlusMeters'

type GoPlusUsageSidebarCardProps = Readonly<{
  collapsed: boolean
}>

export function GoPlusUsageSidebarCard({ collapsed }: GoPlusUsageSidebarCardProps) {
  const { t } = useTranslation('overview')
  const quotas = useQuery({
    queryKey: overviewQueryKeys.providerQuotas,
    queryFn: getProviderQuotas,
    refetchInterval: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retry: 1,
  })

  const snapshot = selectOpenCodeGoSnapshot(quotas.data?.snapshots ?? [])
  const meters = snapshot ? goPlusMetersFromSnapshot(snapshot) : []
  const chipPercent = goPlusChipPercent(meters)
  const title = formatGoProductLabel(snapshot?.workspace_name)
  const percentLabel =
    chipPercent != null
      ? t('goPlus.percentUsedSidebar', { percent: formatNumber(chipPercent) })
      : undefined

  if (collapsed) {
    return (
      <div
        className="mt-3 flex justify-center border-t border-line pt-3"
        data-testid="go-plus-sidebar-collapsed"
        title={percentLabel != null ? `${title} ${percentLabel}` : t('goPlus.noReadingChip')}
      >
        {chipPercent != null ? (
          <span className="rounded-md border border-line bg-white/[0.04] px-1.5 py-1 font-mono text-[0.5625rem] text-[#C5D0E0]">
            {formatNumber(chipPercent)}%
          </span>
        ) : (
          <span className="font-mono text-[0.5625rem] text-muted">—</span>
        )}
      </div>
    )
  }

  return (
    <div className={cn('mt-3 border-t border-line pt-3')} data-testid="go-plus-sidebar">
      {chipPercent != null && percentLabel != null ? (
        <div
          className="rounded-lg border border-line/80 bg-white/[0.025] px-2.5 py-2"
          data-testid="go-plus-sidebar-summary"
        >
          <p className="text-[0.625rem] font-semibold text-[#E7ECF5]">{title}</p>
          <p className="mt-0.5 text-[0.5625rem] text-[#AEBBD0]">{percentLabel}</p>
        </div>
      ) : (
        <p className="px-0.5 text-[0.625rem] text-muted" role="status">
          {t('goPlus.noReadingChip')}
        </p>
      )}
    </div>
  )
}

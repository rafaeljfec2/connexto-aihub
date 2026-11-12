import { useTranslation } from 'react-i18next'
import { formatNumber } from '../../../shared/i18n/format'
import { formatQuotaFetchedAt } from '../../overview/lib/formatQuotaDate'
import type { ProviderQuotaSnapshot } from '../../overview/types/quota'
import {
  cursorMonthlyAdditionalUsage,
  cursorPrepaidBalanceUsd,
  cursorWeeklyByProduct,
  cursorWeeklyIncludedStatus,
  formatAccountUsd,
} from '../lib/cursorQuotaMetrics'

type CursorQuotaBreakdownPanelProps = Readonly<{
  snapshot: ProviderQuotaSnapshot | undefined
  testId?: string
}>

export function CursorQuotaBreakdownPanel({
  snapshot,
  testId = 'cursor-detail-breakdown',
}: CursorQuotaBreakdownPanelProps) {
  const { t } = useTranslation('providers')

  if (!snapshot || snapshot.provider_id !== 'cursor' || snapshot.status !== 'available') {
    return null
  }

  const byProduct = cursorWeeklyByProduct(snapshot)
  const monthly = cursorMonthlyAdditionalUsage(snapshot)
  const weeklyStatus = cursorWeeklyIncludedStatus(snapshot)
  const prepaidBalance = cursorPrepaidBalanceUsd(snapshot)

  if (byProduct.length === 0 && !monthly && !weeklyStatus && prepaidBalance === undefined) {
    return null
  }

  return (
    <div
      className="mt-2 rounded-lg border border-line/60 bg-[#171E29]/90 p-2 space-y-1.5"
      data-testid={testId}
    >
      <div className="flex items-center justify-between text-[0.5625rem] uppercase tracking-wider font-semibold text-[#8290A5]">
        <span>{t('cursorBreakdown.title')}</span>
        {weeklyStatus ? <span className="font-mono text-[0.5rem] text-muted">{weeklyStatus}</span> : null}
      </div>

      {byProduct.length > 0 ? (
        <div className="space-y-1">
          {byProduct.map((entry) => (
            <div
              key={entry.product}
              className="flex items-center justify-between text-[0.5625rem] rounded bg-[#0f1722]/50 p-1.5 border border-line/30"
            >
              <span className="text-[#AEBBD0]">
                {entry.label} {t('cursorBreakdown.weeklySuffix')}
              </span>
              <span className="font-semibold text-[#CBD5E2]">
                {entry.percent != null ? `${formatNumber(entry.percent)}% · ` : ''}
                {formatAccountUsd(entry.usedUsd)}
              </span>
            </div>
          ))}
        </div>
      ) : null}

      {monthly ? (
        <div className="space-y-1 rounded bg-[#0f1722]/50 p-1.5 border border-line/30 text-[0.5625rem]">
          <div className="flex items-center justify-between">
            <span className="text-[#AEBBD0]">{t('cursorBreakdown.monthlyAdditional')}</span>
            <span className="font-semibold text-[#CBD5E2]">
              {monthly.usedUsd != null ? formatAccountUsd(monthly.usedUsd) : '—'}
            </span>
          </div>
          <div className="flex items-center justify-between text-[#8290A5]">
            <span>{t('cursorBreakdown.monthlyLimit')}</span>
            <span>
              {monthly.limitUsd != null ? formatAccountUsd(monthly.limitUsd) : t('cursorBreakdown.noMonthlyLimit')}
            </span>
          </div>
          {monthly.resetsAt ? (
            <div className="flex items-center justify-between text-[#8290A5]">
              <span>{t('cursorBreakdown.monthlyReset')}</span>
              <span>{formatQuotaFetchedAt(monthly.resetsAt)}</span>
            </div>
          ) : null}
        </div>
      ) : null}

      {prepaidBalance !== undefined ? (
        <div className="flex items-center justify-between text-[0.5625rem] rounded bg-[#0f1722]/50 p-1.5 border border-line/30">
          <span className="text-[#AEBBD0]">{t('cursorBreakdown.prepaidBalance')}</span>
          <span className="font-semibold text-[#CBD5E2]">{formatAccountUsd(prepaidBalance)}</span>
        </div>
      ) : null}
    </div>
  )
}

import { useTranslation } from 'react-i18next'
import i18n from '../../../shared/i18n'
import { formatResetsIn } from '../lib/formatResetsIn'
import { formatQuotaValue } from '../lib/formatQuotaValue'
import type { ProviderQuotaMetric } from '../types/quota'

type ConsoleBudgetsPanelProps = Readonly<{
  budgets: readonly ProviderQuotaMetric[]
  now?: Date
  testIdPrefix?: string
}>

function budgetTitle(metric: ProviderQuotaMetric): string {
  const period = metric.period?.trim()
  if (period) {
    return i18n.t('overview:consoleBudgets.budgetWithPeriod', { period })
  }
  if (metric.scope?.trim()) {
    return i18n.t('overview:consoleBudgets.budgetWithScope', { scope: metric.scope })
  }
  return metric.label || i18n.t('overview:consoleBudgets.budgetDefault')
}

export function ConsoleBudgetsPanel({
  budgets,
  now,
  testIdPrefix = 'console-budget',
}: ConsoleBudgetsPanelProps) {
  const { t } = useTranslation('overview')

  if (budgets.length === 0) return null

  return (
    <section
      className="mt-3 space-y-2 rounded-lg border border-line/80 bg-[#182333]/70 p-2.5"
      data-testid={`${testIdPrefix}-panel`}
      aria-label={t('consoleBudgets.title')}
    >
      <h4 className="text-[0.625rem] font-semibold text-[#E7ECF5]">{t('consoleBudgets.title')}</h4>
      <ul className="space-y-2">
        {budgets.map((budget) => {
          const used = budget.used != null ? formatQuotaValue(budget.used, budget.unit) : '—'
          const limit = budget.limit != null ? formatQuotaValue(budget.limit, budget.unit) : '—'
          const resets = budget.resets_at ? formatResetsIn(budget.resets_at, now) : undefined
          return (
            <li
              key={budget.metric}
              className="text-[0.5625rem]"
              data-testid={`${testIdPrefix}-${budget.metric}`}
            >
              <div className="mb-0.5 flex items-baseline justify-between gap-2">
                <span className="min-w-0 truncate font-medium text-[#CBD5E2]">{budgetTitle(budget)}</span>
                {resets ? <span className="shrink-0 text-[#8290A5]">{resets}</span> : null}
              </div>
              <p className="m-0 text-[#A7B3C3]">{t('consoleBudgets.usedLimit', { used, limit })}</p>
            </li>
          )
        })}
      </ul>
    </section>
  )
}

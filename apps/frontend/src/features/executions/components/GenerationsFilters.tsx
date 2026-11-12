import { Filter, Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { FilterChips, Input } from '../../../shared/ui'
import { catalogProviderLabel } from '../../catalog/lib/catalogProviderLabel'
import type { GenerationPeriodPreset } from '../lib/generationPeriodPresets'
import { EXECUTION_OUTCOMES } from '../types'
import type { ExecutionOutcome } from '../types'

export type GenerationFilterDraft = {
  readonly provider_id: string
  readonly model_id: string
  readonly outcome: ExecutionOutcome | 'all'
  readonly start_date: string
  readonly end_date: string
}

type GenerationsFiltersProps = {
  readonly search: string
  readonly onSearchChange: (value: string) => void
  readonly value: GenerationFilterDraft
  readonly periodPreset: GenerationPeriodPreset
  readonly onPeriodPresetChange: (preset: GenerationPeriodPreset) => void
  readonly providers: readonly string[]
  readonly onDraftChange: (value: GenerationFilterDraft) => void
  readonly onOutcomeChange: (outcome: GenerationFilterDraft['outcome']) => void
  readonly onProviderChange: (providerId: string) => void
  readonly onApply: () => void
}

const PERIOD_OPTIONS: Array<{ id: GenerationPeriodPreset; labelKey: string }> = [
  { id: '30d', labelKey: 'filters.period30d' },
  { id: '7d', labelKey: 'filters.period7d' },
  { id: 'all', labelKey: 'filters.periodAll' },
  { id: 'custom', labelKey: 'filters.periodCustom' },
]

export function GenerationsFilters({
  search,
  onSearchChange,
  value,
  periodPreset,
  onPeriodPresetChange,
  providers,
  onDraftChange,
  onOutcomeChange,
  onProviderChange,
  onApply,
}: GenerationsFiltersProps) {
  const { t } = useTranslation('executions')
  const showCustomDates = periodPreset === 'custom'

  const outcomes = [
    { id: 'all' as const, label: t('filters.outcomes.all') },
    ...EXECUTION_OUTCOMES.map((outcome) => ({
      id: outcome,
      label: t(`filters.outcomes.${outcome}`),
    })),
  ]

  return (
    <section className="generations-filter-bar" aria-label={t('filters.ariaLabel')}>
      <div className="generations-filter-primary">
        <label className="generations-search">
          <Search aria-hidden="true" className="size-3.5 shrink-0 text-muted" />
          <Input
            type="search"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder={t('filters.searchPlaceholder')}
            aria-label={t('filters.searchAriaLabel')}
            className="border-0 bg-transparent p-0 shadow-none focus-visible:ring-0"
          />
        </label>

        <FilterChips
          className="generations-outcome-chips"
          options={[...outcomes]}
          selectedId={value.outcome}
          onSelect={(outcome) => onOutcomeChange(outcome as GenerationFilterDraft['outcome'])}
          aria-label={t('filters.outcomeAriaLabel')}
        />

        <select
          value={value.provider_id}
          onChange={(event) => onProviderChange(event.target.value)}
          className="generations-compact-select"
          aria-label={t('filters.providerAriaLabel')}
        >
          <option value="">{t('filters.allProviders')}</option>
          {providers.map((provider) => (
            <option key={provider} value={provider}>
              {catalogProviderLabel(provider)}
            </option>
          ))}
        </select>

        <select
          value={periodPreset}
          onChange={(event) => onPeriodPresetChange(event.target.value as GenerationPeriodPreset)}
          className="generations-compact-select generations-period-select"
          aria-label={t('filters.periodAriaLabel')}
        >
          {PERIOD_OPTIONS.map((option) => (
            <option key={option.id} value={option.id}>
              {t(option.labelKey)}
            </option>
          ))}
        </select>

        <button
          type="button"
          className="generations-filter-apply"
          onClick={onApply}
          aria-label={t('filters.applyAriaLabel')}
        >
          <Filter className="size-3.5" aria-hidden="true" />
        </button>
      </div>

      {showCustomDates ? (
        <div className="generations-filter-advanced">
          <label className="generations-filter-field">
            <span>{t('filters.modelLabel')}</span>
            <Input
              value={value.model_id}
              onChange={(event) => onDraftChange({ ...value, model_id: event.target.value })}
              onKeyDown={(event) => {
                if (event.key !== 'Enter') return
                event.preventDefault()
                onApply()
              }}
              placeholder={t('filters.modelPlaceholder')}
              aria-label={t('filters.modelAriaLabel')}
            />
          </label>
          <label className="generations-filter-field">
            <span>{t('filters.fromLabel')}</span>
            <Input
              type="date"
              value={value.start_date}
              onChange={(event) => onDraftChange({ ...value, start_date: event.target.value })}
              aria-label={t('filters.startDateAriaLabel')}
            />
          </label>
          <label className="generations-filter-field">
            <span>{t('filters.toLabel')}</span>
            <Input
              type="date"
              value={value.end_date}
              onChange={(event) => onDraftChange({ ...value, end_date: event.target.value })}
              aria-label={t('filters.endDateAriaLabel')}
            />
          </label>
        </div>
      ) : null}
    </section>
  )
}

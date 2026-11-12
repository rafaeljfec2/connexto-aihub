import { Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { EXECUTION_OUTCOMES } from '../types'
import type { ExecutionOutcome, ExecutionsUiFilter } from '../types'
import { cn } from '../../../shared/ui/cn'
import { Input } from '../../../shared/ui/input'

type ExecutionsFiltersProps = {
  value: ExecutionsUiFilter
  providerOptions: string[]
  onChange: (next: ExecutionsUiFilter) => void
  onOutcomeChange: (outcome: ExecutionOutcome | 'all') => void
  onProviderChange: (providerId: string) => void
  onApplyModelFilter: () => void
}

export function ExecutionsFilters({
  value,
  providerOptions,
  onChange,
  onOutcomeChange,
  onProviderChange,
  onApplyModelFilter,
}: ExecutionsFiltersProps) {
  const { t } = useTranslation('executions')

  return (
    <div className="executions-filters">
      <label className="executions-search">
        <Search className="size-3.5 text-muted" aria-hidden="true" />
        <Input
          type="search"
          placeholder={t('filters.legacy.searchPlaceholder')}
          value={value.search}
          onChange={(event) => onChange({ ...value, search: event.target.value })}
          className="border-0 bg-transparent p-0 shadow-none focus-visible:ring-0"
        />
      </label>

      <div className="executions-chips" role="group" aria-label={t('filters.legacy.outcomeAriaLabel')}>
        <button
          type="button"
          className={cn('executions-chip', value.outcome === 'all' && 'executions-chip-active')}
          onClick={() => onOutcomeChange('all')}
        >
          {t('filters.legacy.all')}
        </button>
        {EXECUTION_OUTCOMES.map((outcome) => (
          <button
            key={outcome}
            type="button"
            className={cn('executions-chip', value.outcome === outcome && 'executions-chip-active')}
            onClick={() => onOutcomeChange(outcome)}
          >
            {t(`filters.outcomes.${outcome}`)}
          </button>
        ))}
      </div>

      <select
        className="executions-select"
        value={value.provider_id}
        onChange={(event) => onProviderChange(event.target.value)}
        aria-label={t('filters.legacy.providerAriaLabel')}
      >
        <option value="">{t('filters.legacy.allProviders')}</option>
        {providerOptions.map((providerId) => (
          <option key={providerId} value={providerId}>
            {providerId}
          </option>
        ))}
      </select>

      <Input
        className="executions-model-input"
        placeholder={t('filters.legacy.modelPlaceholder')}
        value={value.model_id}
        onChange={(event) => onChange({ ...value, model_id: event.target.value })}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            event.preventDefault()
            onApplyModelFilter()
          }
        }}
        aria-label={t('filters.legacy.modelAriaLabel')}
      />

      <button
        type="button"
        className="executions-chip executions-chip-apply"
        onClick={onApplyModelFilter}
      >
        {t('filters.legacy.applyModelFilter')}
      </button>
    </div>
  )
}

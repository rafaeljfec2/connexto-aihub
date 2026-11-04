import { Search } from 'lucide-react'
import { FilterChips, Input } from '../../../shared/ui'
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
  readonly providers: readonly string[]
  readonly onChange: (value: GenerationFilterDraft) => void
  readonly onApply: () => void
}

const outcomes = [
  { id: 'all', label: 'All' },
  ...EXECUTION_OUTCOMES.map((outcome) => ({ id: outcome, label: outcome.replace('_', ' ') })),
] as const

export function GenerationsFilters({
  search,
  onSearchChange,
  value,
  providers,
  onChange,
  onApply,
}: GenerationsFiltersProps) {
  return (
    <section className="generations-filters" aria-label="Generation filters">
      <label className="generations-search">
        <Search aria-hidden="true" className="size-3.5 shrink-0 text-muted" />
        <Input
          type="search"
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder="Search loaded results by ID, model, provider, or outcome"
          aria-label="Search loaded generations"
          className="border-0 bg-transparent p-0 shadow-none focus-visible:ring-0"
        />
      </label>

      <FilterChips
        options={[...outcomes]}
        selectedId={value.outcome}
        onSelect={(outcome) => onChange({ ...value, outcome: outcome as GenerationFilterDraft['outcome'] })}
        label="Outcome"
        role="group"
        aria-label="Outcome filter"
      />

      <label className="generations-filter-field">
        <span>Provider</span>
        <select
          value={value.provider_id}
          onChange={(event) => onChange({ ...value, provider_id: event.target.value })}
          aria-label="Provider filter"
        >
          <option value="">All providers</option>
          {providers.map((provider) => <option key={provider} value={provider}>{provider}</option>)}
        </select>
      </label>

      <label className="generations-filter-field generations-model-field">
        <span>Model ID</span>
        <Input
          value={value.model_id}
          onChange={(event) => onChange({ ...value, model_id: event.target.value })}
          onKeyDown={(event) => {
            if (event.key !== 'Enter') return
            event.preventDefault()
            onApply()
          }}
          placeholder="Any model"
          aria-label="Model ID filter"
        />
      </label>

      <label className="generations-filter-field">
        <span>From</span>
        <Input
          type="date"
          value={value.start_date}
          onChange={(event) => onChange({ ...value, start_date: event.target.value })}
          aria-label="Start date filter"
        />
      </label>
      <label className="generations-filter-field">
        <span>To</span>
        <Input
          type="date"
          value={value.end_date}
          onChange={(event) => onChange({ ...value, end_date: event.target.value })}
          aria-label="End date filter"
        />
      </label>

      <button type="button" className="generations-apply" onClick={onApply}>
        Apply filters
      </button>
    </section>
  )
}

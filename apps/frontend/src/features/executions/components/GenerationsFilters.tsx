import { Filter, Search } from 'lucide-react'
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

const outcomeLabels: Record<ExecutionOutcome | 'all', string> = {
  all: 'Todas',
  completed: 'Concluídas',
  failed: 'Falhas',
  canceled: 'Canceladas',
  timed_out: 'Tempo esgotado',
}

const outcomes = [
  { id: 'all' as const, label: outcomeLabels.all },
  ...EXECUTION_OUTCOMES.map((outcome) => ({ id: outcome, label: outcomeLabels[outcome] })),
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
  const showCustomDates = periodPreset === 'custom'

  return (
    <section className="generations-filter-bar" aria-label="Filtros de gerações">
      <div className="generations-filter-primary">
        <label className="generations-search">
          <Search aria-hidden="true" className="size-3.5 shrink-0 text-muted" />
          <Input
            type="search"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder="Buscar por request, modelo ou provedor…"
            aria-label="Buscar gerações carregadas"
            className="border-0 bg-transparent p-0 shadow-none focus-visible:ring-0"
          />
        </label>

        <FilterChips
          className="generations-outcome-chips"
          options={[...outcomes]}
          selectedId={value.outcome}
          onSelect={(outcome) => onOutcomeChange(outcome as GenerationFilterDraft['outcome'])}
          aria-label="Filtrar por resultado"
        />

        <select
          value={value.provider_id}
          onChange={(event) => onProviderChange(event.target.value)}
          className="generations-compact-select"
          aria-label="Filtrar por provedor"
        >
          <option value="">Todos os provedores</option>
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
          aria-label="Período"
        >
          <option value="30d">Últimos 30 dias</option>
          <option value="7d">Últimos 7 dias</option>
          <option value="all">Todo o período</option>
          <option value="custom">Personalizado</option>
        </select>

        <button
          type="button"
          className="generations-filter-apply"
          onClick={onApply}
          aria-label="Aplicar filtros"
        >
          <Filter className="size-3.5" aria-hidden="true" />
        </button>
      </div>

      {showCustomDates ? (
        <div className="generations-filter-advanced">
          <label className="generations-filter-field">
            <span>Modelo</span>
            <Input
              value={value.model_id}
              onChange={(event) => onDraftChange({ ...value, model_id: event.target.value })}
              onKeyDown={(event) => {
                if (event.key !== 'Enter') return
                event.preventDefault()
                onApply()
              }}
              placeholder="Qualquer modelo"
              aria-label="Filtrar por ID do modelo"
            />
          </label>
          <label className="generations-filter-field">
            <span>De</span>
            <Input
              type="date"
              value={value.start_date}
              onChange={(event) => onDraftChange({ ...value, start_date: event.target.value })}
              aria-label="Data inicial"
            />
          </label>
          <label className="generations-filter-field">
            <span>Até</span>
            <Input
              type="date"
              value={value.end_date}
              onChange={(event) => onDraftChange({ ...value, end_date: event.target.value })}
              aria-label="Data final"
            />
          </label>
        </div>
      ) : null}
    </section>
  )
}

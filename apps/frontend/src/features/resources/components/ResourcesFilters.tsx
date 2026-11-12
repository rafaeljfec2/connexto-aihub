import { Input } from '../../../shared/ui/input'
import type { ResourcesUiFilter } from '../types'

type ResourcesFiltersProps = {
  draft: ResourcesUiFilter
  providerOptions: string[]
  onChange: (next: ResourcesUiFilter) => void
  onApply: () => void
}

export function ResourcesFilters({
  draft,
  providerOptions,
  onChange,
  onApply,
}: ResourcesFiltersProps) {
  return (
    <div className="resources-filters">
      <label className="grid gap-1">
        <span className="font-mono text-[0.5625rem] uppercase text-muted">De</span>
        <Input
          type="date"
          value={draft.from}
          onChange={(event) => onChange({ ...draft, from: event.target.value })}
          className="h-8 w-[140px] text-xs"
        />
      </label>
      <label className="grid gap-1">
        <span className="font-mono text-[0.5625rem] uppercase text-muted">Até</span>
        <Input
          type="date"
          value={draft.to}
          onChange={(event) => onChange({ ...draft, to: event.target.value })}
          className="h-8 w-[140px] text-xs"
        />
      </label>
      <label className="grid gap-1">
        <span className="font-mono text-[0.5625rem] uppercase text-muted">Provedor</span>
        <select
          value={draft.provider_id}
          onChange={(event) => onChange({ ...draft, provider_id: event.target.value })}
          className="resources-filters-select"
        >
          <option value="">Todos os provedores</option>
          {providerOptions.map((id) => (
            <option key={id} value={id}>
              {id}
            </option>
          ))}
        </select>
      </label>
      <label className="grid gap-1">
        <span className="font-mono text-[0.5625rem] uppercase text-muted">Modelo</span>
        <Input
          value={draft.model_id}
          onChange={(event) => onChange({ ...draft, model_id: event.target.value })}
          placeholder="Opcional"
          className="h-8 w-[160px] text-xs"
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              onApply()
            }
          }}
        />
      </label>
      <button
        type="button"
        className="h-8 rounded-md border border-line px-3 text-[0.6875rem] text-secondary hover:bg-white/[0.03]"
        onClick={onApply}
      >
        Aplicar período
      </button>
    </div>
  )
}

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
    <div className="mb-3 flex flex-wrap items-end gap-2 rounded-[10px] border border-line bg-surface/60 p-3">
      <label className="grid gap-1">
        <span className="font-mono text-[0.5625rem] uppercase text-muted">From</span>
        <Input
          type="date"
          value={draft.from}
          onChange={(event) => onChange({ ...draft, from: event.target.value })}
          className="h-8 w-[140px] text-xs"
        />
      </label>
      <label className="grid gap-1">
        <span className="font-mono text-[0.5625rem] uppercase text-muted">To</span>
        <Input
          type="date"
          value={draft.to}
          onChange={(event) => onChange({ ...draft, to: event.target.value })}
          className="h-8 w-[140px] text-xs"
        />
      </label>
      <label className="grid gap-1">
        <span className="font-mono text-[0.5625rem] uppercase text-muted">Provider filter</span>
        <select
          value={draft.provider_id}
          onChange={(event) => onChange({ ...draft, provider_id: event.target.value })}
          className="h-8 min-w-[160px] rounded-md border border-line bg-[#171E29] px-2 text-xs text-secondary"
        >
          <option value="">All providers</option>
          {providerOptions.map((id) => (
            <option key={id} value={id}>
              {id}
            </option>
          ))}
        </select>
      </label>
      <label className="grid gap-1">
        <span className="font-mono text-[0.5625rem] uppercase text-muted">Model id</span>
        <Input
          value={draft.model_id}
          onChange={(event) => onChange({ ...draft, model_id: event.target.value })}
          placeholder="Optional"
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
        Apply range
      </button>
    </div>
  )
}

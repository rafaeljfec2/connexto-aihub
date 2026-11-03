import type { CatalogFilter } from '../types'

type CatalogFiltersProps = {
  value: CatalogFilter
  onChange: (next: CatalogFilter) => void
}

export function CatalogFilters({ value, onChange }: CatalogFiltersProps) {
  return (
    <div className="catalog-filters">
      <label>
        Provider
        <input
          type="text"
          value={value.provider ?? ''}
          onChange={(event) => onChange({ ...value, provider: event.target.value || undefined })}
        />
      </label>
      <label>
        Capability
        <input
          type="text"
          value={value.capability ?? ''}
          onChange={(event) => onChange({ ...value, capability: event.target.value || undefined })}
        />
      </label>
      <label>
        Presence
        <input
          type="text"
          value={value.presence ?? ''}
          onChange={(event) => onChange({ ...value, presence: event.target.value || undefined })}
        />
      </label>
      <label>
        Listing state
        <input
          type="text"
          value={value.listing_state ?? ''}
          onChange={(event) => onChange({ ...value, listing_state: event.target.value || undefined })}
        />
      </label>
    </div>
  )
}

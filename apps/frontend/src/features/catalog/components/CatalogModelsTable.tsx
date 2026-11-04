import { Brain } from 'lucide-react'
import type { CatalogModel } from '../types'

type CatalogModelsTableProps = {
  models: CatalogModel[]
  isLoading: boolean
  isError: boolean
  selectedModelId?: string | null
  onSelectModel?: (model: CatalogModel) => void
  compareSet?: Set<string>
  onToggleCompare?: (model: CatalogModel) => void
}

export function CatalogModelsTable({
  models,
  isLoading,
  isError,
  selectedModelId,
  onSelectModel,
  compareSet = new Set(),
  onToggleCompare,
}: CatalogModelsTableProps) {
  if (isLoading) {
    return <p className="py-8 text-center text-xs text-muted">Loading catalog models…</p>
  }

  if (isError) {
    return (
      <div className="banner banner-error" role="alert">
        Catalog request failed. Check gateway access and filter values.
      </div>
    )
  }

  if (models.length === 0) {
    return <p className="empty-state py-8 text-center text-xs text-muted">No models match the current filters.</p>
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left">
        <thead>
          <tr className="border-b border-line font-mono text-[9px] uppercase tracking-[0.04em] text-muted">
            <th scope="col" className="px-3 py-2.5">
              Modelo
            </th>
            <th scope="col" className="px-3 py-2.5">
              Provedor
            </th>
            <th scope="col" className="px-3 py-2.5">
              Identificador
            </th>
            <th scope="col" className="px-3 py-2.5">
              Catálogo
            </th>
            {onToggleCompare ? (
              <th scope="col" className="px-3 py-2.5 text-center">
                Comparar
              </th>
            ) : null}
          </tr>
        </thead>
        <tbody className="divide-y divide-line/30 text-xs">
          {models.map((model) => {
            const key = `${model.provider}:${model.model_id}`
            const isSelected = selectedModelId === key
            const isChecked = compareSet.has(key)

            return (
              <tr
                key={key}
                onClick={() => onSelectModel?.(model)}
                className={`cursor-pointer transition-colors ${
                  isSelected
                    ? 'bg-cobalt/10 shadow-[inset_2px_0_#7896E8]'
                    : 'hover:bg-white/[0.02]'
                }`}
              >
                <td className="px-3 py-3">
                  <div className="flex items-center gap-2.5">
                    <span className="grid size-7 place-items-center rounded-lg border border-line bg-raised text-accent-light">
                      <Brain className="size-3.5" />
                    </span>
                    <div>
                      <strong className="block font-semibold text-primary">{model.name}</strong>
                      <div className="font-mono text-[9px] text-muted">{model.model_id}</div>
                    </div>
                  </div>
                </td>
                <td className="px-3 py-3 text-secondary">{model.provider}</td>
                <td className="px-3 py-3 font-mono text-[10px] text-muted">{model.model_id}</td>
                <td className="px-3 py-3">
                  <span className="inline-flex items-center gap-1 font-mono text-[9px] text-accent-light">
                    <i className="size-1 rounded-full bg-cobalt" />
                    {model.listing_state ?? 'Listado'}
                  </span>
                </td>
                {onToggleCompare ? (
                  <td
                    className="px-3 py-3 text-center"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <input
                      type="checkbox"
                      checked={isChecked}
                      onChange={() => onToggleCompare(model)}
                      aria-label={`Comparar ${model.name}`}
                      className="size-3.5 rounded border-line accent-cobalt cursor-pointer"
                    />
                  </td>
                ) : null}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

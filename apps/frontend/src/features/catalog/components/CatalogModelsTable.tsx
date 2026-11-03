import type { CatalogModel } from '../types'

type CatalogModelsTableProps = {
  models: CatalogModel[]
  isLoading: boolean
  isError: boolean
}

export function CatalogModelsTable({ models, isLoading, isError }: CatalogModelsTableProps) {
  if (isLoading) {
    return <p>Loading catalog models…</p>
  }

  if (isError) {
    return (
      <div className="banner banner-error" role="alert">
        Catalog request failed. Check gateway access and filter values.
      </div>
    )
  }

  if (models.length === 0) {
    return <p className="empty-state">No models match the current filters.</p>
  }

  return (
    <div className="table-scroll">
      <table className="data-table">
        <thead>
          <tr>
            <th scope="col">Provider</th>
            <th scope="col">Model</th>
            <th scope="col">Listing</th>
            <th scope="col">Presence</th>
            <th scope="col">Execution verification</th>
            <th scope="col">Capabilities</th>
          </tr>
        </thead>
        <tbody>
          {models.map((model) => (
            <tr key={`${model.provider}:${model.model_id}`}>
              <td>{model.provider}</td>
              <td>
                <strong>{model.name}</strong>
                <div className="muted">{model.model_id}</div>
              </td>
              <td>{model.listing_state}</td>
              <td>{model.presence}</td>
              <td>{model.execution_verification}</td>
              <td>
                {Object.entries(model.capabilities).map(([name, capability]) => (
                  <div key={name} className="capability-chip">
                    {name}: {capability.state} ({capability.provenance})
                  </div>
                ))}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

import { Dialog } from '../../../shared/ui'
import type { CatalogModel } from '../types'

type ModelCompareModalProps = {
  models: CatalogModel[]
  open: boolean
  onClose: () => void
}

export function ModelCompareModal({ models, open, onClose }: ModelCompareModalProps) {
  if (!open) {
    return null
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(value) => {
        if (!value) {
          onClose()
        }
      }}
      title="Comparar modelos"
      description="Metadados e capacidades disponíveis lado a lado"
    >
      <div className="overflow-x-auto">
        <table className="catalog-compare-table">
          <thead>
            <tr>
              <th>Modelo</th>
              <th>Provedor</th>
              <th>Contexto</th>
              <th>Capacidades</th>
              <th>Presença</th>
              <th>Verificação</th>
            </tr>
          </thead>
          <tbody>
            {models.map((model) => {
              const caps = Object.keys(model.capabilities ?? {})
              return (
                <tr key={`${model.provider}:${model.model_id}`}>
                  <td className="font-semibold text-[#EEF2F8]">
                    <div>{model.name}</div>
                    <div className="font-mono text-[9px] text-[#8190A5]">{model.model_id}</div>
                  </td>
                  <td>{model.provider}</td>
                  <td className="font-mono text-[#8290A5]">Não informada</td>
                  <td>
                    {caps.length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {caps.map((capability) => (
                          <span
                            key={capability}
                            className="rounded bg-raised px-1.5 py-0.5 font-mono text-[9px] text-accent-light"
                          >
                            {capability}
                          </span>
                        ))}
                      </div>
                    ) : (
                      <span className="font-mono text-[#8290A5]">Não informadas</span>
                    )}
                  </td>
                  <td className="font-mono text-[#8290A5]">{model.presence}</td>
                  <td className="font-mono text-[#8290A5]">{model.execution_verification}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      <p className="catalog-compare-footnote">
        Campos não retornados pelo catálogo oficial do provedor aparecem como &quot;Não informado&quot;.
        A comparação não estima preços ou limites.
      </p>
    </Dialog>
  )
}

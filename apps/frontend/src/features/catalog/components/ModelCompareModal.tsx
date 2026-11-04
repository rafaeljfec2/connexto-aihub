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
      onOpenChange={(v) => {
        if (!v) onClose()
      }}
      title="Comparar modelos"
      description="Metadados e capacidades disponíveis lado a lado"
    >
      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs">
          <thead>
            <tr className="border-b border-line font-mono text-[9px] uppercase tracking-[0.04em] text-muted">
              <th className="px-3 py-2">Modelo</th>
              <th className="px-3 py-2">Provedor</th>
              <th className="px-3 py-2">Contexto</th>
              <th className="px-3 py-2">Capacidades</th>
              <th className="px-3 py-2">Presença</th>
              <th className="px-3 py-2">Verificação</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line/30">
            {models.map((m) => {
              const caps = Object.keys(m.capabilities ?? {})
              return (
                <tr key={`${m.provider}:${m.model_id}`} className="hover:bg-white/[0.02]">
                  <td className="px-3 py-3 font-semibold text-primary">
                    <div>{m.name}</div>
                    <div className="font-mono text-[9px] text-muted">{m.model_id}</div>
                  </td>
                  <td className="px-3 py-3 text-secondary">{m.provider}</td>
                  <td className="px-3 py-3 font-mono text-muted">Não informada</td>
                  <td className="px-3 py-3">
                    {caps.length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {caps.map((c) => (
                          <span
                            key={c}
                            className="rounded bg-raised px-1.5 py-0.5 font-mono text-[9px] text-accent-light"
                          >
                            {c}
                          </span>
                        ))}
                      </div>
                    ) : (
                      <span className="font-mono text-muted">Não informadas</span>
                    )}
                  </td>
                  <td className="px-3 py-3 font-mono text-muted">{m.presence}</td>
                  <td className="px-3 py-3 font-mono text-muted">
                    {m.execution_verification}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      <p className="mt-4 text-[10px] leading-relaxed text-muted">
        Campos não retornados pelo catálogo oficial do provedor aparecem como &quot;Não informado&quot;.
        A comparação não estima preços ou limites.
      </p>
    </Dialog>
  )
}

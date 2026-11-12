import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { Dialog } from '../../../shared/ui'
import { useProviderMutations } from '../hooks/useProviderMutations'
import { firstMutationError, mutationErrorMessage } from '../lib/mutationErrors'
import type { AdminProvider } from '../types'
import { ProviderActions } from './ProviderActions'
import { ProviderPatchForm } from './ProviderPatchForm'

type ProviderManageDrawerProps = {
  provider: AdminProvider | null
  open: boolean
  onClose: () => void
}

export function ProviderManageDrawer({
  provider,
  open,
  onClose,
}: ProviderManageDrawerProps) {
  if (!provider) {
    return null
  }

  const [confirmDelete, setConfirmDelete] = useState(false)
  const { validate, enable, disable, patch, remove } = useProviderMutations(provider.id)
  const actionsDisabled =
    validate.isPending || enable.isPending || disable.isPending || patch.isPending || remove.isPending

  const actionError = firstMutationError([
    validate.isError ? validate.error : undefined,
    enable.isError ? enable.error : undefined,
    disable.isError ? disable.error : undefined,
    remove.isError ? remove.error : undefined,
  ])

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) onClose()
      }}
      title={`Gerenciar provedor: ${provider.name}`}
      description={`Identificador técnico: ${provider.id}`}
      contentClassName="w-[min(100%,28rem)] max-h-[calc(100dvh-1rem)] sm:max-h-[min(92vh,42rem)]"
      bodyScroll
    >
      <div className="min-w-0 space-y-2 pt-0">
        <div className="min-w-0 overflow-x-hidden rounded-md border border-line bg-raised p-2">
          <h3 className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-primary">
            Ações operacionais
          </h3>
          <ProviderActions
            enabled={provider.enabled}
            validatePending={validate.isPending}
            togglePending={enable.isPending || disable.isPending}
            actionsDisabled={actionsDisabled}
            onValidate={() => validate.mutate()}
            onToggleEnabled={() =>
              provider.enabled ? disable.mutate() : enable.mutate()
            }
          />

          {actionError ? (
            <p className="mt-1 text-[10px] text-failure" role="alert">
              {actionError}
            </p>
          ) : null}
        </div>

        <div className="min-w-0 overflow-x-hidden rounded-md border border-line bg-raised p-2">
          <h3 className="mb-0.5 text-[10px] font-semibold uppercase tracking-wide text-primary">
            Credenciais e configuração
          </h3>
          <p className="mb-1.5 text-[10px] text-muted leading-snug">
            Os segredos salvos nunca são exibidos novamente pela interface. Preencha apenas
            quando desejar atualizar a credencial.
          </p>
          <ProviderPatchForm
            key={`${provider.id}-${provider.updated_at}`}
            provider={provider}
            isPending={patch.isPending}
            errorMessage={patch.isError ? mutationErrorMessage(patch.error) : undefined}
            onSubmit={(payload) => {
              patch.mutate(payload, {
                onSuccess: () => {
                  onClose()
                },
              })
            }}
          />
        </div>

        <div className="min-w-0 overflow-x-hidden rounded-md border border-red-500/25 bg-red-500/[0.04] p-2">
          <div className="flex items-center justify-between gap-2">
            <div>
              <h3 className="text-[10px] font-semibold uppercase tracking-wide text-red-300">
                Zona de risco
              </h3>
              <p className="text-[10px] text-muted leading-snug">
                Remove esta conta e seus modelos sincronizados do AI-HUB.
              </p>
            </div>
            {!confirmDelete ? (
              <button
                type="button"
                disabled={actionsDisabled}
                onClick={() => setConfirmDelete(true)}
                className="inline-flex shrink-0 items-center gap-1 rounded border border-red-500/35 bg-red-500/10 px-2.5 py-1 text-[11px] font-medium text-red-300 hover:bg-red-500/20 transition-colors cursor-pointer disabled:opacity-50"
              >
                <Trash2 className="size-3" />
                Remover conta
              </button>
            ) : (
              <div className="flex shrink-0 items-center gap-1.5">
                <button
                  type="button"
                  disabled={actionsDisabled}
                  onClick={() => setConfirmDelete(false)}
                  className="rounded border border-line px-2 py-1 text-[10px] font-medium text-secondary hover:bg-white/5 cursor-pointer disabled:opacity-50"
                >
                  Cancelar
                </button>
                <button
                  type="button"
                  disabled={actionsDisabled}
                  onClick={() => {
                    remove.mutate(undefined, {
                      onSuccess: () => {
                        setConfirmDelete(false)
                        onClose()
                      },
                    })
                  }}
                  className="rounded border border-red-500/50 bg-red-600 px-2.5 py-1 text-[10px] font-semibold text-white hover:bg-red-500 cursor-pointer disabled:opacity-50"
                >
                  {remove.isPending ? 'Removendo...' : 'Confirmar remoção'}
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </Dialog>
  )
}

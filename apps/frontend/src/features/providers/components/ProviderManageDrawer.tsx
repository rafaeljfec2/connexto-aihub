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

  const { validate, enable, disable, patch } = useProviderMutations(provider.id)
  const actionsDisabled =
    validate.isPending || enable.isPending || disable.isPending || patch.isPending

  const actionError = firstMutationError([
    validate.isError ? validate.error : undefined,
    enable.isError ? enable.error : undefined,
    disable.isError ? disable.error : undefined,
  ])

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) onClose()
      }}
      title={`Gerenciar provedor: ${provider.name}`}
      description={`Identificador técnico: ${provider.id}`}
    >
      <div className="space-y-4 pt-1">
        <div className="rounded-lg border border-line bg-raised p-3">
          <h3 className="mb-2 text-xs font-semibold text-primary">Ações operacionais</h3>
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
            <p className="mt-2 text-xs text-failure" role="alert">
              {actionError}
            </p>
          ) : null}
        </div>

        <div className="rounded-lg border border-line bg-raised p-3">
          <h3 className="mb-2 text-xs font-semibold text-primary">
            Credenciais e configuração
          </h3>
          <p className="mb-3 text-[11px] text-muted leading-relaxed">
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
      </div>
    </Dialog>
  )
}

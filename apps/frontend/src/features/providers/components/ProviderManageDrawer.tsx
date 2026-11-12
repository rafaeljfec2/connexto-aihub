import { useState } from 'react'
import { useTranslation } from 'react-i18next'
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

  const { t } = useTranslation('providers')
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

  const dangerAction = !confirmDelete ? (
    <button
      type="button"
      disabled={actionsDisabled}
      onClick={() => setConfirmDelete(true)}
      className="inline-flex h-8 w-full shrink-0 items-center justify-center gap-1.5 rounded-md border border-red-500/35 bg-red-500/10 px-3 text-[0.6875rem] font-medium text-red-300 hover:bg-red-500/20 transition-colors cursor-pointer disabled:opacity-50 sm:w-auto"
    >
      <Trash2 className="size-3.5" />
      {t('manage.removeAccount')}
    </button>
  ) : (
    <div className="flex flex-wrap items-center gap-1.5">
      <button
        type="button"
        disabled={actionsDisabled}
        onClick={() => setConfirmDelete(false)}
        className="h-8 rounded-md border border-line px-2.5 text-[0.6875rem] font-medium text-secondary hover:bg-white/5 cursor-pointer disabled:opacity-50"
      >
        {t('manage.cancel')}
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
        className="h-8 rounded-md border border-red-500/50 bg-red-600 px-3 text-[0.6875rem] font-semibold text-white hover:bg-red-500 cursor-pointer disabled:opacity-50"
      >
        {remove.isPending ? t('manage.removing') : t('manage.confirmRemove')}
      </button>
    </div>
  )

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) onClose()
      }}
      title={t('manage.title', { name: provider.name })}
      description={t('manage.description', { id: provider.id })}
      contentClassName="w-[min(100%-1rem,44rem)] max-h-[calc(100dvh-1rem)] sm:max-h-[94dvh]"
      bodyScroll
    >
      <div className="min-w-0 space-y-2.5 pt-0">
        <div className="min-w-0 overflow-x-hidden rounded-md border border-line bg-raised px-2.5 py-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h3 className="text-[0.625rem] font-semibold uppercase tracking-wide text-primary">
              {t('manage.opsTitle')}
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
          </div>

          {actionError ? (
            <p className="mt-1.5 text-[0.625rem] text-failure" role="alert">
              {actionError}
            </p>
          ) : null}
        </div>

        <div className="min-w-0 overflow-x-hidden rounded-md border border-line bg-raised px-3 py-2.5">
          <div className="mb-2 flex flex-wrap items-baseline justify-between gap-x-2 gap-y-0.5">
            <h3 className="text-[0.625rem] font-semibold uppercase tracking-wide text-primary">
              {t('manage.credentialsTitle')}
            </h3>
            <p className="text-[0.625rem] text-muted">
              {t('manage.credentialsHint')}
            </p>
          </div>
          <ProviderPatchForm
            key={`${provider.id}-${provider.updated_at}`}
            provider={provider}
            isPending={patch.isPending}
            errorMessage={patch.isError ? mutationErrorMessage(patch.error) : undefined}
            dangerAction={dangerAction}
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

import { useState } from 'react'
import { useProviderMutations } from '../hooks/useProviderMutations'
import { firstMutationError, mutationErrorMessage } from '../lib/mutationErrors'
import type { AdminProvider } from '../types'
import { ProviderActions } from './ProviderActions'
import { ProviderPatchForm } from './ProviderPatchForm'

type ProviderAdminCardProps = {
  provider: AdminProvider
}

export function ProviderAdminCard({ provider }: ProviderAdminCardProps) {
  const [expanded, setExpanded] = useState(false)
  const { validate, enable, disable, patch } = useProviderMutations(provider.id)
  const actionsDisabled =
    validate.isPending || enable.isPending || disable.isPending || patch.isPending

  const actionError = firstMutationError([
    validate.isError ? validate.error : undefined,
    enable.isError ? enable.error : undefined,
    disable.isError ? disable.error : undefined,
  ])

  return (
    <article className="card provider-admin-card">
      <header className="provider-admin-header">
        <div>
          <h3>{provider.id}</h3>
          <p>{provider.name}</p>
          <p>
            Enabled: {provider.enabled ? 'yes' : 'no'} · Credential:{' '}
            {provider.credential_configured ? 'configured' : 'missing'}
          </p>
          {provider.type === 'openrouter' ? (
            <p>Management API Key: {provider.management_credential_configured ? 'configured' : 'missing'}</p>
          ) : null}
        </div>
        <button type="button" className="ghost-button" onClick={() => setExpanded((value) => !value)}>
          {expanded ? 'Collapse' : 'Manage'}
        </button>
      </header>

      <ProviderActions
        enabled={provider.enabled}
        validatePending={validate.isPending}
        togglePending={enable.isPending || disable.isPending}
        actionsDisabled={actionsDisabled}
        onValidate={() => validate.mutate()}
        onToggleEnabled={() => (provider.enabled ? disable.mutate() : enable.mutate())}
      />

      {actionError ? (
        <p className="field-error" role="alert">
          {actionError}
        </p>
      ) : null}

      {expanded ? (
        <ProviderPatchForm
          key={`${provider.id}-${provider.updated_at}`}
          provider={provider}
          isPending={patch.isPending}
          errorMessage={patch.isError ? mutationErrorMessage(patch.error) : undefined}
          onSubmit={(values) => {
            if (Object.keys(values).length === 0) {
              return
            }
            patch.mutate(values)
          }}
        />
      ) : null}
    </article>
  )
}

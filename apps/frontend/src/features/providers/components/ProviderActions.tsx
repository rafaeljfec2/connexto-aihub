type ProviderActionsProps = {
  enabled: boolean
  validatePending: boolean
  togglePending: boolean
  actionsDisabled: boolean
  onValidate: () => void
  onToggleEnabled: () => void
}

export function ProviderActions({
  enabled,
  validatePending,
  togglePending,
  actionsDisabled,
  onValidate,
  onToggleEnabled,
}: ProviderActionsProps) {
  return (
    <div className="provider-actions">
      <button type="button" disabled={actionsDisabled} onClick={onValidate}>
        {validatePending ? 'Validating…' : 'Validate probe'}
      </button>
      <button type="button" disabled={actionsDisabled} onClick={onToggleEnabled}>
        {togglePending ? 'Updating…' : enabled ? 'Disable' : 'Enable'}
      </button>
    </div>
  )
}

import i18n from '../../../shared/i18n'

const TRANSPORT_ONLY_CATALOG_PROVIDER = 'connexto-proxy-cli-llms'

const KNOWN_PROVIDER_LABEL_KEYS = new Set([
  'openrouter',
  'cursor',
  'antigravity',
  'opencode-go',
])

/** Logical catalog provider label; transport adapters are not user-facing providers. */
export function catalogProviderLabel(providerId: string): string {
  if (providerId === TRANSPORT_ONLY_CATALOG_PROVIDER) {
    return i18n.t('catalog:providerLabel.cursor')
  }
  if (KNOWN_PROVIDER_LABEL_KEYS.has(providerId)) {
    return i18n.t(`catalog:providerLabel.${providerId}`)
  }
  return providerId
}

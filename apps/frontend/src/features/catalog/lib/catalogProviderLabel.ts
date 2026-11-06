const TRANSPORT_ONLY_CATALOG_PROVIDER = 'connexto-proxy-cli-llms'

/** Logical catalog provider label; transport adapters are not user-facing providers. */
export function catalogProviderLabel(providerId: string): string {
  if (providerId === TRANSPORT_ONLY_CATALOG_PROVIDER) {
    return 'cursor'
  }
  return providerId
}

export type OperationalEnvelope = {
  schema_version: number
}

export type ReadinessChangedEvent = OperationalEnvelope & {
  status: string
  checks?: Record<string, string>
}

export type ProviderStatusEvent = OperationalEnvelope & {
  providers: Array<{
    id: string
    enabled: boolean
    state: string
  }>
}

export const operationalEventNames = {
  readinessChanged: 'readiness_changed',
  providerStatus: 'provider_status',
} as const

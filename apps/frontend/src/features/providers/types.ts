export type AdminProvider = {
  id: string
  name: string
  type: string
  endpoint: string
  enabled: boolean
  credential_configured: boolean
  management_credential_configured?: boolean
  created_at: string
  updated_at: string
}

export type ProviderListResponse = {
  providers: AdminProvider[]
}

export type UpdateProviderInput = {
  name?: string
  endpoint?: string
  credential?: string
  management_credential?: string
}

export type AvailableProvider = {
  id: string
  name: string
  description: string
  logo: string
  credential_type: string
  credential_label: string
  credential_placeholder: string
  capabilities: string[]
  connection_type: string
  default_endpoint: string
  enabled: boolean
}

export type AvailableProvidersResponse = {
  providers: AvailableProvider[]
}

export type CreateProviderInput = {
  id: string
  name?: string
  type?: string
  endpoint?: string
  credential?: string
  management_credential?: string
  enabled?: boolean
}

export type SyncModelsResponse = {
  provider: string
  outcome: string
  counts: {
    discovered: number
    added: number
    updated: number
    unchanged: number
    missing: number
  }
  duration_ms: number
}

export type ProviderConnectionStatus = {
  id: string
  name: string
  type: string
  enabled: boolean
  connection_state: string
  credential_state: string
  last_validated_at?: string
  last_synced_at?: string
  imported_models_count: number
  sync_outcome?: string
  error_message?: string
}

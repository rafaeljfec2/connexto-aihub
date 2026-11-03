export type AdminProvider = {
  id: string
  name: string
  type: string
  endpoint: string
  enabled: boolean
  credential_configured: boolean
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
}

import { ApiError } from '../../../shared/api/client'

export type CatalogSyncFeedback = {
  tone: 'error' | 'warning' | 'info'
  message: string
}

export function formatCatalogSyncFeedback(error: unknown): CatalogSyncFeedback | undefined {
  if (!error) {
    return undefined
  }

  if (error instanceof ApiError) {
    if (error.status === 409 && error.type === 'sync_in_progress') {
      return {
        tone: 'warning',
        message: 'A catalog synchronization is already in progress. Wait for it to finish before starting another.',
      }
    }
    if (error.status === 409 && error.type === 'provider_disabled') {
      return {
        tone: 'warning',
        message: 'Catalog sync was rejected because the provider is disabled.',
      }
    }
    if (error.status === 409) {
      return {
        tone: 'warning',
        message: error.message || 'Catalog sync conflict (409).',
      }
    }
    return {
      tone: 'error',
      message: error.message,
    }
  }

  if (error instanceof Error) {
    return { tone: 'error', message: error.message }
  }

  return { tone: 'error', message: 'Catalog sync failed.' }
}

import i18n from '../../../shared/i18n'
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
        message: i18n.t('catalog:sync.inProgressConflict'),
      }
    }
    if (error.status === 409 && error.type === 'provider_disabled') {
      return {
        tone: 'warning',
        message: i18n.t('catalog:sync.providerDisabled'),
      }
    }
    if (error.status === 409) {
      return {
        tone: 'warning',
        message: error.message || i18n.t('catalog:sync.conflict409'),
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

  return { tone: 'error', message: i18n.t('catalog:sync.failed') }
}

import i18n from '../../../shared/i18n'

export function formatOverviewBackendTag(dataOrigin?: string): string {
  if (dataOrigin === 'demonstrative') {
    return i18n.t('overview:backendTag.demonstrative')
  }
  if (dataOrigin === 'partial') {
    return i18n.t('overview:backendTag.partial')
  }
  if (dataOrigin === 'unavailable') {
    return i18n.t('overview:backendTag.unavailable')
  }
  return i18n.t('overview:backendTag.live')
}

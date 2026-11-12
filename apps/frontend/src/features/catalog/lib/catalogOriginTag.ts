import i18n from '../../../shared/i18n'

export function formatCatalogOriginTag(dataOrigin?: string): string {
  if (dataOrigin === 'demonstrative') {
    return i18n.t('catalog:originTag.demonstrative')
  }
  if (dataOrigin === 'partial') {
    return i18n.t('catalog:originTag.partial')
  }
  if (dataOrigin === 'unavailable') {
    return i18n.t('catalog:originTag.unavailable')
  }
  return i18n.t('catalog:originTag.live')
}

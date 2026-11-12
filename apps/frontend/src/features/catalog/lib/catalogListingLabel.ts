import i18n from '../../../shared/i18n'
import type { CatalogModel } from '../types'

export function catalogListingLabel(model: CatalogModel): string {
  return model.listing_state || i18n.t('catalog:page.listedDefault')
}

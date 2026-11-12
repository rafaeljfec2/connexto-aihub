import { useTranslation } from 'react-i18next'
import { Brain } from 'lucide-react'
import { Link } from 'react-router-dom'
import { CopyTextButton } from '../../../shared/ui'
import { formatCatalogContextLimit, formatPricingCell } from '../lib/formatCatalogPricing'
import { catalogProviderLabel } from '../lib/catalogProviderLabel'
import { catalogListingLabel } from '../lib/catalogListingLabel'
import type { CatalogModel } from '../types'

type CatalogModelDetailPanelProps = {
  model: CatalogModel | null
}

export function CatalogModelDetailPanel({ model }: CatalogModelDetailPanelProps) {
  const { t } = useTranslation('catalog')

  if (!model) {
    return (
      <aside className="catalog-detail-panel">
        <div className="catalog-detail-top">
          <strong>{t('table.detailTitle')}</strong>
          <span>{t('table.selectModel')}</span>
        </div>
        <p className="py-8 text-center text-xs text-muted">{t('detail.selectHint')}</p>
      </aside>
    )
  }

  const caps = Object.entries(model.capabilities ?? {})
  const providerLabel = catalogProviderLabel(model.provider)

  return (
    <aside className="catalog-detail-panel">
      <div className="catalog-detail-top">
        <strong>{t('detail.title')}</strong>
        <span>{t('detail.selectedBadge')}</span>
      </div>

      <div className="catalog-detail-scroll">
        <div className="catalog-detail-hero">
          <div className="catalog-detail-glyph">
            <Brain className="size-[18px]" aria-hidden />
          </div>
          <h2>{model.name}</h2>
          <div className="catalog-detail-idrow">
            <p>{model.model_id}</p>
            <CopyTextButton text={model.model_id} label={t('detail.copyIdAria', { id: model.model_id })} />
          </div>
        </div>

        <div>
          <div className="catalog-metarow">
            <span>{t('detail.providerLabel')}</span>
            <strong>{providerLabel}</strong>
          </div>
          <div className="catalog-metarow">
            <span>{t('detail.originLabel')}</span>
            <strong>{t('detail.originValue')}</strong>
          </div>
          <div className="catalog-metarow">
            <span>{t('detail.registryLabel')}</span>
            <strong>
              {catalogListingLabel(model) === t('table.listed') ? t('detail.listedFound') : catalogListingLabel(model)}
            </strong>
          </div>
        </div>

        <div className="catalog-detail-metadata">
          <div className="catalog-detail-metadata-head">
            {t('detail.metadataHead')} <span>{t('detail.metadataWhenAvailable')}</span>
          </div>
          <div className="catalog-metadata-grid">
            <div>
              <small>{t('detail.contextWindow')}</small>
              <strong>{formatCatalogContextLimit(model.limits?.context ?? undefined)}</strong>
            </div>
            <div>
              <small>{t('table.capabilities')}</small>
              <strong>
                {caps.length > 0
                  ? t('detail.capabilitiesDeclared', { count: caps.length })
                  : t('compare.capabilitiesNotListed')}
              </strong>
            </div>
            <div>
              <small>{t('detail.inputPrice')}</small>
              <strong>{formatPricingCell(model.pricing, 'input')}</strong>
            </div>
            <div>
              <small>{t('detail.outputPrice')}</small>
              <strong>{formatPricingCell(model.pricing, 'output')}</strong>
            </div>
            <div>
              <small>{t('detail.cacheRead')}</small>
              <strong>{formatPricingCell(model.pricing, 'cache_read')}</strong>
            </div>
            <div>
              <small>{t('detail.cacheWrite')}</small>
              <strong>{formatPricingCell(model.pricing, 'cache_write')}</strong>
            </div>
          </div>
        </div>

        <div className="catalog-detail-note">{t('detail.connectorNote', { provider: providerLabel })}</div>

        <Link to="/providers" className="catalog-detail-btn">
          {t('detail.viewProviderQuotas')}
        </Link>

        <p className="catalog-detail-foot">{t('detail.pricingFootnote')}</p>
      </div>
    </aside>
  )
}

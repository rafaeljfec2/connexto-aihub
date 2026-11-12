import { useTranslation } from 'react-i18next'
import { Dialog } from '../../../shared/ui'
import { formatCatalogContextLimit, formatPricingCell } from '../lib/formatCatalogPricing'
import type { CatalogModel } from '../types'

type ModelCompareModalProps = {
  models: CatalogModel[]
  open: boolean
  onClose: () => void
}

export function ModelCompareModal({ models, open, onClose }: ModelCompareModalProps) {
  const { t } = useTranslation('catalog')

  if (!open) {
    return null
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(value) => {
        if (!value) {
          onClose()
        }
      }}
      title={t('compare.title')}
      description={t('compare.description')}
    >
      <div className="overflow-x-auto">
        <table className="catalog-compare-table">
          <thead>
            <tr>
              <th>{t('table.colModel')}</th>
              <th>{t('compare.colProvider')}</th>
              <th>{t('compare.colContext')}</th>
              <th>{t('compare.colInput')}</th>
              <th>{t('compare.colOutput')}</th>
              <th>{t('compare.colCacheRead')}</th>
              <th>{t('compare.colCacheWrite')}</th>
              <th>{t('compare.colCapabilities')}</th>
              <th>{t('compare.colPresence')}</th>
              <th>{t('compare.colExecVerify')}</th>
            </tr>
          </thead>
          <tbody>
            {models.map((model) => {
              const caps = Object.keys(model.capabilities ?? {})
              return (
                <tr key={`${model.provider}:${model.model_id}`}>
                  <td className="font-semibold text-[#EEF2F8]">
                    <div>{model.name}</div>
                    <div className="font-mono text-[0.5625rem] text-[#8190A5]">{model.model_id}</div>
                  </td>
                  <td>{model.provider}</td>
                  <td className="font-mono text-[#8290A5]">
                    {formatCatalogContextLimit(model.limits?.context ?? undefined)}
                  </td>
                  <td className="font-mono text-[#8290A5]">{formatPricingCell(model.pricing, 'input')}</td>
                  <td className="font-mono text-[#8290A5]">{formatPricingCell(model.pricing, 'output')}</td>
                  <td className="font-mono text-[#8290A5]">
                    {formatPricingCell(model.pricing, 'cache_read')}
                  </td>
                  <td className="font-mono text-[#8290A5]">
                    {formatPricingCell(model.pricing, 'cache_write')}
                  </td>
                  <td>
                    {caps.length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {caps.map((capability) => (
                          <span
                            key={capability}
                            className="rounded bg-raised px-1.5 py-0.5 font-mono text-[0.5625rem] text-accent-light"
                          >
                            {capability}
                          </span>
                        ))}
                      </div>
                    ) : (
                      <span className="font-mono text-[#8290A5]">{t('compare.capabilitiesNotListed')}</span>
                    )}
                  </td>
                  <td className="font-mono text-[#8290A5]">{model.presence}</td>
                  <td className="font-mono text-[#8290A5]">{model.execution_verification}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      <p className="catalog-compare-footnote">{t('compare.footnote')}</p>
    </Dialog>
  )
}

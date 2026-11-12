import { ShieldCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Skeleton } from '../../../shared/ui'

type ProviderQuotaPageFooterProps = Readonly<{
  isLoading?: boolean
}>

export function ProviderQuotaPageFooter({ isLoading = false }: ProviderQuotaPageFooterProps) {
  const { t } = useTranslation('providers')

  return (
    <div className="providers-page-footer" data-testid="providers-page-footer">
      {isLoading ? (
        <div
          className="providers-bottominfo"
          data-testid="provider-quota-footer-skeleton"
          aria-hidden="true"
        >
          <Skeleton className="size-3.5 shrink-0 rounded-full" motionSafe />
          <Skeleton className="h-2.5 w-full max-w-[42rem]" motionSafe />
        </div>
      ) : (
        <div className="providers-bottominfo">
          <ShieldCheck className="size-3.5 shrink-0 text-accent-light" aria-hidden />
          <p>
            <strong className="text-primary">{t('page.footerReliableTitle')}</strong>{' '}
            {t('page.footerReliableBody')}
          </p>
        </div>
      )}
      {isLoading ? (
        <footer className="providers-page-footnote" aria-hidden="true">
          <Skeleton className="h-2 w-52" motionSafe />
          <Skeleton className="h-2 w-36" motionSafe />
        </footer>
      ) : (
        <footer className="providers-page-footnote">
          <span>{t('page.footnoteSources')}</span>
          <span>{t('page.footnoteBrand')}</span>
        </footer>
      )}
    </div>
  )
}

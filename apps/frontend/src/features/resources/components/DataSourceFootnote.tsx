import { useTranslation } from 'react-i18next'

type DataSourceFootnoteProps = {
  sources: string[]
  updatedAt?: number
}

export function DataSourceFootnote({ sources, updatedAt }: DataSourceFootnoteProps) {
  const { t } = useTranslation('resources')
  const timeLabel =
    updatedAt !== undefined
      ? new Date(updatedAt).toISOString().replace('T', ' ').slice(0, 19)
      : '—'

  return (
    <footer className="resources-footnote">
      <span>
        {t('footnote.apiPrefix')} {sources.join(' · ')}
      </span>
      <span>{t('footnote.hubTimestamp', { time: timeLabel })}</span>
    </footer>
  )
}

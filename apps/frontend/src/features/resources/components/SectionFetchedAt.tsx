import { useTranslation } from 'react-i18next'
import { formatFetchedAtUtc } from '../lib/formatFetchedAt'

type SectionFetchedAtProps = {
  updatedAtMs?: number
}

export function SectionFetchedAt({ updatedAtMs }: SectionFetchedAtProps) {
  const { t } = useTranslation('resources')

  if (updatedAtMs === undefined || updatedAtMs <= 0) {
    return null
  }

  return (
    <p className="mt-2 font-mono text-[0.5625rem] text-[#758196]">
      {t('fetchedAt.snapshot', { time: formatFetchedAtUtc(updatedAtMs) })}
    </p>
  )
}

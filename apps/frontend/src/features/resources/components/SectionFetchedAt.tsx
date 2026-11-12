import { formatFetchedAtUtc } from '../lib/formatFetchedAt'

type SectionFetchedAtProps = {
  updatedAtMs?: number
}

export function SectionFetchedAt({ updatedAtMs }: SectionFetchedAtProps) {
  if (updatedAtMs === undefined || updatedAtMs <= 0) {
    return null
  }

  return (
    <p className="mt-2 font-mono text-[0.5625rem] text-[#758196]">
      Snapshot fetched {formatFetchedAtUtc(updatedAtMs)} UTC
    </p>
  )
}

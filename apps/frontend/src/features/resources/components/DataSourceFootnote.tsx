type DataSourceFootnoteProps = {
  sources: string[]
  updatedAt?: number
}

export function DataSourceFootnote({ sources, updatedAt }: DataSourceFootnoteProps) {
  const timeLabel =
    updatedAt !== undefined
      ? new Date(updatedAt).toISOString().replace('T', ' ').slice(0, 19)
      : '—'

  return (
    <footer className="mt-3 flex flex-wrap items-center justify-between gap-2 font-mono text-[0.5625rem] text-[#758196]">
      <span>API · {sources.join(' · ')}</span>
      <span>Snapshot fetched {timeLabel} UTC</span>
    </footer>
  )
}

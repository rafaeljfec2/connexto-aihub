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
    <footer className="resources-footnote">
      <span>API · {sources.join(' · ')}</span>
      <span>CONNEXTO AI HUB · {timeLabel} UTC</span>
    </footer>
  )
}

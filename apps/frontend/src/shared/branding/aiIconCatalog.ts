import iconToc from '../../../vendor/ai-icons/icon-toc.json'

export type AiIconTocEntry = {
  id: string
  title: string
  fullTitle: string
  docsUrl: string
  group: string
  color?: string
  desc?: string
  param?: {
    hasAvatar?: boolean
    hasColor?: boolean
    hasText?: boolean
  }
}

const entries = iconToc as AiIconTocEntry[]

const byDocsUrl = new Map<string, AiIconTocEntry>()
for (const entry of entries) {
  if (entry.docsUrl) {
    byDocsUrl.set(entry.docsUrl.toLowerCase(), entry)
  }
}

/** Lookup a catalog entry by upstream docsUrl (e.g. "open-router"). */
export function findIconByDocsUrl(docsUrl: string): AiIconTocEntry | undefined {
  const key = docsUrl.trim().toLowerCase()
  if (key.length === 0) {
    return undefined
  }
  return byDocsUrl.get(key)
}

export function listAiIconCatalog(): readonly AiIconTocEntry[] {
  return entries
}

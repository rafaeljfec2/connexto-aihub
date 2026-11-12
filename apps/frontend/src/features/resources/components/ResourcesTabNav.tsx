import { cn } from '../../../shared/ui/cn'

import type { ResourcesTabId } from '../types'

type ResourcesTabNavProps = {
  active: ResourcesTabId
  onChange: (tab: ResourcesTabId) => void
}

const tabs: Array<{ id: ResourcesTabId; label: string }> = [
  { id: 'overview', label: 'Overview' },
  { id: 'policies', label: 'Resource policies' },
  { id: 'concurrency', label: 'Concurrency' },
  { id: 'provider-usage', label: 'Provider usage' },
]

export function ResourcesTabNav({ active, onChange }: ResourcesTabNavProps) {
  return (
    <div className="mb-3 flex flex-wrap items-center gap-1 border-b border-line" role="tablist">
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={active === tab.id}
          className={cn(
            'border-b-2 px-3 py-2 text-[0.6875rem] transition-colors',
            active === tab.id
              ? 'border-cobalt text-[#E6EDFA]'
              : 'border-transparent text-[#8E9BB0] hover:text-secondary',
          )}
          onClick={() => onChange(tab.id)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  )
}

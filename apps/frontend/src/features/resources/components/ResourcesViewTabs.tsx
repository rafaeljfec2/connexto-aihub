import { cn } from '../../../shared/ui/cn'
import type { ResourcesTabId } from '../types'

type ResourcesViewTabsProps = {
  active: ResourcesTabId
  onChange: (tab: ResourcesTabId) => void
}

const tabs: Array<{ id: ResourcesTabId; label: string }> = [
  { id: 'overview', label: 'Visão geral' },
  { id: 'policies', label: 'Políticas de recursos' },
  { id: 'concurrency', label: 'Concorrência' },
]

export function ResourcesViewTabs({ active, onChange }: ResourcesViewTabsProps) {
  return (
    <div className="resources-tabs" role="tablist">
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={active === tab.id}
          className={cn('resources-tab', active === tab.id && 'is-active')}
          onClick={() => onChange(tab.id)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  )
}

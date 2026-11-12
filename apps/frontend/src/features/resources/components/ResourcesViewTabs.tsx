import { useTranslation } from 'react-i18next'
import { cn } from '../../../shared/ui/cn'
import type { ResourcesTabId } from '../types'

type ResourcesViewTabsProps = {
  active: ResourcesTabId
  onChange: (tab: ResourcesTabId) => void
}

const tabIds: ResourcesTabId[] = ['overview', 'policies', 'concurrency']

export function ResourcesViewTabs({ active, onChange }: ResourcesViewTabsProps) {
  const { t } = useTranslation('resources')

  return (
    <div className="resources-tabs" role="tablist">
      {tabIds.map((id) => (
        <button
          key={id}
          type="button"
          role="tab"
          aria-selected={active === id}
          className={cn('resources-tab', active === id && 'is-active')}
          onClick={() => onChange(id)}
        >
          {t(`tabs.${id}`)}
        </button>
      ))}
    </div>
  )
}

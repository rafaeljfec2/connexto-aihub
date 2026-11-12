import {
  Activity,
  Brain,
  Layers,
  LayoutDashboard,
  Plug,
  Settings,
  SlidersHorizontal,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { NavLink } from 'react-router-dom'
import { cn } from '../../shared/ui/cn'

type SidebarItem = {
  readonly to?: string
  readonly labelKey: string
  readonly icon: LucideIcon
  readonly end?: boolean
  readonly disabled?: boolean
}

const operationItems: readonly SidebarItem[] = [
  { to: '/', labelKey: 'overview', icon: LayoutDashboard, end: true },
  { to: '/providers', labelKey: 'providers', icon: Plug },
  { to: '/catalog', labelKey: 'catalog', icon: Layers },
  { to: '/intelligence', labelKey: 'intelligence', icon: Brain },
  { to: '/executions', labelKey: 'executions', icon: Activity },
  { to: '/resources', labelKey: 'resources', icon: SlidersHorizontal },
]

const systemItems: readonly SidebarItem[] = [
  { labelKey: 'settings', icon: Settings, disabled: true },
]

export function AppSidebarNavigation({
  collapsed,
  onNavigate,
}: {
  readonly collapsed: boolean
  readonly onNavigate: () => void
}) {
  const { t } = useTranslation(['nav', 'common'])

  return (
    <nav aria-label={t('navPrimary', { ns: 'common' })} className="flex flex-col">
      <NavSection label={t('sectionOperation')} items={operationItems} collapsed={collapsed} onNavigate={onNavigate} />
      <div className="my-4 h-px bg-line" />
      <NavSection label={t('sectionSystem')} items={systemItems} collapsed={collapsed} onNavigate={onNavigate} />
    </nav>
  )
}

function NavSection({
  label,
  items,
  collapsed,
  onNavigate,
}: {
  readonly label: string
  readonly items: readonly SidebarItem[]
  readonly collapsed: boolean
  readonly onNavigate: () => void
}) {
  const { t } = useTranslation(['nav', 'common'])

  return (
    <>
      {!collapsed ? (
        <p className="px-2.5 pb-2 font-mono text-[0.625rem] uppercase tracking-[0.11em] text-muted">{label}</p>
      ) : null}
      <div className="grid gap-1">
        {items.map((item) => {
          const Icon = item.icon
          const itemLabel = t(item.labelKey)
          if (item.disabled || !item.to) {
            return (
              <button
                key={item.labelKey}
                type="button"
                disabled
                aria-disabled="true"
                title={t('unavailable', { ns: 'common', label: itemLabel })}
                className={cn(
                  'flex cursor-not-allowed items-center gap-2.5 rounded-lg border border-transparent px-2.5 py-2.5 text-left text-xs text-muted/70',
                  collapsed && 'justify-center px-2',
                )}
              >
                <Icon className="size-4 shrink-0 opacity-70" aria-hidden />
                {!collapsed ? <span>{itemLabel}</span> : null}
              </button>
            )
          }

          return (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              onClick={onNavigate}
              title={itemLabel}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-2.5 rounded-lg border px-2.5 py-2.5 text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cobalt/40',
                  collapsed && 'justify-center px-2',
                  isActive
                    ? 'border-cobalt/25 bg-[#1F2B42] text-[#EAF0FC]'
                    : 'border-transparent text-secondary hover:bg-white/[0.03] hover:text-primary',
                )
              }
            >
              {({ isActive }) => (
                <>
                  <Icon
                    className={cn('size-4 shrink-0', isActive ? 'text-accent-light' : 'opacity-75')}
                    aria-hidden
                  />
                  {!collapsed ? <span>{itemLabel}</span> : null}
                </>
              )}
            </NavLink>
          )
        })}
      </div>
    </>
  )
}

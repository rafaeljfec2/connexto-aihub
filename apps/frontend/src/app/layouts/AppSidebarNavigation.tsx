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
import { NavLink } from 'react-router-dom'
import { cn } from '../../shared/ui/cn'

type SidebarItem = {
  readonly to?: string
  readonly label: string
  readonly icon: LucideIcon
  readonly end?: boolean
  readonly disabled?: boolean
}

const operationItems: readonly SidebarItem[] = [
  { to: '/', label: 'Visão geral', icon: LayoutDashboard, end: true },
  { to: '/providers', label: 'Provedores', icon: Plug },
  { to: '/catalog', label: 'Modelos', icon: Layers },
  { label: 'Inteligência', icon: Brain, disabled: true },
  { to: '/executions', label: 'Execuções', icon: Activity },
  { to: '/resources', label: 'Recursos', icon: SlidersHorizontal },
]

const systemItems: readonly SidebarItem[] = [
  { label: 'Configurações', icon: Settings, disabled: true },
]

export function AppSidebarNavigation({
  collapsed,
  onNavigate,
}: {
  readonly collapsed: boolean
  readonly onNavigate: () => void
}) {
  return (
    <nav aria-label="Primary navigation" className="flex flex-col">
      <NavSection label="OPERAÇÃO" items={operationItems} collapsed={collapsed} onNavigate={onNavigate} />
      <div className="my-4 h-px bg-line" />
      <NavSection label="SISTEMA" items={systemItems} collapsed={collapsed} onNavigate={onNavigate} />
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
  return (
    <>
      {!collapsed ? (
        <p className="px-2.5 pb-2 font-mono text-[10px] uppercase tracking-[0.11em] text-muted">
          {label}
        </p>
      ) : null}
      <div className="grid gap-1">
        {items.map((item) => {
          const Icon = item.icon
          if (item.disabled || !item.to) {
            return (
              <button
                key={item.label}
                type="button"
                disabled
                aria-disabled="true"
                title={`${item.label} indisponível`}
                className={cn(
                  'flex cursor-not-allowed items-center gap-2.5 rounded-lg border border-transparent px-2.5 py-2.5 text-left text-xs text-muted/70',
                  collapsed && 'justify-center px-2',
                )}
              >
                <Icon className="size-4 shrink-0 opacity-70" aria-hidden />
                {!collapsed ? <span>{item.label}</span> : null}
              </button>
            )
          }

          return (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              onClick={onNavigate}
              title={item.label}
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
                  {!collapsed ? <span>{item.label}</span> : null}
                </>
              )}
            </NavLink>
          )
        })}
      </div>
    </>
  )
}

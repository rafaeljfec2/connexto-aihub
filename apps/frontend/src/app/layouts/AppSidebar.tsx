import { useEffect, useRef } from 'react'
import { FocusScope } from '@radix-ui/react-focus-scope'
import {
  Activity,
  FlaskConical,
  Layers,
  LayoutDashboard,
  PanelLeftClose,
  PanelLeftOpen,
  Plug,
  SlidersHorizontal,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { NavLink } from 'react-router-dom'
import type { OperationalConnectionState } from '../../shared/realtime/useOperationalEvents'
import { cn } from '../../shared/ui/cn'
import { Button } from '../../shared/ui/button'

type NavItemDef = {
  to?: string
  label: string
  icon: LucideIcon
  end?: boolean
  soon?: boolean
}

const operationNav: NavItemDef[] = [
  { to: '/', label: 'Visão geral', icon: LayoutDashboard, end: true },
  { to: '/providers', label: 'Provedores', icon: Plug },
  { to: '/catalog', label: 'Modelos', icon: Layers },
  { to: '/executions', label: 'Execuções', icon: Activity },
  { to: '/resources', label: 'Recursos', icon: SlidersHorizontal },
]

const systemNav: NavItemDef[] = [
  { to: '/playground', label: 'Playground', icon: FlaskConical },
]

type AppSidebarProps = {
  collapsed: boolean
  mobileOpen: boolean
  connectionState: OperationalConnectionState
  onToggleCollapsed: () => void
  onCloseMobile: () => void
}

function sseFootCopy(state: OperationalConnectionState): {
  label: string
  detail: string
  dotClass: string
} {
  if (state === 'live') {
    return {
      label: 'Gateway operacional',
      detail: 'CONEXÃO OPERACIONAL ATIVA',
      dotClass: 'bg-healthy',
    }
  }
  if (state === 'connecting') {
    return {
      label: 'Gateway conectando',
      detail: 'CONECTANDO EVENT STREAM',
      dotClass: 'bg-warning',
    }
  }
  return {
    label: 'Gateway degradado',
    detail: 'RECONEXÃO AGENDADA',
    dotClass: 'bg-failure',
  }
}

function NavSection({
  label,
  items,
  collapsed,
  onNavigate,
}: {
  label: string
  items: NavItemDef[]
  collapsed: boolean
  onNavigate?: () => void
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
          if (item.soon || !item.to) {
            return (
              <span
                key={item.label}
                className={cn(
                  'flex items-center gap-2.5 rounded-lg border border-transparent px-2.5 py-2.5 text-xs text-muted/80',
                  collapsed && 'justify-center px-2',
                )}
                aria-disabled
                title={item.label}
              >
                <Icon className="size-4 shrink-0 opacity-70" aria-hidden />
                {!collapsed ? (
                  <>
                    <span>{item.label}</span>
                    <span className="ml-auto font-mono text-[9px] text-[#748196]">Soon</span>
                  </>
                ) : null}
              </span>
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

export function AppSidebar({
  collapsed,
  mobileOpen,
  connectionState,
  onToggleCollapsed,
  onCloseMobile,
}: AppSidebarProps) {
  const sidebarRef = useRef<HTMLElement>(null)
  const restoreFocusRef = useRef<HTMLElement | null>(null)
  const sseFoot = sseFootCopy(connectionState)

  useEffect(() => {
    if (!mobileOpen) {
      return
    }

    restoreFocusRef.current =
      document.activeElement instanceof HTMLElement ? document.activeElement : null

    const focusFirst = (): void => {
      const root = sidebarRef.current
      if (!root) {
        return
      }
      const target = root.querySelector<HTMLElement>(
        'a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"])',
      )
      target?.focus()
    }

    const frame = requestAnimationFrame(focusFirst)

    const onKeyDown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape') {
        event.preventDefault()
        onCloseMobile()
      }
    }

    document.addEventListener('keydown', onKeyDown)
    return () => {
      cancelAnimationFrame(frame)
      document.removeEventListener('keydown', onKeyDown)
      restoreFocusRef.current?.focus()
      restoreFocusRef.current = null
    }
  }, [mobileOpen, onCloseMobile])

  return (
    <>
      {mobileOpen ? (
        <button
          type="button"
          className="fixed inset-0 z-40 bg-black/60 lg:hidden"
          aria-label="Close navigation"
          onClick={onCloseMobile}
        />
      ) : null}
      <FocusScope trapped={mobileOpen} loop asChild>
        <aside
          ref={sidebarRef}
          id="app-sidebar"
          role="navigation"
          aria-label="Primary navigation"
          aria-modal={mobileOpen ? true : undefined}
          className={cn(
            'z-50 flex h-full flex-col overflow-y-auto scrollbar-slim border-r border-line bg-sidebar px-3.5 pb-4 pt-[22px] transition-[width,transform] duration-200',
            collapsed ? 'w-[72px]' : 'w-[224px]',
            'fixed inset-y-0 left-0 lg:static lg:translate-x-0',
            mobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
          )}
        >
          <div className={cn('mb-6 flex items-center gap-2.5 px-2.5', collapsed && 'justify-center px-0')}>
            <div className="grid size-[30px] place-items-center rounded-[9px] border border-cobalt/30 bg-accent-deep">
              <Activity className="size-[17px] text-accent-light" strokeWidth={1.7} aria-hidden />
            </div>
            {!collapsed ? (
              <div>
                <div className="text-[15px] font-semibold tracking-[-0.03em]">connexto</div>
                <div className="mt-0.5 font-mono text-[9px] uppercase tracking-[0.13em] text-muted">
                  AI Hub · Control plane
                </div>
              </div>
            ) : null}
          </div>

          {!collapsed ? (
            <div className="mx-1 mb-6 flex items-center gap-2.5 rounded-[10px] border border-line bg-white/[0.025] px-3 py-2.5">
              <span className="size-[7px] shrink-0 rounded-full bg-cobalt shadow-[0_0_0_4px_rgba(120,150,232,0.12)]" />
              <div>
                <strong className="block text-[11px] font-semibold">Workspace principal</strong>
                <small className="mt-0.5 block text-[10px] text-muted">Instância de desenvolvimento</small>
              </div>
            </div>
          ) : null}

          <nav aria-label="Primary navigation" className="flex flex-col">
            <NavSection
              label="OPERAÇÃO"
              items={operationNav}
              collapsed={collapsed}
              onNavigate={onCloseMobile}
            />
            <div className="my-4 h-px bg-line" />
            <NavSection label="SISTEMA" items={systemNav} collapsed={collapsed} onNavigate={onCloseMobile} />
          </nav>

          <div className="mt-auto space-y-3 pt-4">
            <Button
              variant="ghost"
              size="sm"
              className={cn('hidden w-full lg:inline-flex', collapsed ? 'justify-center px-0' : '')}
              onClick={onToggleCollapsed}
              aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            >
              {collapsed ? <PanelLeftOpen className="size-4" /> : <PanelLeftClose className="size-4" />}
              {!collapsed ? <span>Collapse</span> : null}
            </Button>
            <div
              className={cn(
                'flex items-center gap-2 border-t border-line pt-3.5',
                collapsed && 'justify-center',
              )}
            >
              <span className={cn('size-2 shrink-0 rounded-full', sseFoot.dotClass)} />
              {!collapsed ? (
                <div>
                  <span className="block text-[11px] text-[#BEC8D6]">{sseFoot.label}</span>
                  <small className="mt-0.5 block font-mono text-[9px] uppercase text-[#8290A5]">
                    {sseFoot.detail}
                  </small>
                </div>
              ) : null}
            </div>
          </div>
        </aside>
      </FocusScope>
    </>
  )
}

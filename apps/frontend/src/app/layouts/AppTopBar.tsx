import { useQuery } from '@tanstack/react-query'
import { ChevronDown, Menu, PanelLeftClose, PanelLeftOpen } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { getRuntimeMeta, overviewQueryKeys } from '../../features/overview/api/queries'
import { Button } from '../../shared/ui/button'
import { APP_SHELL_TOP_BAR_CLASS } from './appShellLayout'
import { AppUserMenu } from './AppUserMenu'

type AppTopBarProps = {
  crumb: string
  sidebarCollapsed?: boolean
  onOpenMobileNav: () => void
  onToggleSidebar?: () => void
}

export function AppTopBar({ crumb, sidebarCollapsed, onOpenMobileNav, onToggleSidebar }: AppTopBarProps) {
  const { t } = useTranslation(['common', 'overview'])
  const runtimeMeta = useQuery({
    queryKey: overviewQueryKeys.runtimeMeta,
    queryFn: getRuntimeMeta,
    refetchInterval: false,
    retry: 1,
  })

  const env = runtimeMeta.data?.environment
  const environmentLabel =
    env === 'production'
      ? t('shell.envProduction', { ns: 'common' })
      : env === 'staging' || env === 'homologation'
        ? t('shell.envStaging', { ns: 'common' })
        : t('shell.envDevelopment', { ns: 'common' })

  const isDemonstrative = runtimeMeta.data?.data_origin === 'demonstrative'
  const originLabel = isDemonstrative
    ? t('backendTag.demonstrative', { ns: 'overview' })
    : t('backendTag.live', { ns: 'overview' })

  return (
    <header className={APP_SHELL_TOP_BAR_CLASS}>
      <div className="flex min-w-0 items-center gap-3">
        <Button
          variant="icon"
          size="sm"
          className="lg:hidden"
          onClick={onOpenMobileNav}
          aria-label={t('shell.openNav', { ns: 'common' })}
        >
          <Menu className="size-4" />
        </Button>
        {onToggleSidebar ? (
          <Button
            variant="icon"
            size="sm"
            className="hidden text-[#94A1B3] hover:text-[#E0E7F1] lg:inline-flex"
            onClick={onToggleSidebar}
            aria-label={
              sidebarCollapsed
                ? t('shell.expandSidebar', { ns: 'common' })
                : t('shell.collapseSidebar', { ns: 'common' })
            }
            title={
              sidebarCollapsed
                ? t('shell.expandSidebar', { ns: 'common' })
                : t('shell.collapseSidebar', { ns: 'common' })
            }
          >
            {sidebarCollapsed ? (
              <PanelLeftOpen className="size-4" />
            ) : (
              <PanelLeftClose className="size-4" />
            )}
          </Button>
        ) : null}
        <p className="truncate text-[0.6875rem] text-[#94A1B3]">
          {t('shell.workspaceCrumb', { ns: 'common' })} <span className="mx-2 text-[#58677B]">/</span>{' '}
          <b className="font-medium text-[#E0E7F1]">{crumb}</b>
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-3 lg:gap-4">
        <div className="hidden items-center gap-2 rounded-[7px] border border-line bg-white/[0.025] px-2.5 py-1.5 text-[0.625rem] text-[#BEC8D6] sm:flex">
          <span className="size-1.5 rounded-full bg-cobalt" />
          {environmentLabel}
          <ChevronDown className="size-3 text-[#8592A7]" aria-hidden />
        </div>
        <span className="hidden font-mono text-[0.625rem] tracking-[0.05em] text-[#8A98AB] md:inline">
          {originLabel}
        </span>
        <AppUserMenu />
      </div>
    </header>
  )
}

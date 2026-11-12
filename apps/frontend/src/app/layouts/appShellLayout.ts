import { cn } from '../../shared/ui/cn'

/**
 * Default AppShell content column (centered max-width + horizontal gutters).
 * Applied only to `<main>` page content in AppShell; do not add page-level horizontal padding.
 *
 * The top bar stays full width (`APP_SHELL_TOP_BAR_CLASS`).
 *
 * Visual tokens live in styles.css.
 */
export const APP_SHELL_CONTENT_CLASS = 'app-shell-content'

/** Full-width shell header; not constrained to the content column. */
export const APP_SHELL_TOP_BAR_CLASS = 'app-shell-topbar'

/**
 * Routes that fill the viewport without scrolling the main column (dashboard-style).
 * All other AppShell routes scroll inside the content frame.
 */
export const APP_SHELL_VIEWPORT_FIT_PATHS = ['/', '/providers', '/catalog', '/resources'] as const

export type AppShellViewportFitPath = (typeof APP_SHELL_VIEWPORT_FIT_PATHS)[number]

export function isAppShellViewportFitPath(pathname: string): boolean {
  return (APP_SHELL_VIEWPORT_FIT_PATHS as readonly string[]).includes(pathname)
}

export function getAppShellMainContentClassName(pathname: string): string {
  const viewportFit = isAppShellViewportFitPath(pathname)
  return cn(
    APP_SHELL_CONTENT_CLASS,
    viewportFit
      ? 'flex h-full min-h-0 flex-col overflow-hidden'
      : 'min-h-0 overflow-y-auto scrollbar-slim lg:h-full',
  )
}

/**
 * Standard root wrapper for feature pages rendered inside AppShell.
 * Horizontal margins come from the shell only (`APP_SHELL_CONTENT_CLASS`).
 */
export const APP_PAGE_CLASS = 'app-page'

export function appPageClassName(...extra: Array<string | false | null | undefined>): string {
  return cn(APP_PAGE_CLASS, ...extra)
}

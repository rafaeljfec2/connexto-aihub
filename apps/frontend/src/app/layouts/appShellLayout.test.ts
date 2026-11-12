import { describe, expect, it } from 'vitest'
import {
  APP_SHELL_CONTENT_CLASS,
  APP_SHELL_VIEWPORT_FIT_PATHS,
  getAppShellMainContentClassName,
  isAppShellViewportFitPath,
} from './appShellLayout'

describe('appShellLayout', () => {
  it('declares viewport-fit paths', () => {
    expect(APP_SHELL_VIEWPORT_FIT_PATHS).toEqual(['/', '/providers', '/catalog', '/resources'])
  })

  it('detects viewport-fit routes', () => {
    expect(isAppShellViewportFitPath('/')).toBe(true)
    expect(isAppShellViewportFitPath('/providers')).toBe(true)
    expect(isAppShellViewportFitPath('/catalog')).toBe(true)
    expect(isAppShellViewportFitPath('/resources')).toBe(true)
    expect(isAppShellViewportFitPath('/executions')).toBe(false)
  })

  it('builds main content classes with shared frame', () => {
    expect(getAppShellMainContentClassName('/catalog')).toContain(APP_SHELL_CONTENT_CLASS)
    expect(getAppShellMainContentClassName('/')).toContain('overflow-hidden')
    expect(getAppShellMainContentClassName('/catalog')).toContain('overflow-hidden')
    expect(getAppShellMainContentClassName('/executions')).toContain('overflow-y-auto')
  })
})

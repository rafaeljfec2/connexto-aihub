import { ApiError } from '../../../shared/api/client'
import i18n from '../../../shared/i18n'

const PROXY_CURSOR_AUTH_FAILED = 'proxy_cursor_auth_failed'

function cursorAuthChromiumUnavailable(): string {
  return i18n.t('providers:cursorErrors.chromiumUnavailable')
}

function cursorAuthGenericFailure(): string {
  return i18n.t('providers:cursorErrors.genericAuthFailure')
}

function suggestsChromiumIssue(message: string, type: string): boolean {
  if (type === PROXY_CURSOR_AUTH_FAILED) {
    const lower = message.toLowerCase()
    return (
      lower.includes('chromium') ||
      lower.includes('browser_unavailable') ||
      lower.includes('chromium_executable_path')
    )
  }
  const lower = message.toLowerCase()
  return lower.includes('browser_unavailable') || lower.includes('chromium')
}

export function cursorProxyAuthErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.type === PROXY_CURSOR_AUTH_FAILED && error.message.trim() !== '') {
      return error.message
    }
    if (suggestsChromiumIssue(error.message, error.type)) {
      return cursorAuthChromiumUnavailable()
    }
  }
  if (error instanceof Error) {
    if (suggestsChromiumIssue(error.message, '')) {
      return cursorAuthChromiumUnavailable()
    }
    if (error.message.trim() !== '') {
      return `${cursorAuthGenericFailure()} ${error.message}`
    }
  }
  return cursorAuthGenericFailure()
}

export function mutationErrorMessage(error: unknown): string | undefined {
  if (error instanceof ApiError) {
    return error.message
  }
  if (error instanceof Error) {
    return error.message
  }
  return undefined
}

export function firstMutationError(errors: unknown[]): string | undefined {
  for (const error of errors) {
    const message = mutationErrorMessage(error)
    if (message) {
      return message
    }
  }
  return undefined
}

import { ApiError } from '../../../shared/api/client'

const PROXY_CURSOR_AUTH_FAILED = 'proxy_cursor_auth_failed'

const cursorAuthChromiumUnavailablePt =
  'Chromium indisponível no proxy. Defina CHROMIUM_EXECUTABLE_PATH no Connexto Proxy e reinicie o serviço (em Docker: imagem já inclui /usr/bin/chromium).'

const cursorAuthGenericFailurePt = 'Falha ao iniciar login web do Cursor no proxy.'

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
    if (suggestsChromiumIssue(error.message, error.type)) {
      return cursorAuthChromiumUnavailablePt
    }
    if (error.type === PROXY_CURSOR_AUTH_FAILED && error.message.trim() !== '') {
      return `${cursorAuthGenericFailurePt} ${error.message}`
    }
  }
  if (error instanceof Error) {
    if (suggestsChromiumIssue(error.message, '')) {
      return cursorAuthChromiumUnavailablePt
    }
    if (error.message.trim() !== '') {
      return `${cursorAuthGenericFailurePt} ${error.message}`
    }
  }
  return cursorAuthGenericFailurePt
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

import { getUiAuthSystem } from '../../features/auth/api/uiAuthSystem'

export type ApiErrorBody = {
  type: string
  message: string
}

type gatewayErrorEnvelope = {
  error: ApiErrorBody
}

export class ApiError extends Error {
  readonly status: number
  readonly type: string

  constructor(status: number, body: ApiErrorBody) {
    super(body.message)
    this.status = status
    this.type = body.type
  }
}

const authPathsNoRefresh = new Set([
  '/api/ui/login',
  '/api/ui/signup',
  '/api/ui/refresh',
  '/api/ui/forgot-password',
  '/api/ui/reset-password',
])

let refreshInFlight: Promise<boolean> | null = null

async function refreshSessionOnce(): Promise<boolean> {
  if (refreshInFlight) {
    return refreshInFlight
  }
  refreshInFlight = fetch('/api/ui/refresh', { method: 'POST', credentials: 'same-origin' })
    .then((response) => response.ok)
    .catch(() => false)
    .finally(() => {
      refreshInFlight = null
    })
  return refreshInFlight
}

function pathFromInput(input: string): string {
  if (input.startsWith('/')) {
    return input.split('?')[0] ?? input
  }
  try {
    return new URL(input, window.location.origin).pathname
  } catch {
    return input
  }
}

export async function fetchJson<T>(input: string, init?: RequestInit, retried = false): Promise<T> {
  const response = await fetch(input, {
    credentials: 'same-origin',
    ...init,
  })

  if (
    response.status === 401 &&
    !retried &&
    getUiAuthSystem() === 'users' &&
    !authPathsNoRefresh.has(pathFromInput(input))
  ) {
    const refreshed = await refreshSessionOnce()
    if (refreshed) {
      return fetchJson<T>(input, init, true)
    }
    if (typeof window !== 'undefined') {
      const redirect = encodeURIComponent(`${window.location.pathname}${window.location.search}`)
      window.location.assign(`/login?redirect=${redirect}`)
    }
  }

  if (!response.ok) {
    let body: ApiErrorBody = { type: 'request_failed', message: response.statusText }
    try {
      const raw = (await response.json()) as ApiErrorBody | gatewayErrorEnvelope
      if ('error' in raw && raw.error !== undefined && typeof raw.error.type === 'string') {
        body = raw.error
      } else if ('type' in raw && typeof raw.type === 'string') {
        body = raw as ApiErrorBody
      }
    } catch {
      // keep default body
    }
    throw new ApiError(response.status, body)
  }
  return (await response.json()) as T
}

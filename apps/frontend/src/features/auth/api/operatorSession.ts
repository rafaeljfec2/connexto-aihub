import { fetchJson } from '../../../shared/api/client'
import { setUiAuthSystem } from './uiAuthSystem'

export type OperatorSessionMode = 'loopback_trust' | 'passphrase' | 'none' | 'users'

export type UiUser = {
  id: string
  email: string
  role: string
}

export type OperatorSession = {
  authenticated: boolean
  mode: OperatorSessionMode
  system: 'operator' | 'users'
  user?: UiUser
}

export const operatorSessionQueryKey = ['operator', 'session'] as const

export function fetchOperatorSession(): Promise<OperatorSession> {
  return fetchJson<OperatorSession>('/api/ui/session').then((session) => {
    setUiAuthSystem(session.system)
    return session
  })
}

export async function loginOperator(passphrase: string): Promise<void> {
  await fetchJson<{ ok: boolean }>('/api/ui/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ passphrase }),
  })
}

export async function loginUser(email: string, password: string): Promise<void> {
  await fetchJson<{ user: UiUser }>('/api/ui/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
}

export async function logoutOperator(): Promise<void> {
  await fetchJson<{ ok: boolean }>('/api/ui/logout', {
    method: 'POST',
  })
}

export async function requestPasswordReset(email: string): Promise<{ message: string }> {
  return fetchJson<{ message: string }>('/api/ui/forgot-password', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email }),
  })
}

export async function resetPassword(token: string, password: string): Promise<void> {
  await fetchJson<{ ok: boolean }>('/api/ui/reset-password', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token, password }),
  })
}

export function requiresOperatorLogin(session: OperatorSession | undefined): boolean {
  if (!session) {
    return false
  }
  if (session.authenticated) {
    return false
  }
  return session.system === 'users' || session.mode === 'none'
}

export function isUsersAuthSystem(session: OperatorSession | undefined): boolean {
  return session?.system === 'users'
}

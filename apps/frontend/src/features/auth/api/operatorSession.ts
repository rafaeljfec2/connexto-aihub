import { fetchJson } from '../../../shared/api/client'

export type OperatorSessionMode = 'loopback_trust' | 'passphrase' | 'none'

export type OperatorSession = {
  authenticated: boolean
  mode: OperatorSessionMode
}

export const operatorSessionQueryKey = ['operator', 'session'] as const

export function fetchOperatorSession(): Promise<OperatorSession> {
  return fetchJson<OperatorSession>('/api/ui/session')
}

export async function loginOperator(passphrase: string): Promise<void> {
  await fetchJson<{ ok: boolean }>('/api/ui/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ passphrase }),
  })
}

export async function logoutOperator(): Promise<void> {
  await fetchJson<{ ok: boolean }>('/api/ui/logout', {
    method: 'POST',
  })
}

export function requiresOperatorLogin(session: OperatorSession | undefined): boolean {
  if (!session) {
    return false
  }
  return !session.authenticated && session.mode === 'none'
}

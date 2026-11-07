import type {
  CursorProxyUsageCollectResponse,
  CursorProxyUsageSessionResponse,
} from '../api/queries'

export function cursorCollectReady(
  session: CursorProxyUsageSessionResponse,
  collect: CursorProxyUsageCollectResponse,
): boolean {
  return (
    session.status === 'authenticated' && collect.status !== 'authentication_required'
  )
}

export function messageForCursorCollectReadiness(
  session: CursorProxyUsageSessionResponse,
  collect: CursorProxyUsageCollectResponse,
): { success: boolean; message: string } {
  if (session.status !== 'authenticated') {
    return {
      success: false,
      message:
        'Nenhuma sessão persistida no proxy para coleta. Use «Login web» no mesmo endpoint do provedor.',
    }
  }
  if (collect.status === 'authentication_required') {
    return {
      success: false,
      message: `Sessão listada no proxy (conta ${session.accountId}), mas a coleta retornou authentication_required. Repita o login ou confira SESSION_STORAGE_PATH no proxy.`,
    }
  }
  const touched = session.touchedAt ?? '—'
  return {
    success: true,
    message: `Coleta pronta no proxy (conta ${session.accountId}). Probe: ${collect.status}. Último toque: ${touched}. Use Atualizar em Franquias.`,
  }
}

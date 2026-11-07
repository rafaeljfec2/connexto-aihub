import type {
  CursorProxyUsageCollectResponse,
  CursorProxyUsageSessionResponse,
} from '../api/queries'

const COLLECT_BLOCKING_STATUSES = new Set([
  'authentication_required',
  'authentication_pending',
  'session_expired',
  'collection_failed',
  'browser_unavailable',
  'timeout',
  'unsupported_response',
  'challenge_required',
])

export function cursorCollectReady(
  session: CursorProxyUsageSessionResponse,
  collect: CursorProxyUsageCollectResponse,
): boolean {
  return session.status === 'authenticated' && collect.status === 'available'
}

export function isCollectProbeBlocking(status: string): boolean {
  return COLLECT_BLOCKING_STATUSES.has(status)
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
  if (collect.status === 'authentication_pending') {
    return {
      success: false,
      message:
        'Login web ainda em andamento no Chromium do proxy. Feche o fluxo no navegador ou aguarde; a coleta só roda com a janela de login fechada.',
    }
  }
  if (collect.status === 'browser_unavailable') {
    return {
      success: false,
      message:
        'Chromium ocupado ou perfil em uso. Feche qualquer janela do login do proxy; se persistir, encerre processos google-chrome/chromium e reinicie o proxy (pnpm dev). Depois use «Ver status».',
    }
  }
  if (collect.status === 'authentication_required') {
    return {
      success: false,
      message: `Sessão listada no proxy (conta ${session.accountId}), mas a coleta retornou authentication_required. Repita o login ou confira SESSION_STORAGE_PATH no proxy.`,
    }
  }
  if (collect.status === 'collection_failed') {
    return {
      success: false,
      message:
        'Coleta falhou no proxy (collection_failed). Feche instâncias Chromium extras, reinicie pnpm dev e repita o login.',
    }
  }
  if (collect.status === 'challenge_required') {
    return {
      success: false,
      message:
        'Login ou coleta bloqueados no navegador do proxy (Google/x.ai ou deteção de automação). Use GitHub ou e-mail no login, termine em cursor.com/dashboard ou Agents, e evite Google. Se persistir, feche a janela do login e clique «Ver status» de novo.',
    }
  }
  if (collect.status === 'session_expired') {
    return {
      success: false,
      message: `Sessão gravada no proxy (toque: ${session.touchedAt ?? '—'}), mas a coleta em grok.com/dashboard falhou (session_expired). Após o login, aguarde o dashboard de uso abrir ou repita «Login web» e conclua até ver o painel de uso no Chromium do proxy.`,
    }
  }
  if (collect.status === 'timeout') {
    return {
      success: false,
      message: 'Coleta no proxy expirou (timeout). Tente «Ver status» ou «Login web» novamente.',
    }
  }
  if (collect.status !== 'available') {
    return {
      success: false,
      message: `Sessão no proxy: authenticated; coleta: ${collect.status}. Não é possível atualizar Franquias até o probe retornar available.`,
    }
  }
  const touched = session.touchedAt ?? '—'
  return {
    success: true,
    message: `Coleta pronta no proxy (conta ${session.accountId}). Último toque: ${touched}. Use Atualizar em Franquias.`,
  }
}

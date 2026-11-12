import i18n from '../../../shared/i18n'
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

export type CursorCollectReadinessCode =
  | 'no_persisted_session'
  | 'authentication_pending'
  | 'browser_unavailable'
  | 'authentication_required'
  | 'collection_failed'
  | 'challenge_required'
  | 'session_expired'
  | 'timeout'
  | 'probe_blocked'
  | 'ready'

export type CursorCollectReadiness = {
  success: boolean
  message: string
  code?: CursorCollectReadinessCode
}

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
): CursorCollectReadiness {
  if (session.status !== 'authenticated') {
    return {
      success: false,
      code: 'no_persisted_session',
      message: i18n.t('providers:collectReadiness.noPersistedSession'),
    }
  }
  if (collect.status === 'authentication_pending') {
    return {
      success: false,
      code: 'authentication_pending',
      message: i18n.t('providers:collectReadiness.authPending'),
    }
  }
  if (collect.status === 'browser_unavailable') {
    return {
      success: false,
      code: 'browser_unavailable',
      message: i18n.t('providers:collectReadiness.browserUnavailable'),
    }
  }
  if (collect.status === 'authentication_required') {
    return {
      success: false,
      code: 'authentication_required',
      message: i18n.t('providers:collectReadiness.authRequired', { accountId: session.accountId }),
    }
  }
  if (collect.status === 'collection_failed') {
    return {
      success: false,
      code: 'collection_failed',
      message: i18n.t('providers:collectReadiness.collectionFailed'),
    }
  }
  if (collect.status === 'challenge_required') {
    return {
      success: false,
      code: 'challenge_required',
      message: i18n.t('providers:collectReadiness.challengeRequired'),
    }
  }
  if (collect.status === 'session_expired') {
    return {
      success: false,
      code: 'session_expired',
      message: i18n.t('providers:collectReadiness.sessionExpired', {
        touchedAt: session.touchedAt ?? '—',
      }),
    }
  }
  if (collect.status === 'timeout') {
    return {
      success: false,
      code: 'timeout',
      message: i18n.t('providers:collectReadiness.timeout'),
    }
  }
  if (collect.status !== 'available') {
    return {
      success: false,
      code: 'probe_blocked',
      message: i18n.t('providers:collectReadiness.probeBlocked', { status: collect.status }),
    }
  }
  const touched = session.touchedAt ?? '—'
  return {
    success: true,
    code: 'ready',
    message: i18n.t('providers:collectReadiness.ready', {
      accountId: session.accountId,
      touchedAt: touched,
    }),
  }
}

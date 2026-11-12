import i18n from '../../../shared/i18n'
import type { AntigravityProxyRuntimeSessionClearResponse } from '../api/queries'

export function antigravityClearSessionsSuccessMessage(
  summary: AntigravityProxyRuntimeSessionClearResponse,
): string {
  const parts: string[] = []
  if (summary.mapped_acp_ids_count !== undefined) {
    parts.push(i18n.t('providers:patchMessages.mappedGateway', { count: summary.mapped_acp_ids_count }))
  }
  if (summary.listed_acp_sessions_count !== undefined) {
    parts.push(i18n.t('providers:patchMessages.listedAcp', { count: summary.listed_acp_sessions_count }))
  }
  if (summary.acp_sessions_closed !== undefined) {
    parts.push(i18n.t('providers:patchMessages.closedSessions', { count: summary.acp_sessions_closed }))
  }
  if (summary.acp_close_failures !== undefined && summary.acp_close_failures > 0) {
    parts.push(i18n.t('providers:patchMessages.closeFailures', { count: summary.acp_close_failures }))
  }
  const detail = parts.length > 0 ? `${parts.join('. ')}. ` : ''
  const warnBlock =
    summary.warnings && summary.warnings.length > 0
      ? i18n.t('providers:patchMessages.warningsPrefix', {
          warnings: `${summary.warnings.slice(0, 5).join(' | ')}${summary.warnings.length > 5 ? '…' : ''}`,
        })
      : ''
  return i18n.t('providers:patchMessages.clearSessionsSuccess', { detail, warnings: warnBlock })
}

export function antigravityClearSessionsFailMessage(): string {
  return i18n.t('providers:patchMessages.clearSessionsFail')
}

export function cursorClearSessionsSuccessMessage(namespace: string | undefined): string {
  return i18n.t('providers:patchMessages.clearCursorSessionsSuccess', {
    namespace: namespace ?? 'cursor-usage',
  })
}

export function cursorClearSessionsFailMessage(): string {
  return i18n.t('providers:patchMessages.clearCursorSessionsFail')
}

export function webLoginDoneProbeFailMessage(sessionId: string): string {
  return i18n.t('providers:patchMessages.webLoginDoneProbeFail', { sessionId })
}

export function webLoginSessionStatusMessage(status: string, expiresAt: string): string {
  return i18n.t('providers:patchMessages.webLoginSessionStatus', { status, expiresAt })
}

export function noActiveCursorSessionMessage(): string {
  return i18n.t('providers:patchMessages.noActiveSession')
}

export function webLoginOpProbeFailMessage(): string {
  return i18n.t('providers:patchMessages.webLoginProbeFail')
}

export function webLoginOpStatusMessage(status: string, accountId: string, expiresAt: string): string {
  return i18n.t('providers:patchMessages.webLoginOpStatus', {
    status,
    accountId: accountId || '—',
    expiresAt,
  })
}

export function proxyStatusProbeFailMessage(): string {
  return i18n.t('providers:patchMessages.statusProbeFail')
}

export function gatewayKeyRequiredBeforeWebLoginMessage(): string {
  return i18n.t('providers:patchMessages.gatewayKeyRequired')
}

export function antigravityOAuthStatusMessage(params: {
  authenticated: boolean
  account?: string
  viabilityGate?: string
  operationStatus?: string
  operationId?: string
  lastCheckedAt: string
}): string {
  const account =
    params.account != null && params.account !== ''
      ? i18n.t('providers:patchMessages.oauthAccount', { account: params.account })
      : ''
  const gate = params.viabilityGate
    ? i18n.t('providers:patchMessages.oauthGate', { gate: params.viabilityGate })
    : ''
  const op = params.operationStatus
    ? i18n.t('providers:patchMessages.oauthOp', {
        status: params.operationStatus,
        id: params.operationId ? ` (${params.operationId})` : '',
      })
    : ''
  return i18n.t('providers:patchMessages.oauthStatus', {
    auth: params.authenticated
      ? i18n.t('providers:patchMessages.oauthAuthenticated')
      : i18n.t('providers:patchMessages.oauthNotAuthenticated'),
    account,
    gate,
    op,
    checkedAt: params.lastCheckedAt,
  })
}

export function antigravityOAuthStatusFailMessage(): string {
  return i18n.t('providers:patchMessages.oauthStatusFail')
}

export function antigravityOAuthPopupBlockedMessage(url: string): string {
  return i18n.t('providers:patchMessages.oauthPopupBlocked', { url })
}

export function antigravityOAuthOpenedMessage(): string {
  return i18n.t('providers:patchMessages.oauthOpened')
}

export function antigravityOAuthStartFailMessage(): string {
  return i18n.t('providers:patchMessages.oauthStartFail')
}

export function testConnectionSuccessMessage(isAntigravity: boolean): string {
  return isAntigravity
    ? i18n.t('providers:patchMessages.testAntigravitySuccess')
    : i18n.t('providers:patchMessages.testConnectionSuccess')
}

export function testConnectionFailMessage(): string {
  return i18n.t('providers:patchMessages.testConnectionFail')
}

export function preSaveTestFailMessage(apiMessage?: string): string {
  if (apiMessage) {
    return i18n.t('providers:patchMessages.preSaveTestFailPrefix', { message: apiMessage })
  }
  return i18n.t('providers:patchMessages.preSaveTestFail')
}

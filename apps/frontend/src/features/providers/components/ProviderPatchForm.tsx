import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { AlertCircle, CheckCircle2, Lock, RefreshCw, ShieldCheck, Trash2 } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { createProviderPatchSchema } from '../lib/providerSchemas'
import {
  clearAntigravityProxyRuntimeSessions,
  clearCursorUsageProxyRuntimeSessions,
  getAntigravityProxyAuthStatus,
  getCursorProxyAuthStatus,
  getCursorProxyUsageSession,
  probeCursorProxyUsageCollect,
  providersQueryKeys,
  startAntigravityProxyAuth,
  startCursorProxyAuth,
  updateProvider,
  validateProvider,
} from '../api/queries'
import { cursorAuthStartSuccessMessage } from '../lib/cursorAuthStartMessage'
import { messageForCursorCollectReadiness } from '../lib/cursorCollectReadiness'
import { cursorProxyAuthErrorMessage } from '../lib/mutationErrors'
import {
  antigravityClearSessionsFailMessage,
  antigravityClearSessionsSuccessMessage,
  antigravityOAuthOpenedMessage,
  antigravityOAuthPopupBlockedMessage,
  antigravityOAuthStartFailMessage,
  antigravityOAuthStatusFailMessage,
  antigravityOAuthStatusMessage,
  cursorClearSessionsFailMessage,
  cursorClearSessionsSuccessMessage,
  gatewayKeyRequiredBeforeWebLoginMessage,
  noActiveCursorSessionMessage,
  preSaveTestFailMessage,
  proxyStatusProbeFailMessage,
  testConnectionFailMessage,
  testConnectionSuccessMessage,
  webLoginDoneProbeFailMessage,
  webLoginOpProbeFailMessage,
  webLoginOpStatusMessage,
  webLoginSessionStatusMessage,
} from '../lib/providerPatchMessages'
import type { AdminProvider } from '../types'

type PatchFormValues = {
  name?: string
  endpoint: string
  credential?: string
  management_credential?: string
  oauth_redirect_uri?: string
}

type ProviderPatchFormProps = {
  provider: AdminProvider
  isPending: boolean
  errorMessage?: string
  dangerAction?: ReactNode
  onSubmit: (values: { endpoint?: string; credential?: string; management_credential?: string }) => void
}

export function ProviderPatchForm({ provider, isPending, errorMessage, dangerAction, onSubmit }: ProviderPatchFormProps) {
  const { t, i18n } = useTranslation('providers')
  const patchSchema = useMemo(() => createProviderPatchSchema(), [i18n.language])
  const queryClient = useQueryClient()
  const [testingConnection, setTestingConnection] = useState(false)
  const [clearingProxySessions, setClearingProxySessions] = useState(false)
  const [loadingAuthStatus, setLoadingAuthStatus] = useState(false)
  const [startingOAuth, setStartingOAuth] = useState(false)
  const [clearingCursorUsage, setClearingCursorUsage] = useState(false)
  const [startingCursorAuth, setStartingCursorAuth] = useState(false)
  const [loadingCursorAuthStatus, setLoadingCursorAuthStatus] = useState(false)
  const [activeCursorSessionId, setActiveCursorSessionId] = useState<string | null>(null)
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null)
  const cursorPollStopRef = useRef(false)

  async function handleClearAntigravitySessions() {
    setClearingProxySessions(true)
    setTestResult(null)
    try {
      const summary = await clearAntigravityProxyRuntimeSessions(provider.id)
      setTestResult({
        success: true,
        message: antigravityClearSessionsSuccessMessage(summary),
      })
    } catch (err) {
      setTestResult({
        success: false,
        message: err instanceof Error ? err.message : antigravityClearSessionsFailMessage(),
      })
    } finally {
      setClearingProxySessions(false)
    }
  }

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<PatchFormValues>({
    resolver: zodResolver(patchSchema),
    defaultValues: {
      name: provider.name,
      endpoint: provider.endpoint,
      credential: '',
      management_credential: '',
      oauth_redirect_uri: '',
    },
  })

  const credentialValue = watch('credential')
  const managementValue = watch('management_credential')
  const oauthRedirectValue = watch('oauth_redirect_uri')
  const isAntigravity = provider.type === 'antigravity'
  const isCursor = provider.type === 'cursor'

  async function evaluateCursorSessionForCollect() {
    const persisted = await getCursorProxyUsageSession(provider.id)
    if (persisted.status !== 'authenticated') {
      return messageForCursorCollectReadiness(persisted, {
        status: 'authentication_required',
        accountId: provider.id,
      })
    }
    const collect = await probeCursorProxyUsageCollect(provider.id)
    return messageForCursorCollectReadiness(persisted, collect)
  }

  useEffect(() => {
    if (!isCursor || activeCursorSessionId === null) {
      return undefined
    }
    cursorPollStopRef.current = false
    const sessionId = activeCursorSessionId
    const poll = async (): Promise<void> => {
      if (cursorPollStopRef.current) {
        return
      }
      try {
        const s = await getCursorProxyAuthStatus(provider.id, sessionId)
        if (s.status === 'authenticated') {
          cursorPollStopRef.current = true
          void queryClient.invalidateQueries({ queryKey: providersQueryKeys.quotas })
          try {
            setTestResult(await evaluateCursorSessionForCollect())
          } catch {
            setTestResult({
              success: false,
              message: webLoginDoneProbeFailMessage(s.sessionId),
            })
          }
          return
        }
        if (s.status === 'expired' || s.status === 'authentication_failed') {
          cursorPollStopRef.current = true
          setTestResult({
            success: false,
            message: webLoginSessionStatusMessage(s.status, s.expiresAt),
          })
        }
      } catch {
        // keep polling until user checks manually or session ends
      }
    }
    void poll()
    const handle = window.setInterval(() => void poll(), 3000)
    return () => {
      cursorPollStopRef.current = true
      window.clearInterval(handle)
    }
  }, [isCursor, activeCursorSessionId, provider.id])

  async function handleClearCursorUsageSessions() {
    setClearingCursorUsage(true)
    setTestResult(null)
    cursorPollStopRef.current = true
    setActiveCursorSessionId(null)
    try {
      const summary = await clearCursorUsageProxyRuntimeSessions(provider.id)
      setTestResult({
        success: true,
        message: cursorClearSessionsSuccessMessage(summary.namespace),
      })
    } catch (err) {
      setTestResult({
        success: false,
        message: err instanceof Error ? err.message : cursorClearSessionsFailMessage(),
      })
    } finally {
      setClearingCursorUsage(false)
    }
  }

  async function handleCursorAuthStatus() {
    setLoadingCursorAuthStatus(true)
    setTestResult(null)
    try {
      const readiness = await evaluateCursorSessionForCollect()
      if (readiness.success) {
        void queryClient.invalidateQueries({ queryKey: providersQueryKeys.quotas })
        setTestResult(readiness)
        return
      }
      if (readiness.code === 'no_persisted_session') {
        // fall through to active login operation status
      } else {
        setTestResult(readiness)
        return
      }
      if (!activeCursorSessionId) {
        setTestResult({
          success: false,
          message: noActiveCursorSessionMessage(),
        })
        return
      }
      const s = await getCursorProxyAuthStatus(provider.id, activeCursorSessionId)
      if (s.status === 'authenticated') {
        cursorPollStopRef.current = true
        try {
          setTestResult(await evaluateCursorSessionForCollect())
        } catch {
          setTestResult({
            success: false,
            message: webLoginOpProbeFailMessage(),
          })
        }
      } else {
        setTestResult({
          success: false,
          message: webLoginOpStatusMessage(s.status, s.accountId, s.expiresAt),
        })
      }
    } catch (err) {
      setTestResult({
        success: false,
        message: err instanceof Error ? err.message : proxyStatusProbeFailMessage(),
      })
    } finally {
      setLoadingCursorAuthStatus(false)
    }
  }

  async function persistCursorGatewayIfNeeded(): Promise<boolean> {
    if (provider.management_credential_configured) {
      return true
    }
    const mgmt = managementValue?.trim() ?? ''
    if (mgmt.length === 0) {
      return false
    }
    await validateProvider(provider.id, { management_credential: mgmt })
    await updateProvider(provider.id, { management_credential: mgmt })
    await queryClient.invalidateQueries({ queryKey: providersQueryKeys.list })
    await queryClient.invalidateQueries({ queryKey: providersQueryKeys.detail(provider.id) })
    return true
  }

  async function handleCursorAuthStart() {
    setStartingCursorAuth(true)
    setTestResult(null)
    try {
      const gatewayReady = await persistCursorGatewayIfNeeded()
      if (!gatewayReady) {
        setTestResult({
          success: false,
          message: gatewayKeyRequiredBeforeWebLoginMessage(),
        })
        return
      }
      const res = await startCursorProxyAuth(provider.id)
      setActiveCursorSessionId(res.sessionId)
      cursorPollStopRef.current = false
      setTestResult({
        success: true,
        message: cursorAuthStartSuccessMessage(res),
      })
    } catch (err) {
      setTestResult({
        success: false,
        message: cursorProxyAuthErrorMessage(err),
      })
    } finally {
      setStartingCursorAuth(false)
    }
  }

  async function handleAntigravityAuthStatus() {
    setLoadingAuthStatus(true)
    setTestResult(null)
    try {
      const s = await getAntigravityProxyAuthStatus(provider.id)
      setTestResult({
        success: true,
        message: antigravityOAuthStatusMessage({
          authenticated: s.authenticated,
          account: s.account ?? undefined,
          viabilityGate: s.viabilityGate,
          operationStatus: s.activeOperation?.status,
          operationId: s.activeOperation?.operationId,
          lastCheckedAt: s.lastCheckedAt,
        }),
      })
    } catch (err) {
      setTestResult({
        success: false,
        message: err instanceof Error ? err.message : antigravityOAuthStatusFailMessage(),
      })
    } finally {
      setLoadingAuthStatus(false)
    }
  }

  async function handleAntigravityAuthStart() {
    setStartingOAuth(true)
    setTestResult(null)
    try {
      const redirect =
        oauthRedirectValue && oauthRedirectValue.trim().length > 0
          ? oauthRedirectValue.trim()
          : undefined
      const res = await startAntigravityProxyAuth(
        provider.id,
        redirect ? { redirect_uri: redirect } : undefined,
      )
      const opened = window.open(res.auth_url, '_blank', 'noopener,noreferrer')
      if (!opened) {
        setTestResult({
          success: true,
          message: antigravityOAuthPopupBlockedMessage(res.auth_url),
        })
        return
      }
      setTestResult({
        success: true,
        message: antigravityOAuthOpenedMessage(),
      })
    } catch (err) {
      setTestResult({
        success: false,
        message: err instanceof Error ? err.message : antigravityOAuthStartFailMessage(),
      })
    } finally {
      setStartingOAuth(false)
    }
  }

  async function handleTestConnection() {
    setTestingConnection(true)
    setTestResult(null)
    try {
      const cred = credentialValue && credentialValue.trim().length > 0 ? credentialValue.trim() : undefined
      const mgmt =
        managementValue && managementValue.trim().length > 0 ? managementValue.trim() : undefined
      if (isAntigravity) {
        await validateProvider(
          provider.id,
          mgmt ? { management_credential: mgmt } : undefined,
        )
      } else {
        await validateProvider(
          provider.id,
          cred ? { credential: cred } : undefined,
        )
      }
      setTestResult({
        success: true,
        message: testConnectionSuccessMessage(isAntigravity),
      })
    } catch (err) {
      setTestResult({
        success: false,
        message: err instanceof Error ? err.message : testConnectionFailMessage(),
      })
    } finally {
      setTestingConnection(false)
    }
  }

  const isBusy =
    isPending ||
    testingConnection ||
    clearingProxySessions ||
    loadingAuthStatus ||
    startingOAuth ||
    clearingCursorUsage ||
    loadingCursorAuthStatus ||
    startingCursorAuth

  const hasManagementKey = provider.type === 'openrouter' || isAntigravity || isCursor

  return (
    <form
      className="provider-patch-form min-w-0 max-w-full space-y-2"
      onSubmit={handleSubmit(async (values) => {
        setTestResult(null)
        const cred = values.credential && values.credential.trim().length > 0 ? values.credential.trim() : undefined
        const mgmtIncoming =
          values.management_credential && values.management_credential.trim().length > 0
            ? values.management_credential.trim()
            : undefined

        // Always perform connection test before saving when a new credential is provided
        const shouldPreTestMgmt = (isAntigravity || isCursor) && mgmtIncoming
        const shouldPreTestCred = !isAntigravity && cred
        if (shouldPreTestMgmt || shouldPreTestCred) {
          setTestingConnection(true)
          try {
            if (isAntigravity && mgmtIncoming) {
              await validateProvider(provider.id, { management_credential: mgmtIncoming })
            } else {
              if (shouldPreTestCred) {
                await validateProvider(provider.id, { credential: cred! })
              }
              if (isCursor && mgmtIncoming) {
                await validateProvider(provider.id, { management_credential: mgmtIncoming })
              }
            }
          } catch (err) {
            setTestingConnection(false)
            setTestResult({
              success: false,
              message: preSaveTestFailMessage(err instanceof Error ? err.message : undefined),
            })
            return
          }
          setTestingConnection(false)
        }

        const payload: { endpoint?: string; credential?: string; management_credential?: string } = {}
        if (values.endpoint !== provider.endpoint) {
          payload.endpoint = values.endpoint
        }
        if (!isAntigravity && cred) {
          payload.credential = cred
        }
        if (provider.type === 'openrouter' && values.management_credential && values.management_credential.length > 0) {
          payload.management_credential = values.management_credential
        }
        if ((isAntigravity || isCursor) && mgmtIncoming) {
          payload.management_credential = mgmtIncoming
        }
        onSubmit(payload)
      })}
    >
      <div className="grid grid-cols-1 gap-x-3 gap-y-2 sm:grid-cols-2">
        <div className={hasManagementKey ? '' : 'sm:col-span-2'}>
          <div className="mb-0.5 flex items-center justify-between gap-2">
            <label htmlFor="provider-name-input" className="text-[0.6875rem] font-medium text-secondary">
              {t('patchForm.nameLabel')}
            </label>
            <span className="flex items-center gap-1 rounded bg-[#0D121B] px-1.5 py-0.5 text-[0.625rem] text-muted border border-line/60">
              <Lock className="size-2.5 text-cobalt" />
              {t('patchForm.nameFixed')}
            </span>
          </div>
          <input
            id="provider-name-input"
            type="text"
            readOnly
            disabled
            aria-readonly="true"
            className="w-full rounded-md border border-line/40 bg-[#0D121B]/80 px-2 py-1 text-[0.6875rem] text-muted cursor-not-allowed outline-none select-none"
            {...register('name')}
          />
        </div>

        {hasManagementKey ? (
          <div>
            <label htmlFor="provider-management-cred" className="mb-0.5 block text-[0.6875rem] font-medium text-secondary">
              {isAntigravity || isCursor
                ? t('patchForm.gatewayKey')
                : t('patchForm.managementKeyOpenRouter')}
            </label>
            <input
              id="provider-management-cred"
              type="password"
              autoComplete="new-password"
              placeholder={
                isAntigravity || isCursor
                  ? provider.management_credential_configured
                    ? t('patchForm.placeholderConfiguredReplace')
                    : t('patchForm.placeholderGatewayKey')
                  : provider.management_credential_configured
                    ? t('patchForm.placeholderConfiguredReplace')
                    : t('patchForm.placeholderNotConfigured')
              }
              className="w-full rounded-md border border-line bg-[#09090b] px-2 py-1 text-[0.6875rem] text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20"
              {...register('management_credential')}
            />
            {errors.management_credential ? (
              <span className="mt-1 block text-xs text-failure">{errors.management_credential.message}</span>
            ) : null}
          </div>
        ) : null}

        <div>
          <div className="mb-0.5 flex flex-wrap items-center justify-between gap-x-2 gap-y-0.5">
            <label htmlFor="provider-endpoint-input" className="text-[0.6875rem] font-medium text-secondary">
              {t('patchForm.endpointLabel')}
            </label>
            {isAntigravity ? (
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleTestConnection()}
                className="inline-flex shrink-0 items-center gap-1 text-[0.6875rem] font-medium text-cobalt hover:text-[#7DA4FF] transition-colors disabled:opacity-50 cursor-pointer"
              >
                <ShieldCheck className="size-3" />
                {t('patchForm.testConnection')}
              </button>
            ) : null}
          </div>
          <input
            id="provider-endpoint-input"
            type="url"
            autoComplete="off"
            disabled={isBusy}
            className="w-full rounded-md border border-line bg-[#09090b] px-2 py-1 text-[0.6875rem] text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20 disabled:opacity-50"
            {...register('endpoint')}
          />
          {errors.endpoint ? <span className="mt-1 block text-xs text-failure">{errors.endpoint.message}</span> : null}
        </div>

        {isAntigravity ? (
          <div>
            <label htmlFor="antigravity-oauth-redirect" className="mb-0.5 block text-[0.6875rem] font-medium text-secondary">
              {t('patchForm.oauthRedirectOptional')}
            </label>
            <input
              id="antigravity-oauth-redirect"
              type="url"
              autoComplete="off"
              disabled={isBusy}
              placeholder={t('patchForm.oauthRedirectPlaceholder')}
              className="w-full rounded-md border border-line bg-[#09090b] px-2 py-1 text-[0.6875rem] text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20 disabled:opacity-50"
              {...register('oauth_redirect_uri')}
            />
            {errors.oauth_redirect_uri ? (
              <span className="mt-1 block text-xs text-failure">{errors.oauth_redirect_uri.message}</span>
            ) : null}
          </div>
        ) : (
          <div>
            <div className="mb-0.5 flex items-center justify-between">
              <label htmlFor="provider-credential-patch" className="text-[0.6875rem] font-medium text-secondary">
                {t('patchForm.credentialLabel')}
              </label>
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleTestConnection()}
                className="inline-flex items-center gap-1 text-[0.6875rem] font-medium text-cobalt hover:text-[#7DA4FF] transition-colors disabled:opacity-50 cursor-pointer"
              >
                <ShieldCheck className="size-3" />
                {t('patchForm.testConnection')}
              </button>
            </div>
            <input
              id="provider-credential-patch"
              type="password"
              autoComplete="new-password"
              disabled={isBusy}
              placeholder={
                provider.credential_configured
                  ? t('patchForm.placeholderConfiguredReplace')
                  : t('patchForm.placeholderNotConfigured')
              }
              className="w-full rounded-md border border-line bg-[#09090b] px-2 py-1 text-[0.6875rem] text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20 disabled:opacity-50"
              {...register('credential')}
            />
            {errors.credential ? (
              <span className="mt-1 block text-xs text-failure">{errors.credential.message}</span>
            ) : null}
          </div>
        )}

        {isAntigravity ? (
          <div className="sm:col-span-2">
            <div className="grid min-w-0 grid-cols-3 gap-1.5">
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleAntigravityAuthStart()}
                className="inline-flex h-8 min-h-[2rem] w-full min-w-0 flex-col items-center justify-center gap-0 rounded-md border border-cobalt/55 bg-cobalt/18 px-1 py-0.5 text-center text-[0.5625rem] font-semibold leading-tight text-[#B8D4FF] shadow-sm transition-colors [text-wrap:balance] hover:border-cobalt/70 hover:bg-cobalt/25 hover:text-[#D4E6FF] disabled:cursor-not-allowed disabled:opacity-50 sm:flex-row sm:gap-1 sm:px-2 sm:text-[0.625rem]"
              >
                {startingOAuth ? (
                  <>
                    <RefreshCw className="size-3 shrink-0 animate-spin" />
                    <span>{t('patchForm.starting')}</span>
                  </>
                ) : (
                  <span>{t('patchForm.oauthLogin')}</span>
                )}
              </button>
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleAntigravityAuthStatus()}
                className="inline-flex h-8 min-h-[2rem] w-full min-w-0 flex-col items-center justify-center gap-0 rounded-md border border-line/55 bg-[#0D121B] px-1 py-0.5 text-center text-[0.5625rem] font-medium leading-tight text-secondary shadow-sm transition-colors [text-wrap:balance] hover:border-cobalt/35 hover:text-primary disabled:cursor-not-allowed disabled:opacity-50 sm:flex-row sm:gap-1 sm:px-2 sm:text-[0.625rem]"
              >
                {loadingAuthStatus ? (
                  <>
                    <RefreshCw className="size-3 shrink-0 animate-spin" />
                    <span>{t('patchForm.checking')}</span>
                  </>
                ) : (
                  <span>{t('patchForm.oauthStatus')}</span>
                )}
              </button>
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleClearAntigravitySessions()}
                className="inline-flex h-8 min-h-[2rem] w-full min-w-0 flex-col items-center justify-center gap-0 rounded-md border border-line/50 bg-[#141a24] px-1 py-0.5 text-center text-[0.5625rem] font-medium leading-tight text-amber-200/90 shadow-sm transition-colors [text-wrap:balance] hover:border-amber-500/25 hover:bg-amber-500/5 disabled:cursor-not-allowed disabled:opacity-50 sm:flex-row sm:gap-1 sm:px-2 sm:text-[0.625rem]"
              >
                {clearingProxySessions ? (
                  <>
                    <RefreshCw className="size-3 shrink-0 animate-spin" />
                    <span>{t('patchForm.clearing')}</span>
                  </>
                ) : (
                  <>
                    <Trash2 className="size-3 shrink-0 text-amber-300/80" />
                    <span>{t('patchForm.endSession')}</span>
                  </>
                )}
              </button>
            </div>
            <details className="group mt-1 w-full min-w-0 rounded border border-line/40 bg-[#0D121B]/40 px-2 py-1 text-[0.625rem] text-muted">
              <summary className="cursor-pointer list-none font-medium text-secondary marker:content-none [&::-webkit-details-marker]:hidden">
                {t('patchForm.antigravitySessionSummaryTitle')}
              </summary>
              <ul className="mt-1 list-disc space-y-0.5 pl-3.5 leading-snug marker:text-muted">
                <li>{t('patchForm.antigravitySessionBullet1')}</li>
                <li>{t('patchForm.antigravitySessionBullet2')}</li>
                <li>{t('patchForm.antigravitySessionBullet3')}</li>
              </ul>
            </details>
          </div>
        ) : null}

        {isCursor ? (
          <div className="sm:col-span-2">
            <div className="grid min-w-0 grid-cols-3 gap-1.5">
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleCursorAuthStart()}
                className="inline-flex h-8 min-h-[2rem] w-full min-w-0 flex-col items-center justify-center gap-0 rounded-md border border-cobalt/55 bg-cobalt/18 px-1 py-0.5 text-center text-[0.5625rem] font-semibold leading-tight text-[#B8D4FF] shadow-sm transition-colors [text-wrap:balance] hover:border-cobalt/70 hover:bg-cobalt/25 hover:text-[#D4E6FF] disabled:cursor-not-allowed disabled:opacity-50 sm:flex-row sm:gap-1 sm:px-2 sm:text-[0.625rem]"
              >
                {startingCursorAuth ? (
                  <>
                    <RefreshCw className="size-3 shrink-0 animate-spin" />
                    <span>{t('patchForm.starting')}</span>
                  </>
                ) : (
                  <span>{t('patchForm.webLogin')}</span>
                )}
              </button>
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleCursorAuthStatus()}
                className="inline-flex h-8 min-h-[2rem] w-full min-w-0 flex-col items-center justify-center gap-0 rounded-md border border-line/55 bg-[#0D121B] px-1 py-0.5 text-center text-[0.5625rem] font-medium leading-tight text-secondary shadow-sm transition-colors [text-wrap:balance] hover:border-cobalt/35 hover:text-primary disabled:cursor-not-allowed disabled:opacity-50 sm:flex-row sm:gap-1 sm:px-2 sm:text-[0.625rem]"
              >
                {loadingCursorAuthStatus ? (
                  <>
                    <RefreshCw className="size-3 shrink-0 animate-spin" />
                    <span>{t('patchForm.checking')}</span>
                  </>
                ) : (
                  <span>{t('patchForm.checkStatus')}</span>
                )}
              </button>
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleClearCursorUsageSessions()}
                className="inline-flex h-8 min-h-[2rem] w-full min-w-0 flex-col items-center justify-center gap-0 rounded-md border border-line/50 bg-[#141a24] px-1 py-0.5 text-center text-[0.5625rem] font-medium leading-tight text-amber-200/90 shadow-sm transition-colors [text-wrap:balance] hover:border-amber-500/25 hover:bg-amber-500/5 disabled:cursor-not-allowed disabled:opacity-50 sm:flex-row sm:gap-1 sm:px-2 sm:text-[0.625rem]"
              >
                {clearingCursorUsage ? (
                  <>
                    <RefreshCw className="size-3 shrink-0 animate-spin" />
                    <span>{t('patchForm.clearing')}</span>
                  </>
                ) : (
                  <>
                    <Trash2 className="size-3 shrink-0 text-amber-300/80" />
                    <span>{t('patchForm.clearSession')}</span>
                  </>
                )}
              </button>
            </div>
            <details className="group mt-1 w-full min-w-0 rounded border border-line/40 bg-[#0D121B]/40 px-2 py-1 text-[0.625rem] text-muted">
              <summary className="cursor-pointer list-none font-medium text-secondary marker:content-none [&::-webkit-details-marker]:hidden">
                {t('patchForm.cursorSessionSummaryTitle')}
              </summary>
              <ul className="mt-1 list-disc space-y-0.5 pl-3.5 leading-snug marker:text-muted">
                <li>{t('patchForm.cursorSessionBullet1')}</li>
                <li>{t('patchForm.cursorSessionBullet2')}</li>
                <li>{t('patchForm.cursorSessionBullet3')}</li>
              </ul>
            </details>
          </div>
        ) : null}
      </div>

      {testResult ? (
        <div
          className={`flex max-h-24 min-w-0 max-w-full items-start gap-1.5 overflow-y-auto rounded-md border px-2 py-1.5 text-[0.625rem] leading-snug ${
            testResult.success
              ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-200'
              : 'border-red-500/30 bg-red-500/10 text-red-200'
          }`}
        >
          {testResult.success ? (
            <CheckCircle2 className="size-3.5 shrink-0 text-emerald-400 mt-px" />
          ) : (
            <AlertCircle className="size-3.5 shrink-0 text-red-400 mt-px" />
          )}
          <p
            className="min-w-0 flex-1 [overflow-wrap:anywhere] break-words"
            title={testResult.message}
          >
            {testResult.message}
          </p>
        </div>
      ) : null}

      {errorMessage ? (
        <div className="flex max-h-24 min-w-0 max-w-full items-start gap-1.5 overflow-y-auto rounded-md border border-red-500/30 bg-red-500/10 px-2 py-1.5 text-[0.625rem] leading-snug text-red-200">
          <AlertCircle className="size-3.5 shrink-0 text-red-400 mt-px" />
          <p className="min-w-0 flex-1 [overflow-wrap:anywhere] break-words" title={errorMessage}>
            {errorMessage}
          </p>
        </div>
      ) : null}

      <div className="flex flex-col-reverse gap-2 border-t border-line pt-2 sm:flex-row sm:items-center sm:justify-between">
        {dangerAction ? <div className="min-w-0">{dangerAction}</div> : <div />}
        <button
          type="submit"
          disabled={isBusy}
          aria-label={t('patchForm.saveAria')}
          className="flex h-8 w-full shrink-0 items-center justify-center gap-1.5 rounded-md border border-cobalt/40 bg-accent-deep px-4 text-[0.6875rem] font-semibold text-[#E4EBFC] shadow-sm hover:bg-[#314A78] transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 sm:w-auto sm:min-w-[11rem]"
        >
          {isBusy ? (
            <>
              <RefreshCw className="size-3.5 animate-spin" />
              <span>
                {testingConnection ? t('patchForm.testingConnection') : t('patchForm.saving')}
              </span>
            </>
          ) : (
            <>
              <ShieldCheck className="size-3.5 text-cobalt" />
              <span>{t('patchForm.save')}</span>
            </>
          )}
        </button>
      </div>
    </form>
  )
}

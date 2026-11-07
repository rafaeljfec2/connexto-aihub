import { useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { AlertCircle, CheckCircle2, Lock, RefreshCw, ShieldCheck, Trash2 } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import {
  clearAntigravityProxyRuntimeSessions,
  getAntigravityProxyAuthStatus,
  startAntigravityProxyAuth,
  validateProvider,
} from '../api/queries'
import type { AdminProvider } from '../types'

const patchSchema = z.object({
  name: z.string().optional(),
  endpoint: z.string().url('Endpoint must be a valid URL'),
  credential: z.string().optional(),
  management_credential: z.string().optional(),
  oauth_redirect_uri: z
    .string()
    .url('OAuth redirect URI must be a valid URL')
    .optional()
    .or(z.literal('')),
})

type PatchFormValues = z.infer<typeof patchSchema>

type ProviderPatchFormProps = {
  provider: AdminProvider
  isPending: boolean
  errorMessage?: string
  onSubmit: (values: { endpoint?: string; credential?: string; management_credential?: string }) => void
}

export function ProviderPatchForm({ provider, isPending, errorMessage, onSubmit }: ProviderPatchFormProps) {
  const [testingConnection, setTestingConnection] = useState(false)
  const [clearingProxySessions, setClearingProxySessions] = useState(false)
  const [loadingAuthStatus, setLoadingAuthStatus] = useState(false)
  const [startingOAuth, setStartingOAuth] = useState(false)
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null)

  async function handleClearAntigravitySessions() {
    setClearingProxySessions(true)
    setTestResult(null)
    try {
      const summary = await clearAntigravityProxyRuntimeSessions(provider.id)
      const parts: string[] = []
      if (summary.mapped_acp_ids_count !== undefined) {
        parts.push(`Mapeados no gateway: ${summary.mapped_acp_ids_count}`)
      }
      if (summary.listed_acp_sessions_count !== undefined) {
        parts.push(`Listados no ACP: ${summary.listed_acp_sessions_count}`)
      }
      if (summary.acp_sessions_closed !== undefined) {
        parts.push(`Fechados (session/close): ${summary.acp_sessions_closed}`)
      }
      if (summary.acp_close_failures !== undefined && summary.acp_close_failures > 0) {
        parts.push(`Falhas ao fechar: ${summary.acp_close_failures}`)
      }
      const detail = parts.length > 0 ? `${parts.join('. ')}. ` : ''
      const warnBlock =
        summary.warnings && summary.warnings.length > 0
          ? ` Avisos: ${summary.warnings.slice(0, 5).join(' | ')}${summary.warnings.length > 5 ? '…' : ''}`
          : ''
      setTestResult({
        success: true,
        message: `${detail}Sessões ACP e cache do gateway foram limpos. «Ver status OAuth» pode continuar «autenticado» enquanto o host do proxy tiver tokens Google em disco (ex.: ~/.gemini) — use revogação/remoção no servidor do proxy para encerrar o OAuth de verdade.${warnBlock}`,
      })
    } catch (err) {
      setTestResult({
        success: false,
        message:
          err instanceof Error
            ? err.message
            : 'Falha ao limpar sessões no proxy. Verifique o endpoint, a Gateway API Key e se o proxy está acessível.',
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

  async function handleAntigravityAuthStatus() {
    setLoadingAuthStatus(true)
    setTestResult(null)
    try {
      const s = await getAntigravityProxyAuthStatus(provider.id)
      const account = s.account != null && s.account !== '' ? ` Conta: ${s.account}.` : ''
      const gate = s.viabilityGate ? ` Viabilidade: ${s.viabilityGate}.` : ''
      const op = s.activeOperation?.status
        ? ` Operação pendente: ${s.activeOperation.status}${s.activeOperation.operationId ? ` (${s.activeOperation.operationId})` : ''}.`
        : ''
      setTestResult({
        success: true,
        message: `OAuth no proxy: ${s.authenticated ? 'autenticado' : 'não autenticado'}.${account}${gate}${op} Última checagem: ${s.lastCheckedAt}.`,
      })
    } catch (err) {
      setTestResult({
        success: false,
        message:
          err instanceof Error
            ? err.message
            : 'Falha ao consultar status OAuth no proxy. Verifique endpoint e Gateway API Key.',
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
          message:
            'Fluxo OAuth iniciado. O navegador bloqueou nova aba; copie e abra o link: ' + res.auth_url,
        })
        return
      }
      setTestResult({
        success: true,
        message:
          'Fluxo OAuth aberto em nova aba. Após concluir no Google, use "Ver status OAuth" para confirmar.',
      })
    } catch (err) {
      setTestResult({
        success: false,
        message:
          err instanceof Error
            ? err.message
            : 'Falha ao iniciar OAuth no proxy. Verifique endpoint e Gateway API Key.',
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
        message: isAntigravity
          ? 'Gateway alcançável e operacional (teste de listagem de modelos). Isso não verifica se o contexto Antigravity/ACP foi encerrado.'
          : 'Conexão testada com sucesso! O provedor está acessível e operacional.',
      })
    } catch (err) {
      setTestResult({
        success: false,
        message:
          err instanceof Error
            ? err.message
            : 'Falha no teste de conexão. Verifique a credencial e o endpoint informados.',
      })
    } finally {
      setTestingConnection(false)
    }
  }

  const isBusy =
    isPending || testingConnection || clearingProxySessions || loadingAuthStatus || startingOAuth

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
        if (isAntigravity ? mgmtIncoming : cred) {
          setTestingConnection(true)
          try {
            if (isAntigravity) {
              await validateProvider(provider.id, { management_credential: mgmtIncoming! })
            } else {
              await validateProvider(provider.id, { credential: cred! })
            }
          } catch (err) {
            setTestingConnection(false)
            setTestResult({
              success: false,
              message:
                err instanceof Error
                  ? `Falha no teste de conexão: ${err.message}`
                  : 'Falha no teste de conexão com o provedor. Credencial rejeitada.',
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
        if (isAntigravity && mgmtIncoming) {
          payload.management_credential = mgmtIncoming
        }
        onSubmit(payload)
      })}
    >
      <div>
        <div className="flex items-center justify-between mb-0.5">
          <label htmlFor="provider-name-input" className="text-[11px] font-medium text-secondary">
            Name
          </label>
          <span className="flex items-center gap-1 rounded bg-[#0D121B] px-1.5 py-0.5 text-[10px] text-muted border border-line/60">
            <Lock className="size-2.5 text-cobalt" />
            Nome fixo (não editável)
          </span>
        </div>
        <input
          id="provider-name-input"
          type="text"
          readOnly
          disabled
          aria-readonly="true"
          className="w-full rounded-md border border-line/40 bg-[#0D121B]/80 px-2 py-1 text-[11px] text-muted cursor-not-allowed outline-none select-none"
          {...register('name')}
        />
      </div>

      {provider.type === 'openrouter' || isAntigravity ? (
        <div>
          <label htmlFor="provider-management-cred" className="mb-0.5 block text-[11px] font-medium text-secondary">
            {isAntigravity ? 'Gateway API Key (write-only)' : 'Management API Key (write-only)'}
          </label>
          <input
            id="provider-management-cred"
            type="password"
            autoComplete="new-password"
            placeholder={
              isAntigravity
                ? provider.management_credential_configured
                  ? 'Configured — enter to replace'
                  : 'Gateway API key (X-Gateway-API-Key)'
                : provider.management_credential_configured
                  ? 'Configured — enter to replace'
                  : 'Not configured'
            }
            className="w-full rounded-md border border-line bg-[#09090b] px-2 py-1 text-[11px] text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20"
            {...register('management_credential')}
          />
          {errors.management_credential ? (
            <span className="mt-1 block text-xs text-failure">{errors.management_credential.message}</span>
          ) : null}
        </div>
      ) : null}

      <div>
        <div className="mb-0.5 flex flex-wrap items-center justify-between gap-x-2 gap-y-0.5">
          <label htmlFor="provider-endpoint-input" className="text-[11px] font-medium text-secondary">
            Endpoint
          </label>
          {isAntigravity ? (
            <button
              type="button"
              disabled={isBusy}
              onClick={() => void handleTestConnection()}
              className="inline-flex shrink-0 items-center gap-1 text-[11px] font-medium text-cobalt hover:text-[#7DA4FF] transition-colors disabled:opacity-50 cursor-pointer"
            >
              <ShieldCheck className="size-3" />
              Testar conexão
            </button>
          ) : null}
        </div>
        <input
          id="provider-endpoint-input"
          type="url"
          autoComplete="off"
          disabled={isBusy}
          className="w-full rounded-md border border-line bg-[#09090b] px-2 py-1 text-[11px] text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20 disabled:opacity-50"
          {...register('endpoint')}
        />
        {errors.endpoint ? <span className="mt-1 block text-xs text-failure">{errors.endpoint.message}</span> : null}
        {isAntigravity ? (
          <div className="mt-1.5">
            <label htmlFor="antigravity-oauth-redirect" className="mb-0.5 block text-[11px] font-medium text-secondary">
              OAuth redirect URI (opcional)
            </label>
            <input
              id="antigravity-oauth-redirect"
              type="url"
              autoComplete="off"
              disabled={isBusy}
              placeholder="https://seu-proxy.example/v1/providers/antigravity/auth/callback"
              className="w-full rounded-md border border-line bg-[#09090b] px-2 py-1 text-[11px] text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20 disabled:opacity-50"
              {...register('oauth_redirect_uri')}
            />
            {errors.oauth_redirect_uri ? (
              <span className="mt-1 block text-xs text-failure">{errors.oauth_redirect_uri.message}</span>
            ) : null}
          </div>
        ) : null}
        {isAntigravity ? (
          <>
            <div className="mt-1.5 grid min-w-0 grid-cols-3 gap-1">
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleAntigravityAuthStatus()}
                className="inline-flex h-8 min-h-[2rem] w-full min-w-0 flex-col items-center justify-center gap-0 rounded-md border border-line/55 bg-[#0D121B] px-1 py-0.5 text-center text-[9px] font-medium leading-tight text-secondary shadow-sm transition-colors [text-wrap:balance] hover:border-cobalt/35 hover:text-primary disabled:cursor-not-allowed disabled:opacity-50 sm:flex-row sm:gap-0.5 sm:px-1.5 sm:text-[10px]"
              >
                {loadingAuthStatus ? (
                  <>
                    <RefreshCw className="size-3 shrink-0 animate-spin" />
                    <span>Consultando…</span>
                  </>
                ) : (
                  <span>Status OAuth</span>
                )}
              </button>
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleAntigravityAuthStart()}
                className="inline-flex h-8 min-h-[2rem] w-full min-w-0 flex-col items-center justify-center gap-0 rounded-md border border-cobalt/55 bg-cobalt/18 px-1 py-0.5 text-center text-[9px] font-semibold leading-tight text-[#B8D4FF] shadow-sm transition-colors [text-wrap:balance] hover:border-cobalt/70 hover:bg-cobalt/25 hover:text-[#D4E6FF] disabled:cursor-not-allowed disabled:opacity-50 sm:flex-row sm:gap-0.5 sm:px-1.5 sm:text-[10px]"
              >
                {startingOAuth ? (
                  <>
                    <RefreshCw className="size-3 shrink-0 animate-spin" />
                    <span>Iniciando…</span>
                  </>
                ) : (
                  <span>Login OAuth</span>
                )}
              </button>
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleClearAntigravitySessions()}
                className="inline-flex h-8 min-h-[2rem] w-full min-w-0 flex-col items-center justify-center gap-0 rounded-md border border-line/50 bg-[#141a24] px-1 py-0.5 text-center text-[9px] font-medium leading-tight text-amber-200/90 shadow-sm transition-colors [text-wrap:balance] hover:border-amber-500/25 hover:bg-amber-500/5 disabled:cursor-not-allowed disabled:opacity-50 sm:flex-row sm:gap-0.5 sm:px-1.5 sm:text-[10px]"
              >
                {clearingProxySessions ? (
                  <>
                    <RefreshCw className="size-3 shrink-0 animate-spin" />
                    <span>Limpando…</span>
                  </>
                ) : (
                  <>
                    <Trash2 className="size-3 shrink-0 text-amber-300/80" />
                    <span>Encerrar sessão</span>
                  </>
                )}
              </button>
            </div>
            <details className="group mt-1 w-full min-w-0 rounded border border-line/40 bg-[#0D121B]/40 px-2 py-1 text-[10px] text-muted">
              <summary className="cursor-pointer list-none font-medium text-secondary marker:content-none [&::-webkit-details-marker]:hidden">
                Sobre «Encerrar sessão» e OAuth
              </summary>
              <ul className="mt-1 list-disc space-y-0.5 pl-3.5 leading-snug marker:text-muted">
                <li>Limpa ACP/cache no gateway neste fluxo.</li>
                <li>Não apaga tokens OAuth em disco no host do proxy (~/.gemini).</li>
                <li>«Ver status OAuth» pode seguir autenticado até revogar ou remover tokens lá.</li>
              </ul>
            </details>
          </>
        ) : null}
      </div>

      {!isAntigravity ? (
        <div>
          <div className="mb-0.5 flex items-center justify-between">
            <label htmlFor="provider-credential-patch" className="text-[11px] font-medium text-secondary">
              Credential (write-only)
            </label>
            <button
              type="button"
              disabled={isBusy}
              onClick={() => void handleTestConnection()}
              className="inline-flex items-center gap-1 text-[11px] font-medium text-cobalt hover:text-[#7DA4FF] transition-colors disabled:opacity-50 cursor-pointer"
            >
              <ShieldCheck className="size-3" />
              Testar conexão
            </button>
          </div>
          <input
            id="provider-credential-patch"
            type="password"
            autoComplete="new-password"
            disabled={isBusy}
            placeholder={provider.credential_configured ? 'Configured — enter to replace' : 'Not configured'}
            className="w-full rounded-md border border-line bg-[#09090b] px-2 py-1 text-[11px] text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20 disabled:opacity-50"
            {...register('credential')}
          />
          {errors.credential ? (
            <span className="mt-1 block text-xs text-failure">{errors.credential.message}</span>
          ) : null}
        </div>
      ) : null}

      {testResult ? (
        <div
          className={`flex min-w-0 max-w-full items-start gap-1.5 overflow-hidden rounded-md border px-1.5 py-1 text-[10px] leading-snug ${
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
        <div className="flex min-w-0 max-w-full items-start gap-1.5 overflow-hidden rounded-md border border-red-500/30 bg-red-500/10 px-1.5 py-1 text-[10px] leading-snug text-red-200">
          <AlertCircle className="size-3.5 shrink-0 text-red-400 mt-px" />
          <p className="min-w-0 flex-1 [overflow-wrap:anywhere] break-words" title={errorMessage}>
            {errorMessage}
          </p>
        </div>
      ) : null}

      <div className="border-t border-line pt-1.5">
        <button
          type="submit"
          disabled={isBusy}
          aria-label="Save changes"
          className="flex h-9 w-full items-center justify-center gap-1.5 rounded-md border border-cobalt/40 bg-accent-deep px-3 text-[11px] font-semibold text-[#E4EBFC] shadow-sm hover:bg-[#314A78] transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isBusy ? (
            <>
              <RefreshCw className="size-3.5 animate-spin" />
              <span>{testingConnection ? 'Testando conexão...' : 'Salvando alterações...'}</span>
            </>
          ) : (
            <>
              <ShieldCheck className="size-3.5 text-cobalt" />
              <span>Salvar alterações</span>
            </>
          )}
        </button>
      </div>
    </form>
  )
}

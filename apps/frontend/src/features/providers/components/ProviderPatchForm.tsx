import { useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { AlertCircle, CheckCircle2, Lock, RefreshCw, ShieldCheck, Trash2 } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { clearAntigravityProxyRuntimeSessions, validateProvider } from '../api/queries'
import type { AdminProvider } from '../types'

const patchSchema = z.object({
  name: z.string().optional(),
  endpoint: z.string().url('Endpoint must be a valid URL'),
  credential: z.string().optional(),
  management_credential: z.string().optional(),
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
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null)

  async function handleClearAntigravitySessions() {
    setClearingProxySessions(true)
    setTestResult(null)
    try {
      await clearAntigravityProxyRuntimeSessions(provider.id)
      setTestResult({
        success: true,
        message:
          'Sessões persistidas do Antigravity foram removidas no proxy (armazenamento de sessão do gateway). OAuth em ~/.gemini não é apagado por esta ação.',
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
    },
  })

  const credentialValue = watch('credential')
  const managementValue = watch('management_credential')
  const isAntigravity = provider.type === 'antigravity'

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
        message: 'Conexão testada com sucesso! O provedor está acessível e operacional.',
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

  const isBusy = isPending || testingConnection || clearingProxySessions

  return (
    <form
      className="provider-patch-form space-y-2.5"
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
        <div className="flex items-center justify-between mb-1">
          <label htmlFor="provider-name-input" className="text-xs font-medium text-secondary">
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
          className="w-full rounded-md border border-line/40 bg-[#0D121B]/80 px-2.5 py-1.5 text-xs text-muted cursor-not-allowed outline-none select-none"
          {...register('name')}
        />
      </div>

      {provider.type === 'openrouter' || isAntigravity ? (
        <div>
          <label htmlFor="provider-management-cred" className="block text-xs font-medium text-secondary mb-1">
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
            className="w-full rounded-md border border-line bg-[#09090b] px-2.5 py-1.5 text-xs text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20"
            {...register('management_credential')}
          />
          {errors.management_credential ? (
            <span className="mt-1 block text-xs text-failure">{errors.management_credential.message}</span>
          ) : null}
        </div>
      ) : null}

      <div>
        <div className="mb-1 flex flex-wrap items-center justify-between gap-x-2 gap-y-0.5">
          <label htmlFor="provider-endpoint-input" className="text-xs font-medium text-secondary">
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
          className="w-full rounded-md border border-line bg-[#09090b] px-2.5 py-1.5 text-xs text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20 disabled:opacity-50"
          {...register('endpoint')}
        />
        {errors.endpoint ? <span className="mt-1 block text-xs text-failure">{errors.endpoint.message}</span> : null}
        {isAntigravity ? (
          <>
            <div className="mt-2 flex flex-wrap items-center justify-end gap-2">
              <button
                type="button"
                disabled={isBusy}
                onClick={() => void handleClearAntigravitySessions()}
                className="inline-flex items-center gap-1 rounded-md border border-amber-500/35 bg-amber-500/10 px-2.5 py-1 text-[11px] font-medium text-amber-200 hover:bg-amber-500/15 transition-colors disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
              >
                {clearingProxySessions ? (
                  <>
                    <RefreshCw className="size-3 animate-spin" />
                    <span>Limpando sessão…</span>
                  </>
                ) : (
                  <>
                    <Trash2 className="size-3" />
                    <span>Encerrar sessão no proxy</span>
                  </>
                )}
              </button>
            </div>
          </>
        ) : null}
      </div>

      {!isAntigravity ? (
        <div>
          <div className="flex items-center justify-between mb-1">
            <label htmlFor="provider-credential-patch" className="text-xs font-medium text-secondary">
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
            className="w-full rounded-md border border-line bg-[#09090b] px-2.5 py-1.5 text-xs text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20 disabled:opacity-50"
            {...register('credential')}
          />
          {errors.credential ? (
            <span className="mt-1 block text-xs text-failure">{errors.credential.message}</span>
          ) : null}
        </div>
      ) : null}

      {testResult ? (
        <div
          className={`flex items-start gap-2 rounded-md border p-2 text-xs ${
            testResult.success
              ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-200'
              : 'border-red-500/30 bg-red-500/10 text-red-200'
          }`}
        >
          {testResult.success ? (
            <CheckCircle2 className="size-4 shrink-0 text-emerald-400 mt-0.5" />
          ) : (
            <AlertCircle className="size-4 shrink-0 text-red-400 mt-0.5" />
          )}
          <p className="leading-relaxed">{testResult.message}</p>
        </div>
      ) : null}

      {errorMessage ? (
        <div className="flex items-start gap-2 rounded-md border border-red-500/30 bg-red-500/10 p-2 text-xs text-red-200">
          <AlertCircle className="size-4 shrink-0 text-red-400 mt-0.5" />
          <p className="leading-relaxed">{errorMessage}</p>
        </div>
      ) : null}

      <div className="border-t border-line pt-2">
        <button
          type="submit"
          disabled={isBusy}
          aria-label="Save changes"
          className="flex w-full items-center justify-center gap-2 rounded-md border border-cobalt/40 bg-accent-deep px-3 py-2 text-xs font-semibold text-[#E4EBFC] shadow-sm hover:bg-[#314A78] transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-50"
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

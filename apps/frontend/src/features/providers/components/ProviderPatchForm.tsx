import { useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { AlertCircle, CheckCircle2, Lock, RefreshCw, ShieldCheck } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { validateProvider } from '../api/queries'
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
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null)

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

  async function handleTestConnection() {
    setTestingConnection(true)
    setTestResult(null)
    try {
      const cred = credentialValue && credentialValue.trim().length > 0 ? credentialValue.trim() : undefined
      await validateProvider(provider.id, cred)
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

  const isBusy = isPending || testingConnection

  return (
    <form
      className="provider-patch-form space-y-3.5"
      onSubmit={handleSubmit(async (values) => {
        setTestResult(null)
        const cred = values.credential && values.credential.trim().length > 0 ? values.credential.trim() : undefined

        // Always perform connection test before saving when a new credential is provided
        if (cred) {
          setTestingConnection(true)
          try {
            await validateProvider(provider.id, cred)
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
        if (cred) {
          payload.credential = cred
        }
        if (provider.type === 'openrouter' && values.management_credential && values.management_credential.length > 0) {
          payload.management_credential = values.management_credential
        }
        onSubmit(payload)
      })}
    >
      <div>
        <div className="flex items-center justify-between mb-1.5">
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
          className="w-full rounded-md border border-line/40 bg-[#0D121B]/80 px-3 py-2 text-xs text-muted cursor-not-allowed outline-none select-none"
          {...register('name')}
        />
      </div>

      {provider.type === 'openrouter' ? (
        <div>
          <label htmlFor="provider-management-cred" className="block text-xs font-medium text-secondary mb-1.5">
            Management API Key (write-only)
          </label>
          <input
            id="provider-management-cred"
            type="password"
            autoComplete="new-password"
            placeholder={provider.management_credential_configured ? 'Configured — enter to replace' : 'Not configured'}
            className="w-full rounded-md border border-line bg-[#09090b] px-3 py-2 text-xs text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20"
            {...register('management_credential')}
          />
          {errors.management_credential ? (
            <span className="mt-1 block text-xs text-failure">{errors.management_credential.message}</span>
          ) : null}
        </div>
      ) : null}

      <div>
        <label htmlFor="provider-endpoint-input" className="block text-xs font-medium text-secondary mb-1.5">
          Endpoint
        </label>
        <input
          id="provider-endpoint-input"
          type="url"
          autoComplete="off"
          disabled={isBusy}
          className="w-full rounded-md border border-line bg-[#09090b] px-3 py-2 text-xs text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20 disabled:opacity-50"
          {...register('endpoint')}
        />
        {errors.endpoint ? <span className="mt-1 block text-xs text-failure">{errors.endpoint.message}</span> : null}
      </div>

      <div>
        <div className="flex items-center justify-between mb-1.5">
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
          className="w-full rounded-md border border-line bg-[#09090b] px-3 py-2 text-xs text-primary outline-none focus:border-cobalt focus:ring-1 focus:ring-cobalt/20 disabled:opacity-50"
          {...register('credential')}
        />
        {errors.credential ? (
          <span className="mt-1 block text-xs text-failure">{errors.credential.message}</span>
        ) : null}
      </div>

      {testResult ? (
        <div
          className={`flex items-start gap-2 rounded-lg border p-2.5 text-xs ${
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
        <div className="flex items-start gap-2 rounded-lg border border-red-500/30 bg-red-500/10 p-2.5 text-xs text-red-200">
          <AlertCircle className="size-4 shrink-0 text-red-400 mt-0.5" />
          <p className="leading-relaxed">{errorMessage}</p>
        </div>
      ) : null}

      <div className="pt-2 border-t border-line">
        <button
          type="submit"
          disabled={isBusy}
          aria-label="Save changes"
          className="flex w-full items-center justify-center gap-2 rounded-md border border-cobalt/40 bg-accent-deep px-4 py-2.5 text-xs font-semibold text-[#E4EBFC] shadow-sm hover:bg-[#314A78] transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-50"
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

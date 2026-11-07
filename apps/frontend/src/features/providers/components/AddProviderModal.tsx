import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import {
  AlertCircle,
  ArrowLeft,
  CheckCircle,
  Edit2,
  Eye,
  EyeOff,
  Layers,
  Lock,
  RefreshCw,
  ShieldCheck,
} from 'lucide-react'
import { AiBrandIcon, hubProviderIconSlug } from '../../../shared/branding'
import { Dialog } from '../../../shared/ui'
import {
  createProvider,
  listAvailableProviders,
  providersQueryKeys,
  syncProviderModels,
  updateProvider,
  validateProvider,
} from '../api/queries'
import type { AdminProvider, AvailableProvider, SyncModelsResponse } from '../types'

type AddProviderModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  existingProviders: AdminProvider[]
  onProviderAdded?: (provider: AdminProvider) => void
}

type ModalPhase = 'select' | 'credential' | 'validating' | 'saving' | 'syncing' | 'completed' | 'sync_failed'

export function AddProviderModal({
  open,
  onOpenChange,
  existingProviders,
  onProviderAdded,
}: AddProviderModalProps) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const [selectedProvider, setSelectedProvider] = useState<AvailableProvider | null>(null)
  const [credential, setCredential] = useState('')
  const [managementCredential, setManagementCredential] = useState('')
  const [showSecret, setShowSecret] = useState(false)
  const [showMgmtSecret, setShowMgmtSecret] = useState(false)
  const [phase, setPhase] = useState<ModalPhase>('select')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [probeSuccessMessage, setProbeSuccessMessage] = useState<string | null>(null)
  const [testingProbe, setTestingProbe] = useState(false)
  const [syncResult, setSyncResult] = useState<SyncModelsResponse | null>(null)

  const availableQuery = useQuery({
    queryKey: providersQueryKeys.available,
    queryFn: listAvailableProviders,
    enabled: open,
    staleTime: 60_000,
  })

  const availableProviders = availableQuery.data?.providers ?? []
  const configuredIds = new Set(existingProviders.map((p) => p.id))
  const isEditing = selectedProvider ? configuredIds.has(selectedProvider.id) : false

  function handleReset() {
    setSelectedProvider(null)
    setCredential('')
    setManagementCredential('')
    setShowSecret(false)
    setShowMgmtSecret(false)
    setPhase('select')
    setErrorMessage(null)
    setProbeSuccessMessage(null)
    setTestingProbe(false)
    setSyncResult(null)
  }

  function handleClose(nextOpen: boolean) {
    if (!nextOpen) {
      handleReset()
    }
    onOpenChange(nextOpen)
  }

  function handleSelectProvider(provider: AvailableProvider) {
    setSelectedProvider(provider)
    setPhase('credential')
    setErrorMessage(null)
    setProbeSuccessMessage(null)
  }

  const isGatewayOnlyProvider =
    selectedProvider?.credential_type === 'none' && !!selectedProvider?.management_credential_label
  const isCursorProvider = selectedProvider?.connection_type === 'cursor'
  const showCursorGatewayField =
    isCursorProvider && !isGatewayOnlyProvider && !!selectedProvider?.management_credential_label

  async function handleQuickTestConnection() {
    if (!selectedProvider) return
    if (isGatewayOnlyProvider) {
      if (!managementCredential.trim()) return
    } else if (!credential.trim()) {
      return
    }
    setTestingProbe(true)
    setErrorMessage(null)
    setProbeSuccessMessage(null)
    try {
      if (isGatewayOnlyProvider) {
        await validateProvider(selectedProvider.id, {
          management_credential: managementCredential.trim(),
        })
      } else {
        await validateProvider(selectedProvider.id, { credential: credential.trim() })
      }
      setProbeSuccessMessage('Conexão testada com sucesso! O provedor está acessível e a credencial respondeu.')
    } catch (err) {
      setErrorMessage(
        err instanceof Error
          ? `Falha no teste de conexão: ${err.message}`
          : 'Falha no teste de conexão. Verifique o segredo informado.',
      )
    } finally {
      setTestingProbe(false)
    }
  }

  async function handleValidateAndSave() {
    if (!selectedProvider) return
    if (isGatewayOnlyProvider) {
      if (!managementCredential.trim()) return
    } else if (!credential.trim()) {
      return
    }

    setErrorMessage(null)
    setPhase('validating')

    try {
      // 1. Validate the credential with the provider upstream probe
      if (isGatewayOnlyProvider) {
        await validateProvider(selectedProvider.id, {
          management_credential: managementCredential.trim(),
        })
      } else {
        await validateProvider(selectedProvider.id, { credential: credential.trim() })
      }

      // 2. Persist the connection with the encrypted credential
      setPhase('saving')
      let savedProvider: AdminProvider
      const mgmtPayload =
        managementCredential.trim().length > 0
          ? { management_credential: managementCredential.trim() }
          : {}
      if (isEditing) {
        savedProvider = await updateProvider(selectedProvider.id, {
          ...(isGatewayOnlyProvider
            ? { management_credential: managementCredential.trim() }
            : { credential: credential.trim(), ...mgmtPayload }),
        })
      } else {
        savedProvider = await createProvider({
          id: selectedProvider.id,
          name: selectedProvider.name,
          type: selectedProvider.connection_type,
          endpoint: selectedProvider.default_endpoint,
          ...(isGatewayOnlyProvider
            ? { management_credential: managementCredential.trim() }
            : { credential: credential.trim(), ...mgmtPayload }),
          enabled: true,
        })
      }

      onProviderAdded?.(savedProvider)

      // Invalidate queries so lists update in background
      void queryClient.invalidateQueries({ queryKey: providersQueryKeys.list })
      void queryClient.invalidateQueries({ queryKey: providersQueryKeys.quotas })
      void queryClient.invalidateQueries({ queryKey: ['catalog'] })

      // 3. Automatically synchronize models
      setPhase('syncing')
      try {
        const synced = await syncProviderModels(selectedProvider.id)
        setSyncResult(synced)
        setPhase('completed')
        void queryClient.invalidateQueries({ queryKey: ['catalog'] })
      } catch (syncErr) {
        setErrorMessage(
          syncErr instanceof Error
            ? syncErr.message
            : 'Falha temporária ao sincronizar modelos. A conexão foi salva e você pode sincronizar novamente.',
        )
        setPhase('sync_failed')
      }
    } catch (err) {
      setErrorMessage(
        err instanceof Error
          ? err.message
          : 'Falha ao validar a credencial ou conectar com o provedor. Verifique o segredo informado.',
      )
      setPhase('credential')
    }
  }

  async function handleRetrySync() {
    if (!selectedProvider) return
    setErrorMessage(null)
    setPhase('syncing')
    try {
      const synced = await syncProviderModels(selectedProvider.id)
      setSyncResult(synced)
      setPhase('completed')
      void queryClient.invalidateQueries({ queryKey: ['catalog'] })
      void queryClient.invalidateQueries({ queryKey: providersQueryKeys.list })
    } catch (syncErr) {
      setErrorMessage(
        syncErr instanceof Error
          ? syncErr.message
          : 'A sincronização de modelos ainda não respondeu. Tente novamente mais tarde.',
      )
      setPhase('sync_failed')
    }
  }

  const isBusy = phase === 'validating' || phase === 'saving' || phase === 'syncing'

  return (
    <Dialog
      open={open}
      onOpenChange={handleClose}
      title={
        phase === 'select'
          ? 'Adicionar ou configurar provedor de IA'
          : isEditing
            ? `Atualizar credencial: ${selectedProvider?.name ?? 'Provedor'}`
            : `Conectar ${selectedProvider?.name ?? 'Provedor'}`
      }
      description={
        phase === 'select'
          ? 'Selecione um dos provedores de IA suportados para conectar ou atualizar a credencial no Connexto AI-HUB.'
          : isEditing
            ? isGatewayOnlyProvider
              ? 'Informe a nova Gateway API Key (chave do proxy) para substituir a credencial de gateway salva.'
              : 'Informe sua nova chave de API ou token do provedor para substituir a credencial salva.'
            : isGatewayOnlyProvider
              ? 'Informe a Gateway API Key do Connexto Proxy (X-Gateway-API-Key). A autenticação do Antigravity permanece no runtime do proxy.'
              : 'Informe sua chave de API ou token de autenticação oficial do provedor.'
      }
    >
      <div className="space-y-4 pt-1">
        {phase === 'select' ? (
          <div className="space-y-3">
            {availableQuery.isLoading ? (
              <div className="py-8 text-center text-xs text-muted animate-pulse">
                Carregando provedores disponíveis...
              </div>
            ) : availableProviders.length === 0 ? (
              <div className="py-8 text-center text-xs text-muted">
                Nenhum provedor disponível para cadastro no momento.
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-2.5 max-h-[380px] overflow-y-auto pr-1 scrollbar-slim">
                {availableProviders.map((provider) => {
                  const isConfigured = configuredIds.has(provider.id)
                  return (
                    <button
                      key={provider.id}
                      type="button"
                      onClick={() => handleSelectProvider(provider)}
                      className="group flex items-start gap-3 rounded-lg border border-line bg-surface p-3.5 text-left transition-all hover:border-cobalt/60 hover:bg-[#1E2638] cursor-pointer"
                    >
                      <AiBrandIcon
                        slug={hubProviderIconSlug(provider.logo || provider.id)}
                        label={provider.name}
                        size={40}
                        className="providers-provider-icon size-10 rounded-md border border-line bg-[#0D121B]"
                      />

                      <div className="min-w-0 flex-1">
                        <div className="flex items-center justify-between gap-2">
                          <h4 className="text-sm font-semibold text-primary">{provider.name}</h4>
                          {isConfigured ? (
                            <span className="inline-flex items-center gap-1 rounded bg-cobalt/15 px-2 py-0.5 text-[10px] font-medium text-[#A6C4FF] border border-cobalt/30 group-hover:bg-cobalt/25">
                              <Edit2 className="size-2.5" />
                              Configurado · Editar
                            </span>
                          ) : (
                            <span className="rounded bg-emerald-500/15 px-2 py-0.5 text-[10px] font-medium text-emerald-400 border border-emerald-500/30">
                              Disponível
                            </span>
                          )}
                        </div>
                        <p className="mt-1 text-xs text-secondary leading-relaxed">
                          {provider.description}
                        </p>
                        <div className="mt-2.5 flex flex-wrap items-center gap-1.5">
                          <span className="rounded bg-[#0D121B] px-2 py-0.5 text-[10px] text-muted border border-line/60">
                            {provider.credential_type === 'none' && provider.management_credential_label
                              ? provider.management_credential_label
                              : provider.credential_label}
                          </span>
                          {provider.capabilities?.map((cap) => (
                            <span
                              key={cap}
                              className="rounded bg-[#0D121B] px-1.5 py-0.5 text-[10px] text-muted"
                            >
                              {cap}
                            </span>
                          ))}
                        </div>
                      </div>
                    </button>
                  )
                })}
              </div>
            )}
          </div>
        ) : null}

        {phase === 'credential' || isBusy ? (
          <div className="space-y-4">
            <button
              type="button"
              disabled={isBusy}
              onClick={() => {
                setPhase('select')
                setErrorMessage(null)
              }}
              className="inline-flex items-center gap-1.5 text-xs text-muted hover:text-primary transition-colors disabled:opacity-50"
            >
              <ArrowLeft className="size-3.5" />
              Escolher outro provedor
            </button>

            {errorMessage ? (
              <div className="flex items-start gap-2.5 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-xs text-red-200">
                <AlertCircle className="size-4 shrink-0 text-red-400 mt-0.5" />
                <div className="space-y-0.5">
                  <p className="font-semibold text-red-300">Falha na conexão</p>
                  <p className="text-red-200/90 leading-relaxed">{errorMessage}</p>
                </div>
              </div>
            ) : null}

            {probeSuccessMessage ? (
              <div className="flex items-start gap-2.5 rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-3 text-xs text-emerald-200">
                <CheckCircle className="size-4 shrink-0 text-emerald-400 mt-0.5" />
                <div className="space-y-0.5">
                  <p className="font-semibold text-emerald-300">Conexão verificada</p>
                  <p className="text-emerald-200/90 leading-relaxed">{probeSuccessMessage}</p>
                </div>
              </div>
            ) : null}

            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label
                  htmlFor={
                    isGatewayOnlyProvider ? 'provider-mgmt-credential-input' : 'provider-credential-input'
                  }
                  className="block text-xs font-medium text-secondary"
                >
                  {isGatewayOnlyProvider
                    ? (selectedProvider?.management_credential_label ?? 'Gateway API Key')
                    : (selectedProvider?.credential_label ?? 'API Key / Token')}
                </label>
                <button
                  type="button"
                  disabled={
                    isBusy ||
                    testingProbe ||
                    (isGatewayOnlyProvider ? !managementCredential.trim() : !credential.trim())
                  }
                  onClick={() => void handleQuickTestConnection()}
                  className="inline-flex items-center gap-1 text-[11px] font-medium text-cobalt hover:text-[#7DA4FF] transition-colors disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
                >
                  {testingProbe ? (
                    <>
                      <RefreshCw className="size-3 animate-spin text-cobalt" />
                      <span>Testando...</span>
                    </>
                  ) : (
                    <>
                      <ShieldCheck className="size-3" />
                      <span>Testar conexão</span>
                    </>
                  )}
                </button>
              </div>
              <div className="relative">
                {isGatewayOnlyProvider ? (
                  <>
                    <input
                      id="provider-mgmt-credential-input"
                      type={showMgmtSecret ? 'text' : 'password'}
                      autoComplete="off"
                      disabled={isBusy}
                      value={managementCredential}
                      onChange={(e) => setManagementCredential(e.target.value)}
                      placeholder={
                        selectedProvider?.management_credential_placeholder ??
                        'Gateway API Key do Connexto Proxy'
                      }
                      className="w-full rounded-md border border-line bg-[#09090b] px-3 py-2 pr-10 text-sm text-primary outline-none transition-colors placeholder:text-muted focus:border-cobalt focus:ring-2 focus:ring-cobalt/15 disabled:opacity-50"
                    />
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => setShowMgmtSecret((prev) => !prev)}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted hover:text-primary"
                      aria-label={showMgmtSecret ? 'Ocultar segredo' : 'Exibir segredo'}
                    >
                      {showMgmtSecret ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                    </button>
                  </>
                ) : (
                  <>
                    <input
                      id="provider-credential-input"
                      type={showSecret ? 'text' : 'password'}
                      autoComplete="off"
                      disabled={isBusy}
                      value={credential}
                      onChange={(e) => setCredential(e.target.value)}
                      placeholder={
                        selectedProvider?.credential_placeholder ?? 'Cole sua chave de autenticação'
                      }
                      className="w-full rounded-md border border-line bg-[#09090b] px-3 py-2 pr-10 text-sm text-primary outline-none transition-colors placeholder:text-muted focus:border-cobalt focus:ring-2 focus:ring-cobalt/15 disabled:opacity-50"
                    />
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => setShowSecret((prev) => !prev)}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted hover:text-primary"
                      aria-label={showSecret ? 'Ocultar segredo' : 'Exibir segredo'}
                    >
                      {showSecret ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                    </button>
                  </>
                )}
              </div>
            </div>

            {showCursorGatewayField ? (
              <div className="space-y-1.5">
                <label
                  htmlFor="provider-cursor-gateway-input"
                  className="block text-xs font-medium text-secondary"
                >
                  {selectedProvider?.management_credential_label ?? 'Gateway API Key'}{' '}
                  <span className="font-normal text-muted">(opcional)</span>
                </label>
                <div className="relative">
                  <input
                    id="provider-cursor-gateway-input"
                    type={showMgmtSecret ? 'text' : 'password'}
                    autoComplete="off"
                    disabled={isBusy}
                    value={managementCredential}
                    onChange={(e) => setManagementCredential(e.target.value)}
                    placeholder={
                      selectedProvider?.management_credential_placeholder ??
                      'Chave do gateway do Connexto Proxy (X-Gateway-API-Key)'
                    }
                    className="w-full rounded-md border border-line bg-[#09090b] px-3 py-2 pr-10 text-sm text-primary outline-none transition-colors placeholder:text-muted focus:border-cobalt focus:ring-2 focus:ring-cobalt/15 disabled:opacity-50"
                  />
                  <button
                    type="button"
                    disabled={isBusy}
                    onClick={() => setShowMgmtSecret((prev) => !prev)}
                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted hover:text-primary"
                    aria-label={showMgmtSecret ? 'Ocultar Gateway API Key' : 'Exibir Gateway API Key'}
                  >
                    {showMgmtSecret ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                  </button>
                </div>
                <p className="text-[11px] leading-relaxed text-muted">
                  Recomendado quando o endpoint for o Connexto Proxy. Após salvar a Gateway API Key,
                  use <strong className="font-medium text-secondary">Gerenciar provedor</strong> →{' '}
                  <strong className="font-medium text-secondary">Login web</strong> para autorizar a
                  coleta de uso.
                </p>
              </div>
            ) : null}

            <div className="flex items-center gap-2 rounded-md border border-line/60 bg-[#0D121B] p-2.5 text-[11px] text-muted">
              <Lock className="size-3.5 shrink-0 text-cobalt" />
              <span>
                Criptografia segura em repouso (AES-256-GCM). O segredo nunca é exposto em logs ou
                respostas de API.
              </span>
            </div>

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-line">
              <button
                type="button"
                disabled={isBusy}
                onClick={() => handleClose(false)}
                className="rounded-md border border-line px-3 py-1.5 text-xs font-medium text-secondary hover:bg-white/5 transition-colors disabled:opacity-50"
              >
                Cancelar
              </button>
              <button
                type="button"
                disabled={
                  isBusy ||
                  (isGatewayOnlyProvider ? !managementCredential.trim() : !credential.trim())
                }
                onClick={() => void handleValidateAndSave()}
                className="inline-flex items-center gap-1.5 rounded-md border border-cobalt/40 bg-accent-deep px-4 py-1.5 text-xs font-semibold text-[#E4EBFC] hover:bg-[#314A78] transition-colors disabled:cursor-not-allowed disabled:opacity-50"
              >
                {isBusy ? (
                  <>
                    <RefreshCw className="size-3.5 animate-spin" />
                    {phase === 'validating'
                      ? 'Validando...'
                      : phase === 'saving'
                        ? 'Salvando...'
                        : 'Sincronizando modelos...'}
                  </>
                ) : (
                  <>
                    <ShieldCheck className="size-3.5 text-cobalt" />
                    {isEditing ? 'Validar e atualizar' : 'Validar e adicionar'}
                  </>
                )}
              </button>
            </div>
          </div>
        ) : null}

        {phase === 'completed' ? (
          <div className="space-y-4 py-2">
            <div className="flex items-start gap-3 rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-3.5 text-xs text-emerald-200">
              <CheckCircle className="size-5 shrink-0 text-emerald-400 mt-0.5" />
              <div className="space-y-1">
                <p className="font-semibold text-emerald-300">
                  {selectedProvider?.name} {isEditing ? 'atualizado' : 'conectado'} com sucesso!
                </p>
                <p className="text-emerald-200/90 leading-relaxed">
                  {isEditing
                    ? 'A credencial foi validada, atualizada com segurança e os modelos foram sincronizados no catálogo do AI-HUB.'
                    : 'A credencial foi validada, armazenada com segurança e os modelos foram importados para o catálogo do AI-HUB.'}
                </p>
              </div>
            </div>

            <div className="rounded-lg border border-line bg-surface p-3.5 text-xs space-y-2">
              <div className="flex items-center justify-between text-secondary">
                <span className="flex items-center gap-1.5">
                  <Layers className="size-4 text-cobalt" />
                  Modelos descobertos:
                </span>
                <span className="font-semibold text-primary">
                  {syncResult?.counts.discovered ?? 0}
                </span>
              </div>
              <div className="flex items-center justify-between text-secondary">
                <span>Modelos adicionados:</span>
                <span className="font-semibold text-emerald-400">
                  {syncResult?.counts.added ?? 0}
                </span>
              </div>
              <div className="flex items-center justify-between text-secondary">
                <span>Identidade do provedor:</span>
                <span className="font-mono text-[11px] text-[#A6C4FF]">
                  {selectedProvider?.id}
                </span>
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-line">
              <button
                type="button"
                onClick={() => {
                  handleClose(false)
                  navigate('/catalog')
                }}
                className="rounded-md border border-line px-3 py-1.5 text-xs font-medium text-secondary hover:bg-white/5 transition-colors"
              >
                Ver no Catálogo
              </button>
              <button
                type="button"
                onClick={() => handleClose(false)}
                className="rounded-md border border-cobalt/40 bg-accent-deep px-4 py-1.5 text-xs font-semibold text-[#E4EBFC] hover:bg-[#314A78] transition-colors"
              >
                Concluir
              </button>
            </div>
          </div>
        ) : null}

        {phase === 'sync_failed' ? (
          <div className="space-y-4 py-2">
            <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3.5 text-xs text-amber-200">
              <AlertCircle className="size-5 shrink-0 text-amber-400 mt-0.5" />
              <div className="space-y-1">
                <p className="font-semibold text-amber-300">
                  Conexão cadastrada, sincronização pendente
                </p>
                <p className="text-amber-200/90 leading-relaxed">
                  A conexão com o provedor {selectedProvider?.name} foi criada e a credencial foi
                  salva. No entanto, a importação inicial dos modelos falhou temporariamente.
                </p>
                {errorMessage ? (
                  <p className="text-amber-300/80 font-mono text-[10px] mt-1">{errorMessage}</p>
                ) : null}
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-line">
              <button
                type="button"
                onClick={() => handleClose(false)}
                className="rounded-md border border-line px-3 py-1.5 text-xs font-medium text-secondary hover:bg-white/5 transition-colors"
              >
                Fechar
              </button>
              <button
                type="button"
                onClick={() => void handleRetrySync()}
                className="inline-flex items-center gap-1.5 rounded-md border border-cobalt/40 bg-accent-deep px-4 py-1.5 text-xs font-semibold text-[#E4EBFC] hover:bg-[#314A78] transition-colors"
              >
                <RefreshCw className="size-3.5" />
                Tentar sincronizar novamente
              </button>
            </div>
          </div>
        ) : null}
      </div>
    </Dialog>
  )
}

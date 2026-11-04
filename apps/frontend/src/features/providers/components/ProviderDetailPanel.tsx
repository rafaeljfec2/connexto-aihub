import { Panel, PanelHeader } from '../../../shared/ui'
import type { AdminProvider } from '../types'

type ProviderDetailPanelProps = {
  provider: AdminProvider | null
  onOpenManage: (provider: AdminProvider) => void
}

export function ProviderDetailPanel({ provider, onOpenManage }: ProviderDetailPanelProps) {
  if (!provider) {
    return (
      <Panel>
        <PanelHeader title="Detalhe da franquia" subtitle="SELECIONE UM PROVEDOR" />
        <p className="py-8 text-center text-xs text-muted">
          Clique em uma linha da tabela para visualizar os detalhes e gerenciar a conexão.
        </p>
      </Panel>
    )
  }

  const initials = provider.name ? provider.name.slice(0, 2).toUpperCase() : provider.id.slice(0, 2).toUpperCase()

  return (
    <Panel className="space-y-3">
      <PanelHeader title="Detalhe da franquia" subtitle="INSPEÇÃO DE GOVERNANÇA" />

      <div className="flex items-center gap-2.5 border-b border-line pb-3">
        <span className="grid size-9 place-items-center rounded-lg border border-line bg-raised font-mono text-xs font-semibold text-accent-light">
          {initials}
        </span>
        <div>
          <strong className="block text-sm font-semibold text-primary">{provider.name}</strong>
          <small className="block font-mono text-[9px] text-muted">{provider.id}</small>
        </div>
      </div>

      <div className="rounded-lg border border-cobalt/20 bg-[#171E29] p-3">
        <span className="font-mono text-[8px] uppercase tracking-[0.05em] text-muted">
          Consumo da franquia
        </span>
        <div className="my-1.5 flex items-baseline gap-1.5">
          <strong className="text-xl font-semibold text-primary">
            {provider.credential_configured ? 'Sem leitura' : 'Desconfigurado'}
          </strong>
        </div>
        <div className="my-2 h-1.5 w-full overflow-hidden rounded-full bg-[#2A3443]">
          <div
            className={`h-full rounded-full ${
              provider.credential_configured ? 'w-0 bg-[#5E6B7C]' : 'w-0 bg-failure'
            }`}
          />
        </div>
        <div className="flex justify-between font-mono text-[9px] text-muted">
          <span>Consumo: não reportado</span>
          <span>Limite: sob consulta</span>
        </div>
      </div>

      <div className="divide-y divide-line/30 text-[10px]">
        <div className="flex justify-between py-2">
          <span className="text-muted">Status operacional</span>
          <strong
            className={`font-mono ${
              provider.enabled ? 'text-healthy' : 'text-muted'
            }`}
          >
            {provider.enabled ? 'Habilitado' : 'Desabilitado'}
          </strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">Credencial</span>
          <strong
            className={`font-mono ${
              provider.credential_configured ? 'text-healthy' : 'text-warning'
            }`}
          >
            {provider.credential_configured ? 'Configurada' : 'Pendente'}
          </strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">Fonte da medição</span>
          <strong className="font-mono text-secondary">API do provedor / manual</strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">Última atualização</span>
          <strong className="font-mono text-secondary">
            {provider.updated_at
              ? new Date(provider.updated_at).toLocaleDateString('pt-BR')
              : '—'}
          </strong>
        </div>
      </div>

      <div className="rounded-md border border-cobalt/20 bg-cobalt/5 p-2 text-[9px] leading-relaxed text-secondary">
        A unidade, o ciclo e a fonte podem variar por provedor. O AI Hub apresenta apenas dados
        autenticados, sem estimar saldos artificialmente.
      </div>

      <div className="pt-1">
        <button
          type="button"
          onClick={() => onOpenManage(provider)}
          className="w-full rounded-md border border-cobalt/30 bg-accent-deep py-2 text-center text-xs font-semibold text-[#EAF0FC] transition-colors hover:bg-cobalt/30 cursor-pointer"
        >
          Gerenciar credenciais & status ↗
        </button>
      </div>
    </Panel>
  )
}


import { CircleHelp } from 'lucide-react'
import { Link } from 'react-router-dom'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { appPageClassName } from '../../../app/layouts/appShellLayout'
import { IntelligenceNoticeBanner } from '../components/IntelligenceNoticeBanner'
import { IntelligenceStatCards } from '../components/IntelligenceStatCards'
import { ContextAdequacyPanel } from '../components/ContextAdequacyPanel'
import { DecisionStackPanel } from '../components/DecisionStackPanel'
import { SignalsPanel } from '../components/SignalsPanel'
import { EvolutionTimelinePanel } from '../components/EvolutionTimelinePanel'
import { Button } from '../../../shared/ui/button'

export function IntelligencePage() {
  return (
    <div className={appPageClassName('space-y-2.5', 'py-3', 'pb-6')}>
      <AppPageHeader
        actions={
          <div className="flex items-center gap-2">
            <Button asChild variant="ghost" size="sm">
              <Link to="/intelligence/help">
                <CircleHelp className="size-3.5 text-cobalt" aria-hidden />
                Ajuda
              </Link>
            </Button>
            <span className="rounded-full border border-cobalt/25 bg-cobalt/10 px-2.5 py-0.5 font-mono text-[0.625rem] font-medium text-cobalt">
              E08 · ATIVO
            </span>
            <span className="rounded-full border border-warning/25 bg-warning/10 px-2.5 py-0.5 font-mono text-[0.625rem] font-medium text-warning">
              E10 · PLANEJADO
            </span>
          </div>
        }
      />

      <IntelligenceNoticeBanner />

      <IntelligenceStatCards />

      <div className="grid grid-cols-1 gap-2.5 lg:grid-cols-2">
        <ContextAdequacyPanel />
        <DecisionStackPanel />
      </div>

      <div className="grid grid-cols-1 gap-2.5 lg:grid-cols-2">
        <SignalsPanel />
        <EvolutionTimelinePanel />
      </div>

      <div className="flex items-center gap-2.5 rounded-[10px] border border-line bg-surface/80 px-3 py-2 text-[0.6875rem] text-secondary">
        <div className="size-1.5 shrink-0 rounded-full bg-cobalt" />
        <p>
          <strong className="font-semibold text-primary">Princípio de produto:</strong> nenhuma recomendação adaptativa deve alterar o roteamento sem evidência rastreável, avaliação e política explícita.
        </p>
      </div>
    </div>
  )
}
